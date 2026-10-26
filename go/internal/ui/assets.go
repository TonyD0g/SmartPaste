package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/tray.svg
var traySVG []byte

//go:embed assets/appicon.svg
var appIconSVG []byte

func trayIcon() fyne.Resource {
	return fyne.NewStaticResource("smart-paste-tray.svg", traySVG)
}

func appIcon() fyne.Resource {
	return fyne.NewStaticResource("smart-paste.svg", appIconSVG)
}
