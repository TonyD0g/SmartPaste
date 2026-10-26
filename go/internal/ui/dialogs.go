package ui

import (
	"fmt"
	"image/color"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"smartpaste/internal/config"
	"smartpaste/internal/platform"
)

func (u *UI) showWelcome() {
	message := u.localization.L(
		"welcome.body",
		platform.ShortcutLabel(string(u.settings.Get().Shortcut)),
	)
	popup := dialog.NewInformation(
		u.localization.L("welcome.windowTitle"),
		message,
		u.panel.window,
	)
	popup.SetOnClosed(func() {
		_ = u.settings.Update(func(settings *config.Settings) {
			settings.HasShownFirstLaunch = true
		})
		u.showSetup(func() {
			u.panel.Hide()
		})
	})
	popup.Show()
}

func (u *UI) showSetup(onClosed func()) {
	if !u.setupVisible.Load() {
		u.setupWasPanelVisible = u.panel.Visible()
		if !u.setupWasPanelVisible {
			u.panel.Show()
		}
	}
	u.setupOnClosed = onClosed

	if u.setupWindow == nil {
		u.setupWindow = u.fyneApp.NewWindow(u.localization.L("setup.title"))
		u.setupWindow.SetPadded(true)
		u.setupWindow.SetCloseIntercept(u.hideSetup)
		u.setupWindow.Resize(fyne.NewSize(600, 560))
		u.setupWindow.CenterOnScreen()
	}
	u.setupWindow.SetTitle(u.localization.L("setup.title"))

	closeButton := widget.NewButton(u.localization.L("setup.continue"), u.hideSetup)
	footer := container.NewHBox(layout.NewSpacer(), closeButton)
	u.setupWindow.SetContent(container.NewBorder(
		nil,
		footer,
		nil,
		nil,
		container.NewVScroll(u.buildSetupBody()),
	))
	u.setupWindow.Show()
	u.setupVisible.Store(true)
	u.setupWindow.RequestFocus()
}

func (u *UI) hideSetup() {
	u.setupVisible.Store(false)
	if u.setupWindow != nil {
		u.setupWindow.Hide()
	}
	if !u.setupWasPanelVisible {
		u.panel.Hide()
	}
	callback := u.setupOnClosed
	u.setupOnClosed = nil
	if callback != nil {
		callback()
	}
	u.rebuildMenu()
}

func (u *UI) buildSetupBody() fyne.CanvasObject {
	permissions := u.permissionSnapshot()
	statusLabels := make(map[string]*widget.Label, len(permissions))
	rows := make([]fyne.CanvasObject, 0, len(permissions)+1)

	refresh := func() {
		for _, permission := range u.permissionSnapshot() {
			if label, ok := statusLabels[permission.Key]; ok {
				u.styleStatus(label, permission)
			}
		}
	}

	for _, permission := range permissions {
		status := widget.NewLabel("")
		statusLabels[permission.Key] = status
		u.styleStatus(status, permission)
		rows = append(rows, u.permissionRow(permission, status))
	}

	title := widget.NewLabelWithStyle(
		u.localization.L("setup.title"),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	title.SizeName = theme.SizeNameSubHeadingText

	subtitle := widget.NewLabel(u.localization.L("setup.subtitle"))
	subtitle.Importance = widget.LowImportance
	subtitle.Wrapping = fyne.TextWrapWord

	shortcut := widget.NewLabel(u.localization.L(
		"setup.shortcut",
		platform.ShortcutLabel(string(u.settings.Get().Shortcut)),
	))
	shortcut.Wrapping = fyne.TextWrapWord

	noteKey := "setup.note.linux"
	switch runtime.GOOS {
	case "darwin":
		noteKey = "setup.note.macos"
	case "windows":
		noteKey = "setup.note.windows"
	}
	note := widget.NewLabel(u.localization.L(noteKey))
	note.Importance = widget.LowImportance
	note.Wrapping = fyne.TextWrapWord

	buttons := container.NewHBox()
	if runtime.GOOS == "darwin" {
		buttons.Add(widget.NewButton(u.localization.L("setup.requestPermissions"), func() {
			platform.RequestPermissions()
			time.AfterFunc(900*time.Millisecond, func() {
				fyne.Do(refresh)
			})
		}))
	}
	buttons.Add(widget.NewButton(u.localization.L("setup.refresh"), func() {
		u.showSetup(u.setupOnClosed)
	}))

	body := container.NewVBox(
		title,
		subtitle,
		shortcut,
		widget.NewSeparator(),
		container.NewVBox(rows...),
		buttons,
		widget.NewSeparator(),
		note,
	)
	if platform.IsWayland() {
		wayland := widget.NewLabel(u.localization.L("status.wayland"))
		wayland.Importance = widget.WarningImportance
		wayland.Wrapping = fyne.TextWrapWord
		body.Add(wayland)
	}
	return container.NewPadded(body)
}

func (u *UI) permissionRow(permission platform.Permission, status *widget.Label) fyne.CanvasObject {
	label := widget.NewLabel(u.localization.L(permission.LabelKey))
	label.TextStyle = fyne.TextStyle{Bold: true}

	var action fyne.CanvasObject = canvas.NewRectangle(color.Transparent)
	if permission.SettingsURL != "" {
		action = widget.NewButton(u.localization.L("setup.openSettings"), func() {
			_ = platform.OpenPermissionSettings(permission)
		})
	}
	actionBox := container.NewGridWrap(fyne.NewSize(132, 34), action)
	return container.NewHBox(label, layout.NewSpacer(), status, actionBox)
}

func (u *UI) styleStatus(label *widget.Label, permission platform.Permission) {
	switch {
	case permission.Allowed:
		label.SetText("✓ " + u.localization.L("setup.permission.allowed"))
		label.Importance = widget.SuccessImportance
	case !permission.Required:
		label.SetText("○ " + u.localization.L("setup.permission.notRequired"))
		label.Importance = widget.LowImportance
	default:
		label.SetText("○ " + u.localization.L("setup.permission.notAllowed"))
		label.Importance = widget.LowImportance
	}
}

func (u *UI) permissionSnapshot() []platform.Permission {
	permissions := platform.Permissions()
	hasHotkey := false
	for index := range permissions {
		if permissions[index].Key == "hotkey" {
			permissions[index].Allowed = u.hotkeyErr == nil
			hasHotkey = true
		}
	}
	if runtime.GOOS == "windows" {
		permissions = append(permissions,
			platform.Permission{
				Key:      "clipboard",
				LabelKey: "setup.permission.clipboard",
				Allowed:  platform.ClipboardSupported(),
				Required: true,
			},
			platform.Permission{
				Key:      "focus",
				LabelKey: "setup.permission.focus",
				Allowed:  platform.FocusSupported(),
			},
		)
	}
	if !hasHotkey {
		permissions = append(permissions, platform.Permission{
			Key:      "hotkey",
			LabelKey: "setup.permission.hotkey",
			Allowed:  u.hotkeyErr == nil,
			Required: runtime.GOOS != "linux" || !platform.IsWayland(),
		})
	}
	return permissions
}

func (u *UI) showKeyDialog() {
	wasVisible := u.panel.Visible()
	if !wasVisible {
		u.panel.Show()
	}

	entry := widget.NewPasswordEntry()
	entry.SetPlaceHolder(u.localization.L("alert.key.placeholder"))
	message := widget.NewLabel(u.localization.L("alert.key.message"))
	message.Wrapping = fyne.TextWrapWord
	envNote := widget.NewLabel(u.localization.L("alert.key.envNote"))
	envNote.Importance = widget.LowImportance
	envNote.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(message, entry, envNote)
	title := u.localization.L("alert.key.titleUpdate")
	if !u.keyConfigured.Load() {
		title = u.localization.L("alert.key.titleConfigure")
	}

	popup := dialog.NewCustomConfirm(
		title,
		u.localization.L("alert.key.save"),
		u.localization.L("alert.cancel"),
		content,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			value := strings.TrimSpace(entry.Text)
			if value == "" {
				return
			}
			if err := u.secrets.Set(value); err != nil {
				u.showError(fmt.Errorf("%s: %w", u.localization.L("error.keyring"), err))
				return
			}
			u.controller.ResetKeyWarning()
			go u.refreshKeyState()
		},
		u.panel.window,
	)
	popup.Resize(fyne.NewSize(480, 250))
	popup.Show()
	u.panel.window.Canvas().Focus(entry)
	popup.SetOnClosed(func() {
		if !wasVisible {
			u.panel.Hide()
		}
	})
}

func (u *UI) showRemoveKey() {
	wasVisible := u.panel.Visible()
	if !wasVisible {
		u.panel.Show()
	}
	popup := dialog.NewConfirm(
		u.localization.L("alert.removeKey.title"),
		u.localization.L("alert.removeKey.message"),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := u.secrets.Delete(); err != nil {
				u.showError(err)
				return
			}
			go u.refreshKeyState()
		},
		u.panel.window,
	)
	popup.SetOnClosed(func() {
		if !wasVisible {
			u.panel.Hide()
		}
	})
	popup.Show()
}

func (u *UI) showClearHistory() {
	wasVisible := u.panel.Visible()
	if !wasVisible {
		u.panel.Show()
	}
	count := len(u.store.Clips())
	message := u.localization.L("alert.clear.message.one")
	if count != 1 {
		message = u.localization.L("alert.clear.message.other", count)
	}
	popup := dialog.NewConfirm(
		u.localization.L("alert.clear.title"),
		message,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			u.store.Clear()
			u.controller.Reset()
			u.panel.Reload()
			u.rebuildMenu()
		},
		u.panel.window,
	)
	popup.SetOnClosed(func() {
		if !wasVisible {
			u.panel.Hide()
		}
	})
	popup.Show()
}

func (u *UI) showLoadLanguage() {
	if !u.panel.Visible() {
		u.panel.Show()
	}
	popup := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			u.showError(err)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		path := nativePath(reader.URI())
		pack, installErr := u.localization.Install(path)
		if installErr != nil {
			u.showError(fmt.Errorf("%s: %s", u.localization.L("language.error.title"), u.localization.Error(installErr)))
			return
		}
		u.localization.Select(pack.Code)
		_ = u.settings.Update(func(settings *config.Settings) {
			settings.Language = pack.Code
		})
		u.panel.ApplyLanguage()
		u.rebuildMenu()
	}, u.panel.window)
	popup.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	popup.Show()
}

func (u *UI) openLanguagesFolder() {
	u.openFolder(u.paths.LanguagesDir)
}

func nativePath(uri fyne.URI) string {
	path := filepath.FromSlash(uri.Path())
	if runtime.GOOS == "windows" && len(path) > 2 && path[0] == filepath.Separator && path[2] == ':' {
		path = path[1:]
	}
	return path
}

func fileURL(path string) *url.URL {
	slashed := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return &url.URL{Scheme: "file", Path: slashed}
}
