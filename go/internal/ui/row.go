package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"smartpaste/internal/i18n"
	"smartpaste/internal/model"
	"smartpaste/internal/store"
)

const (
	rowHeight = float32(74)
	rowGap    = float32(8)
	iconSize  = float32(44)
)

type clipRow struct {
	widget.BaseWidget

	clip         model.Clip
	localization *i18n.Manager
	store        *store.Store
	index        int
	selected     bool
	hovered      bool
	lastTap      time.Time
	canvas       fyne.Canvas

	background *canvas.Rectangle
	iconSlot   *fyne.Container
	title      *widget.Label
	meta       *widget.Label
	stack      *fyne.Container

	onSelect   func(int)
	onActivate func(int)
	onDelete   func(int)
}

func newClipRow(
	clip model.Clip,
	dataStore *store.Store,
	localization *i18n.Manager,
	index int,
	canvas fyne.Canvas,
	onSelect func(int),
	onActivate func(int),
	onDelete func(int),
) *clipRow {
	row := &clipRow{
		clip:         clip,
		localization: localization,
		store:        dataStore,
		index:        index,
		canvas:       canvas,
		onSelect:     onSelect,
		onActivate:   onActivate,
		onDelete:     onDelete,
	}
	row.build()
	row.ExtendBaseWidget(row)
	return row
}

func (r *clipRow) build() {
	r.background = canvas.NewRectangle(color.NRGBA{R: 0x26, G: 0x26, B: 0x28, A: 0xFF})
	r.background.CornerRadius = 10
	r.background.StrokeWidth = 1
	r.background.StrokeColor = color.NRGBA{R: 0x3A, G: 0x3A, B: 0x3C, A: 0xFF}

	icon := r.iconObject()
	r.iconSlot = container.NewGridWrap(fyne.NewSize(iconSize, iconSize), icon)

	r.title = widget.NewLabel(r.previewText())
	r.title.Truncation = fyne.TextTruncateEllipsis
	r.title.Wrapping = fyne.TextWrapOff

	r.meta = widget.NewLabel(r.metaText())
	r.meta.Importance = widget.LowImportance
	r.meta.Truncation = fyne.TextTruncateEllipsis
	r.meta.Wrapping = fyne.TextWrapOff
	r.meta.SizeName = theme.SizeNameCaptionText

	textBox := container.NewVBox(r.title, r.meta)
	inner := container.NewBorder(nil, nil, r.iconSlot, nil, textBox)
	padded := container.NewPadded(inner)
	r.stack = container.NewStack(r.background, padded)
	r.updateColors()
}

func (r *clipRow) CreateRenderer() fyne.WidgetRenderer {
	r.ExtendBaseWidget(r)
	return widget.NewSimpleRenderer(r.stack)
}

func (r *clipRow) iconObject() fyne.CanvasObject {
	if r.clip.IsImage() {
		if thumb, ok := r.store.Thumbnail(r.clip); ok {
			image := canvas.NewImageFromImage(thumb)
			image.FillMode = canvas.ImageFillContain
			image.SetMinSize(fyne.NewSize(iconSize, iconSize))
			return image
		}
	}

	icon := widget.NewIcon(iconForText(r.clip.Text))
	iconBox := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x0A})
	iconBox.CornerRadius = 8
	return container.NewStack(iconBox, container.NewCenter(icon))
}

func iconForText(text string) fyne.Resource {
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return theme.ComputerIcon()
	}
	if strings.Contains(lower, "@") && !strings.Contains(lower, " ") {
		return theme.MailComposeIcon()
	}
	return theme.FileTextIcon()
}

func (r *clipRow) previewText() string {
	if r.clip.IsImage() {
		return r.localization.L("timeline.image")
	}
	text := strings.ReplaceAll(r.clip.Text, "\n", " ")
	text = strings.TrimSpace(text)
	if len([]rune(text)) > 240 {
		text = string([]rune(text)[:240])
	}
	if text == "" {
		return " "
	}
	return text
}

func (r *clipRow) metaText() string {
	parts := make([]string, 0, 3)
	if relative := relativeTime(r.localization, r.clip.CreatedAt.Time); relative != "" {
		parts = append(parts, relative)
	}
	if source := strings.TrimSpace(r.clip.SourceApp); source != "" {
		parts = append(parts, source)
	}
	if r.clip.IsImage() && r.clip.ImagePixelWidth > 0 && r.clip.ImagePixelHeight > 0 {
		parts = append(parts, fmt.Sprintf("%d×%d", r.clip.ImagePixelWidth, r.clip.ImagePixelHeight))
	}
	return strings.Join(parts, " · ")
}

func relativeTime(localization *i18n.Manager, created time.Time) string {
	if created.IsZero() {
		return ""
	}
	elapsed := time.Since(created)
	switch {
	case elapsed < time.Minute:
		return localization.L("time.justNow")
	case elapsed < time.Hour:
		return localization.L("time.minutesAgo", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return localization.L("time.hoursAgo", int(elapsed.Hours()))
	case elapsed < 7*24*time.Hour:
		return localization.L("time.daysAgo", int(elapsed.Hours()/24))
	default:
		return created.Format("Jan 2")
	}
}

func (r *clipRow) SetSelected(selected bool) {
	if r.selected == selected {
		return
	}
	r.selected = selected
	r.updateColors()
}

func (r *clipRow) SetClip(clip model.Clip) {
	r.clip = clip
	r.title.SetText(r.previewText())
	r.meta.SetText(r.metaText())
	r.iconSlot.Objects[0] = r.iconObject()
	r.iconSlot.Refresh()
}

func (r *clipRow) SetIndex(index int) { r.index = index }

func (r *clipRow) updateColors() {
	if r.background == nil {
		return
	}
	switch {
	case r.selected:
		r.background.FillColor = color.NRGBA{R: 0x0A, G: 0x84, B: 0xFF, A: 0x28}
		r.background.StrokeColor = color.NRGBA{R: 0x0A, G: 0x84, B: 0xFF, A: 0x9A}
	case r.hovered:
		r.background.FillColor = color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2E, A: 0xFF}
		r.background.StrokeColor = color.NRGBA{R: 0x4A, G: 0x4A, B: 0x4C, A: 0xFF}
	default:
		r.background.FillColor = color.NRGBA{R: 0x26, G: 0x26, B: 0x28, A: 0xFF}
		r.background.StrokeColor = color.NRGBA{R: 0x3A, G: 0x3A, B: 0x3C, A: 0xFF}
	}
	r.background.Refresh()
}

func (r *clipRow) Tapped(*fyne.PointEvent) {
	if r.onSelect != nil {
		r.onSelect(r.index)
	}
	if time.Since(r.lastTap) < 420*time.Millisecond {
		if r.onActivate != nil {
			r.onActivate(r.index)
		}
		r.lastTap = time.Time{}
		return
	}
	r.lastTap = time.Now()
}

func (r *clipRow) TappedSecondary(event *fyne.PointEvent) {
	if r.canvas == nil || r.onDelete == nil {
		return
	}
	menu := widget.NewPopUpMenu(fyne.NewMenu("",
		fyne.NewMenuItem(r.localization.L("timeline.delete"), func() {
			r.onDelete(r.index)
		}),
	), r.canvas)
	menu.ShowAtPosition(event.AbsolutePosition)
}

func (r *clipRow) MouseIn(*desktop.MouseEvent) {
	r.hovered = true
	r.updateColors()
}

func (r *clipRow) MouseMoved(*desktop.MouseEvent) {}

func (r *clipRow) MouseOut() {
	r.hovered = false
	r.updateColors()
}
