package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"

	"smartpaste/internal/app"
	"smartpaste/internal/config"
	"smartpaste/internal/i18n"
	"smartpaste/internal/platform"
	"smartpaste/internal/secret"
	"smartpaste/internal/store"
)

func TestRenderSetupDialog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SMARTPASTE_DATA_DIRECTORY", dir)
	paths, err := config.ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	settings, err := config.OpenSettings(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	localization := i18n.New(paths.LanguagesDir, "zh-Hans")
	dataStore, err := store.Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer dataStore.Close()

	testApp := test.NewApp()
	testApp.Settings().SetTheme(newHUDTheme())
	controller := app.New(dataStore, settings, localization, secret.New(), platform.NewClipboard())
	application := New(testApp, controller, dataStore, settings, localization, secret.New(), paths)
	application.panel = NewPanel(testApp, controller, dataStore, settings, localization)
	application.panel.Show()
	application.showSetup(nil)

	capturable, ok := application.setupWindow.Canvas().(interface{ Capture() image.Image })
	if !ok {
		t.Fatalf("canvas is not capturable")
	}
	out, err := os.Create(filepath.Join(os.TempDir(), "smartpaste-setup-zh.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, capturable.Capture()); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", out.Name())
}
