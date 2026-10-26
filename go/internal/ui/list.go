package ui

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"smartpaste/internal/i18n"
	"smartpaste/internal/model"
	"smartpaste/internal/store"
)

const (
	visibleRows = 3
	rowStep     = rowHeight + rowGap
)

type snappingList struct {
	widget.BaseWidget

	items        []model.Clip
	rows         []*clipRow
	content      *fyne.Container
	scroller     *scrollClip
	selected     int
	offset       float32
	canvas       fyne.Canvas
	dataStore    *store.Store
	localization *i18n.Manager

	onActivate func(model.Clip)
	onDelete   func(model.Clip)
}

func newSnappingList(
	dataStore *store.Store,
	localization *i18n.Manager,
	canvas fyne.Canvas,
	onActivate func(model.Clip),
	onDelete func(model.Clip),
) *snappingList {
	list := &snappingList{
		canvas:       canvas,
		dataStore:    dataStore,
		localization: localization,
		onActivate:   onActivate,
		onDelete:     onDelete,
	}
	list.content = container.NewWithoutLayout()
	list.scroller = newScrollClip(list.content, list.handleScroll)
	list.ExtendBaseWidget(list)
	return list
}

func (l *snappingList) CreateRenderer() fyne.WidgetRenderer {
	l.ExtendBaseWidget(l)
	return widget.NewSimpleRenderer(l.scroller)
}

func (l *snappingList) MinSize() fyne.Size {
	return fyne.NewSize(320, rowStep*visibleRows-rowGap+16)
}

func (l *snappingList) Resize(size fyne.Size) {
	l.BaseWidget.Resize(size)
	if l.scroller != nil {
		l.scroller.Resize(size)
	}
	l.layout()
}

func (l *snappingList) SetClips(clips []model.Clip) {
	for _, row := range l.rows {
		l.content.Remove(row)
	}
	l.items = clips
	l.rows = make([]*clipRow, 0, len(clips))
	for index, clip := range clips {
		row := newClipRow(
			clip,
			l.dataStore,
			l.localization,
			index,
			l.canvas,
			l.selectIndex,
			l.activateIndex,
			l.deleteIndex,
		)
		l.rows = append(l.rows, row)
		l.content.Add(row)
	}
	l.selected = clampIndex(l.selected, len(l.rows))
	l.updateSelection()
	l.layout()
	l.content.Refresh()
}

func (l *snappingList) Refresh() {
	for _, row := range l.rows {
		row.Refresh()
	}
	l.layout()
	l.BaseWidget.Refresh()
}

func (l *snappingList) layout() {
	if l.content == nil || l.Size().Width <= 0 {
		return
	}
	width := l.Size().Width
	height := float32(len(l.rows))*rowStep + 16
	if height < l.Size().Height {
		height = l.Size().Height
	}
	l.content.Resize(fyne.NewSize(width, height))
	for index, row := range l.rows {
		row.Move(fyne.NewPos(12, 8+float32(index)*rowStep-l.offset))
		row.Resize(fyne.NewSize(maxFloat(width-24, 80), rowHeight))
	}
}

func (l *snappingList) handleScroll(event *fyne.ScrollEvent) {
	if len(l.items) <= visibleRows {
		return
	}
	direction := float32(1)
	if event.Scrolled.DY > 0 {
		direction = -1
	}
	l.setOffset(l.offset + direction*rowStep)
}

func (l *snappingList) setOffset(offset float32) {
	maximum := maxFloat(0, float32(len(l.items)-visibleRows)*rowStep)
	offset = minFloat(maxFloat(offset, 0), maximum)
	offset = float32(math.Round(float64(offset/rowStep))) * rowStep
	if offset == l.offset {
		return
	}
	l.offset = offset
	l.layout()
}

func (l *snappingList) Select(index int) {
	index = clampIndex(index, len(l.rows))
	if index == l.selected {
		l.ensureVisible(index)
		return
	}
	l.selected = index
	l.updateSelection()
	l.ensureVisible(index)
}

func (l *snappingList) MoveSelection(delta int) {
	l.Select(l.selected + delta)
}

func (l *snappingList) Selected() int { return l.selected }

func (l *snappingList) SelectedClip() (model.Clip, bool) {
	if l.selected < 0 || l.selected >= len(l.items) {
		return model.Clip{}, false
	}
	return l.items[l.selected], true
}

func (l *snappingList) ActivateSelected() {
	if l.onActivate != nil {
		if clip, ok := l.SelectedClip(); ok {
			l.onActivate(clip)
		}
	}
}

func (l *snappingList) DeleteSelected() {
	if l.onDelete != nil {
		if clip, ok := l.SelectedClip(); ok {
			l.onDelete(clip)
		}
	}
}

func (l *snappingList) ensureVisible(index int) {
	first := int(math.Round(float64(l.offset / rowStep)))
	if index < first {
		l.setOffset(float32(index) * rowStep)
		return
	}
	if index >= first+visibleRows {
		l.setOffset(float32(index-visibleRows+1) * rowStep)
	}
}

func (l *snappingList) updateSelection() {
	for index, row := range l.rows {
		row.SetSelected(index == l.selected)
	}
}

func (l *snappingList) selectIndex(index int) {
	l.Select(index)
}

func (l *snappingList) activateIndex(index int) {
	l.Select(index)
	if l.onActivate != nil {
		if clip, ok := l.SelectedClip(); ok {
			l.onActivate(clip)
		}
	}
}

func (l *snappingList) deleteIndex(index int) {
	l.Select(index)
	if l.onDelete != nil {
		if clip, ok := l.SelectedClip(); ok {
			l.onDelete(clip)
		}
	}
}

func clampIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

type scrollClip struct {
	widget.BaseWidget

	clip     *container.Clip
	onScroll func(*fyne.ScrollEvent)
}

func newScrollClip(content fyne.CanvasObject, onScroll func(*fyne.ScrollEvent)) *scrollClip {
	clip := &scrollClip{
		clip:     container.NewClip(content),
		onScroll: onScroll,
	}
	clip.ExtendBaseWidget(clip)
	return clip
}

func (c *scrollClip) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.clip)
}

func (c *scrollClip) Scrolled(event *fyne.ScrollEvent) {
	if c.onScroll != nil {
		c.onScroll(event)
	}
}
