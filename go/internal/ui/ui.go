package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"smartpaste/internal/app"
	"smartpaste/internal/config"
	"smartpaste/internal/i18n"
	"smartpaste/internal/platform"
	"smartpaste/internal/secret"
	"smartpaste/internal/store"
)

type UI struct {
	fyneApp      fyne.App
	controller   *app.Controller
	store        *store.Store
	settings     *config.SettingsStore
	localization *i18n.Manager
	secrets      *secret.Store
	paths        config.Paths

	panel      *Panel
	hotkey     *platform.HotkeyManager
	desktopApp desktop.App
	hotkeyErr  error

	setupWindow          fyne.Window
	setupWasPanelVisible bool
	setupOnClosed        func()

	keyConfigured atomic.Bool
	setupVisible  atomic.Bool
	ctx           context.Context
	cancel        context.CancelFunc
	stopOnce      sync.Once
}

func New(
	fyneApp fyne.App,
	controller *app.Controller,
	dataStore *store.Store,
	settings *config.SettingsStore,
	localization *i18n.Manager,
	secrets *secret.Store,
	paths config.Paths,
) *UI {
	return &UI{
		fyneApp:      fyneApp,
		controller:   controller,
		store:        dataStore,
		settings:     settings,
		localization: localization,
		secrets:      secrets,
		paths:        paths,
	}
}

func (u *UI) Run() {
	u.fyneApp.Settings().SetTheme(newHUDTheme())
	u.fyneApp.SetIcon(appIcon())

	u.panel = NewPanel(u.fyneApp, u.controller, u.store, u.settings, u.localization)
	u.desktopApp, _ = u.fyneApp.(desktop.App)

	u.controller.Callbacks = app.Callbacks{
		ShowPanel: func() { fyne.Do(u.panel.Show) },
		HidePanel: func() { fyne.Do(u.panel.Hide) },
		PanelVisible: func() bool {
			return u.panel.Visible()
		},
		Refresh: func() { fyne.Do(u.panel.Reload) },
		Status:  func(message string) { fyne.Do(func() { u.notify(message) }) },
	}

	u.ctx, u.cancel = context.WithCancel(context.Background())
	u.controller.Start(u.ctx)
	go u.watchStore()
	u.hotkey = platform.NewHotkeyManager(func() {
		u.controller.HandleHotkey()
	})

	u.installTray()
	u.fyneApp.Lifecycle().SetOnStarted(u.started)
	u.fyneApp.Lifecycle().SetOnStopped(u.stop)
	go u.watchLanguage()
	u.fyneApp.Run()
}

func (u *UI) installTray() {
	log.Printf("install tray (desktopApp=%v)", u.desktopApp != nil)
	if u.desktopApp == nil {
		log.Printf("system tray is not available on this platform")
		return
	}
	u.desktopApp.SetSystemTrayIcon(trayIcon())
	u.desktopApp.SetSystemTrayWindow(u.panel.window)
	u.rebuildMenu()
}

func (u *UI) started() {
	log.Printf("lifecycle started")
	for _, permission := range platform.Permissions() {
		log.Printf("permission %s: allowed=%v required=%v", permission.Key, permission.Allowed, permission.Required)
	}
	u.registerHotkey(string(u.settings.Get().Shortcut))
	go u.refreshKeyState()

	if !u.settings.Get().HasShownFirstLaunch {
		u.panel.Show()
		u.showWelcome()
	} else if u.needsSetup() {
		u.panel.Show()
		u.showSetup(nil)
	}

	if !platform.ClipboardSupported() {
		u.panel.Show()
		u.showError(fmt.Errorf("clipboard initialization failed"))
	}
}

func (u *UI) watchStore() {
	for range u.store.Changes() {
		fyne.Do(u.rebuildMenu)
	}
}

func (u *UI) watchLanguage() {
	for range u.localization.Changes() {
		fyne.Do(func() {
			u.panel.ApplyLanguage()
			if u.setupVisible.Load() {
				u.showSetup(u.setupOnClosed)
			}
			u.rebuildMenu()
		})
	}
}

func (u *UI) stop() {
	u.stopOnce.Do(func() {
		if u.cancel != nil {
			u.cancel()
		}
		if u.hotkey != nil {
			u.hotkey.Close()
		}
		u.store.Close()
	})
}

func (u *UI) quit() {
	u.stop()
	u.fyneApp.Quit()
}

func (u *UI) Quit() { u.quit() }

func (u *UI) registerHotkey(mode string) {
	log.Printf("register hotkey %s", platform.ShortcutLabel(mode))
	err := u.hotkey.Register(mode)
	u.hotkeyErr = err
	if err == nil {
		log.Printf("hotkey registered: %s", platform.ShortcutLabel(mode))
	}
	if err != nil {
		log.Printf("hotkey %s: %v", platform.ShortcutLabel(mode), err)
		u.notify(u.localization.L("status.hotkey.failed", platform.ShortcutLabel(mode), err.Error()))
	}
	u.rebuildMenu()
}

func (u *UI) needsSetup() bool {
	for _, permission := range platform.Permissions() {
		if permission.Required && !permission.Allowed {
			return true
		}
	}
	return false
}

func (u *UI) refreshKeyState() {
	value, err := u.secrets.Get()
	if err != nil {
		log.Printf("read secret: %v", err)
	}
	configured := strings.TrimSpace(value) != ""
	u.keyConfigured.Store(configured)
	if !configured && !u.secrets.FromEnvironment() {
		fyne.Do(func() { u.notify(u.localization.L("status.key.missing")) })
	}
	fyne.Do(u.rebuildMenu)
}

func (u *UI) rebuildMenu() {
	if u.desktopApp == nil {
		return
	}
	u.desktopApp.SetSystemTrayMenu(u.buildMenu())
}

func (u *UI) buildMenu() *fyne.Menu {
	settings := u.settings.Get()
	shortcut := platform.ShortcutLabel(string(settings.Shortcut))

	openItem := fyne.NewMenuItem(u.localization.L("menu.open", shortcut), func() {
		fyne.Do(u.panel.Show)
	})

	captureLabel := u.localization.L("menu.pauseCapture")
	if settings.CapturePaused {
		captureLabel = u.localization.L("menu.resumeCapture")
	}
	captureItem := fyne.NewMenuItem(captureLabel, func() {
		fyne.Do(u.toggleCapture)
	})
	captureItem.Checked = settings.CapturePaused

	clearItem := fyne.NewMenuItem(u.localization.L("menu.clearHistory"), func() {
		fyne.Do(u.showClearHistory)
	})
	clearItem.Disabled = len(u.store.Clips()) == 0

	configureLabel := u.localization.L("menu.key.configure")
	if u.keyConfigured.Load() {
		configureLabel = u.localization.L("menu.key.update")
	}
	configureItem := fyne.NewMenuItem(configureLabel, func() {
		fyne.Do(u.showKeyDialog)
	})
	removeKeyItem := fyne.NewMenuItem(u.localization.L("menu.key.remove"), func() {
		fyne.Do(u.showRemoveKey)
	})
	removeKeyItem.Disabled = !u.keyConfigured.Load()

	vItem := fyne.NewMenuItem(u.localization.L("menu.shortcut.useV", platform.ShortcutLabel("shiftV")), func() {
		fyne.Do(func() { u.selectShortcut(string(config.ShortcutShiftV)) })
	})
	vItem.Checked = settings.Shortcut == config.ShortcutShiftV
	jItem := fyne.NewMenuItem(u.localization.L("menu.shortcut.useJ", platform.ShortcutLabel("shiftJ")), func() {
		fyne.Do(func() { u.selectShortcut(string(config.ShortcutShiftJ)) })
	})
	jItem.Checked = settings.Shortcut == config.ShortcutShiftJ
	hintItem := fyne.NewMenuItem(u.localization.L("menu.shortcut.hint"), nil)
	hintItem.Disabled = true

	shortcutMenu := fyne.NewMenu(
		u.localization.L("menu.shortcut", shortcut),
		vItem, jItem, fyne.NewMenuItemSeparator(), hintItem,
	)

	languageItems := make([]*fyne.MenuItem, 0, len(u.localization.Available())+4)
	for _, pack := range u.localization.Available() {
		code := pack.Code
		item := fyne.NewMenuItem(pack.Name, func() {
			fyne.Do(func() { u.selectLanguage(code) })
		})
		item.Checked = code == u.localization.CurrentCode()
		languageItems = append(languageItems, item)
	}
	languageItems = append(languageItems,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(u.localization.L("menu.language.load"), func() {
			fyne.Do(u.showLoadLanguage)
		}),
		fyne.NewMenuItem(u.localization.L("menu.language.openFolder"), func() {
			fyne.Do(u.openLanguagesFolder)
		}),
	)
	languageMenu := fyne.NewMenu(u.localization.L("menu.language"), languageItems...)

	setupItem := fyne.NewMenuItem(u.localization.L("menu.setup"), func() {
		fyne.Do(func() { u.showSetup(nil) })
	})

	quitItem := fyne.NewMenuItem(u.localization.L("menu.quit"), u.quit)
	quitItem.IsQuit = true

	return fyne.NewMenu("",
		openItem,
		captureItem,
		clearItem,
		fyne.NewMenuItemSeparator(),
		configureItem,
		removeKeyItem,
		&fyne.MenuItem{Label: shortcutMenu.Label, ChildMenu: shortcutMenu},
		&fyne.MenuItem{Label: languageMenu.Label, ChildMenu: languageMenu},
		setupItem,
		fyne.NewMenuItemSeparator(),
		quitItem,
	)
}

func (u *UI) toggleCapture() {
	paused := !u.settings.Get().CapturePaused
	if err := u.settings.Update(func(settings *config.Settings) {
		settings.CapturePaused = paused
	}); err != nil {
		u.showError(err)
		return
	}
	u.rebuildMenu()
}

func (u *UI) selectShortcut(mode string) {
	current := string(u.settings.Get().Shortcut)
	if mode == current {
		return
	}
	if err := u.hotkey.Register(mode); err != nil {
		if errors.Is(err, platform.ErrRestartRequired) {
			_ = u.settings.Update(func(settings *config.Settings) {
				settings.Shortcut = config.ShortcutMode(mode)
			})
			u.rebuildMenu()
			u.showMessage(u.localization.L("app.name"), u.localization.L("status.restartShortcut"))
			return
		}
		u.hotkeyErr = err
		u.showError(fmt.Errorf("%s: %w", u.localization.L("status.hotkey.failed", platform.ShortcutLabel(mode), err.Error()), err))
		_ = u.hotkey.Register(current)
		return
	}
	u.hotkeyErr = nil
	if err := u.settings.Update(func(settings *config.Settings) {
		settings.Shortcut = config.ShortcutMode(mode)
	}); err != nil {
		u.showError(err)
	}
	u.rebuildMenu()
}

func (u *UI) selectLanguage(code string) {
	if !u.localization.Select(code) {
		return
	}
	_ = u.settings.Update(func(settings *config.Settings) {
		settings.Language = code
	})
	u.panel.ApplyLanguage()
	u.rebuildMenu()
}

func (u *UI) notify(message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	log.Print(message)
	u.fyneApp.SendNotification(&fyne.Notification{
		Title:   u.localization.L("app.name"),
		Content: message,
	})
}

func (u *UI) showError(err error) {
	u.showMessage(u.localization.L("error.title"), err.Error())
}

func (u *UI) showMessage(title, message string) {
	if u.panel == nil {
		log.Printf("%s: %s", title, message)
		return
	}
	if !u.panel.Visible() {
		u.panel.Show()
	}
	dialog.ShowInformation(title, message, u.panel.window)
}

func (u *UI) openFolder(path string) {
	url := fileURL(path)
	if err := u.fyneApp.OpenURL(url); err == nil {
		return
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", path)
	case "windows":
		command = exec.Command("explorer", path)
	default:
		command = exec.Command("xdg-open", path)
	}
	if err := command.Start(); err != nil {
		u.showError(err)
	}
}
