package ui

import (
	"image/color"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"smartpaste/internal/app"
	"smartpaste/internal/config"
	"smartpaste/internal/i18n"
	"smartpaste/internal/model"
	"smartpaste/internal/platform"
	"smartpaste/internal/store"
)

const (
	panelHeight  = float32(313)
	minWidth     = float32(440)
	defaultWidth = float32(560)
)

type Panel struct {
	fyneApp      fyne.App
	window       fyne.Window
	controller   *app.Controller
	store        *store.Store
	settings     *config.SettingsStore
	localization *i18n.Manager

	title   *widget.Label
	search  *widget.Entry
	list    *snappingList
	empty   *widget.Label
	content *fyne.Container

	results        []model.Clip
	previous       platform.Frontmost
	suppressSearch bool
	visible        atomic.Bool
}

func NewPanel(
	fyneApp fyne.App,
	controller *app.Controller,
	dataStore *store.Store,
	settings *config.SettingsStore,
	localization *i18n.Manager,
) *Panel {
	panel := &Panel{
		fyneApp:      fyneApp,
		controller:   controller,
		store:        dataStore,
		settings:     settings,
		localization: localization,
	}
	panel.build()
	return panel
}

func (p *Panel) build() {
	p.window = p.fyneApp.NewWindow(p.localization.L("app.name"))
	p.window.SetPadded(false)
	p.window.SetFixedSize(false)
	p.window.SetIcon(appIcon())
	p.window.SetCloseIntercept(p.Hide)
	if window, ok := p.window.(desktop.Window); ok {
		window.RequestAlwaysOnTop()
	}

	p.title = widget.NewLabelWithStyle(p.localization.L("app.name"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.title.SizeName = theme.SizeNameSubHeadingText

	badge := p.buildBadge()

	p.search = widget.NewEntry()
	p.search.SetPlaceHolder(p.localization.L("timeline.search"))
	p.search.OnChanged = func(string) {
		if p.suppressSearch {
			return
		}
		p.list.Select(0)
		p.reload()
	}
	searchBox := container.NewGridWrap(fyne.NewSize(210, 34), p.search)

	headerRow := container.NewHBox(p.title, badge, layout.NewSpacer(), searchBox)
	header := container.New(&insetLayout{left: 18, top: 10, right: 14, bottom: 10}, headerRow)

	separator := widget.NewSeparator()
	separatorBox := container.New(&insetLayout{left: 16, right: 16, top: 0, bottom: 0}, separator)
	top := container.NewVBox(header, separatorBox)

	p.list = newSnappingList(p.store, p.localization, p.window.Canvas(), p.pasteClip, p.deleteClip)

	p.empty = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{})
	p.empty.Importance = widget.LowImportance
	p.empty.Wrapping = fyne.TextWrapWord
	p.empty.Hide()

	listArea := container.NewStack(p.list, container.NewCenter(p.empty))

	p.content = container.NewBorder(top, nil, nil, nil, listArea)
	p.window.SetContent(p.content)

	p.installShortcuts()
	p.reload()
}

func (p *Panel) buildBadge() fyne.CanvasObject {
	background := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x0A})
	background.CornerRadius = 5
	background.StrokeWidth = 1
	background.StrokeColor = color.NRGBA{R: 0x3A, G: 0x3A, B: 0x3C, A: 0xFF}

	label := widget.NewLabelWithStyle("BETA", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	label.Importance = widget.LowImportance
	label.SizeName = theme.SizeNameCaptionText

	return container.NewGridWrap(
		fyne.NewSize(42, 18),
		container.NewStack(background, container.NewCenter(label)),
	)
}

func (p *Panel) installShortcuts() {
	canvas := p.window.Canvas()
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyDown}, func(fyne.Shortcut) {
		p.list.MoveSelection(1)
	})
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyUp}, func(fyne.Shortcut) {
		p.list.MoveSelection(-1)
	})
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyReturn}, func(fyne.Shortcut) {
		p.list.ActivateSelected()
	})
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEnter}, func(fyne.Shortcut) {
		p.list.ActivateSelected()
	})
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEscape}, func(fyne.Shortcut) {
		p.Hide()
	})
	canvas.AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyBackspace,
		Modifier: fyne.KeyModifierControl,
	}, func(fyne.Shortcut) {
		p.list.DeleteSelected()
	})
	if runtime.GOOS == "darwin" {
		canvas.AddShortcut(&desktop.CustomShortcut{
			KeyName:  fyne.KeyBackspace,
			Modifier: fyne.KeyModifierSuper,
		}, func(fyne.Shortcut) {
			p.list.DeleteSelected()
		})
	}
}

func (p *Panel) Show() {
	if p.Visible() {
		p.window.RequestFocus()
		p.window.Canvas().Focus(p.search)
		return
	}

	p.previous = platform.FrontmostApp()

	p.suppressSearch = true
	p.search.SetText("")
	p.suppressSearch = false
	p.list.Select(0)
	p.reload()

	width := float32(p.settings.Get().TimelineWidth)
	if width < minWidth {
		width = defaultWidth
	}
	p.window.Resize(fyne.NewSize(width, panelHeight))
	p.window.CenterOnScreen()
	p.window.Show()
	p.visible.Store(true)
	p.window.RequestFocus()
	p.window.Canvas().Focus(p.search)
}

func (p *Panel) Hide() {
	if !p.Visible() {
		return
	}
	width := p.window.Content().Size().Width
	if width >= minWidth {
		_ = p.settings.Update(func(settings *config.Settings) {
			settings.TimelineWidth = float64(width)
		})
	}
	p.visible.Store(false)
	p.window.Hide()
}

func (p *Panel) Visible() bool {
	return p.visible.Load()
}

func (p *Panel) Reload() {
	p.reload()
}

func (p *Panel) ApplyLanguage() {
	p.window.SetTitle(p.localization.L("app.name"))
	p.title.SetText(p.localization.L("app.name"))
	p.search.SetPlaceHolder(p.localization.L("timeline.search"))
	p.reload()
}

func (p *Panel) reload() {
	query := strings.ToLower(strings.TrimSpace(p.search.Text))

	rankByText := make(map[string]int)
	for index, text := range p.controller.RankedTexts() {
		rankByText[strings.ToLower(text)] = index
	}

	clips := p.store.Clips()
	indexed := make([]struct {
		clip model.Clip
		pos  int
	}, len(clips))
	for index, clip := range clips {
		indexed[index] = struct {
			clip model.Clip
			pos  int
		}{clip: clip, pos: index}
	}
	sort.SliceStable(indexed, func(i, j int) bool {
		left, leftOK := rankByText[strings.ToLower(indexed[i].clip.Text)]
		right, rightOK := rankByText[strings.ToLower(indexed[j].clip.Text)]
		switch {
		case leftOK && rightOK:
			return left < right
		case leftOK:
			return true
		case rightOK:
			return false
		default:
			return indexed[i].pos < indexed[j].pos
		}
	})

	results := make([]model.Clip, 0, len(indexed))
	for _, entry := range indexed {
		if query != "" && !strings.Contains(strings.ToLower(entry.clip.Text), query) {
			continue
		}
		results = append(results, entry.clip)
	}
	p.results = results

	previous := p.list.Selected()
	p.list.SetClips(results)
	if len(results) > 0 {
		p.list.Select(clampIndex(previous, len(results)))
	}

	if len(results) == 0 {
		p.list.Hide()
		p.empty.Show()
		switch {
		case len(clips) == 0:
			p.empty.SetText(p.localization.L("timeline.empty"))
		default:
			p.empty.SetText(p.localization.L("timeline.noResults"))
		}
	} else {
		p.empty.Hide()
		p.list.Show()
	}
}

func (p *Panel) pasteClip(clip model.Clip) {
	p.paste(clip)
}

func (p *Panel) paste(clip model.Clip) {
	previous := p.previous
	p.Hide()
	go func() {
		time.Sleep(120 * time.Millisecond)
		p.controller.PasteClip(clip, &previous)
	}()
}

func (p *Panel) deleteClip(clip model.Clip) {
	index := p.list.Selected()
	p.store.Remove([]model.Clip{clip})
	p.reload()
	if len(p.results) > 0 {
		p.list.Select(clampIndex(index, len(p.results)))
	}
}

type insetLayout struct {
	left   float32
	top    float32
	right  float32
	bottom float32
}

func (l *insetLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(l.left, l.top))
	objects[0].Resize(fyne.NewSize(
		maxFloat(size.Width-l.left-l.right, 0),
		maxFloat(size.Height-l.top-l.bottom, 0),
	))
}

func (l *insetLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(l.left+l.right, l.top+l.bottom)
	}
	min := objects[0].MinSize()
	return fyne.NewSize(min.Width+l.left+l.right, min.Height+l.top+l.bottom)
}
