package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type hudTheme struct {
	base fyne.Theme
}

func newHUDTheme() fyne.Theme {
	return &hudTheme{base: theme.DarkTheme()}
}

func (t *hudTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x1C, G: 0x1C, B: 0x1E, A: 0xFF}
	case theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		return color.NRGBA{R: 0x24, G: 0x24, B: 0x26, A: 0xFF}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2E, A: 0xFF}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xF2, G: 0xF2, B: 0xF7, A: 0xFF}
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return color.NRGBA{R: 0x8E, G: 0x8E, B: 0x93, A: 0xFF}
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return color.NRGBA{R: 0x0A, G: 0x84, B: 0xFF, A: 0xFF}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2E, A: 0xFF}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x0A, G: 0x84, B: 0xFF, A: 0x38}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x3A, G: 0x3A, B: 0x3C, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2E, A: 0xFF}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66}
	}
	return t.base.Color(name, theme.VariantDark)
}

func (t *hudTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *hudTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *hudTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameText:
		return 13
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 6
	}
	return t.base.Size(name)
}
