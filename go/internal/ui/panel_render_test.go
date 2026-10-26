package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"smartpaste/internal/app"
	"smartpaste/internal/config"
	"smartpaste/internal/i18n"
	"smartpaste/internal/platform"
	"smartpaste/internal/secret"
	"smartpaste/internal/store"
)

func TestRenderPanel(t *testing.T) {
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
	dataStore.AddText("https://example.com/docs", "Safari")
	dataStore.AddText("hello@example.com", "Mail")
	dataStore.AddText("一段很长的中文剪贴板内容，用来检查中文字体回退是否正常。", "Notes")
	var imageBuffer bytes.Buffer
	gradient := image.NewRGBA(image.Rect(0, 0, 160, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 160; x++ {
			gradient.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}
	if err := png.Encode(&imageBuffer, gradient); err != nil {
		t.Fatal(err)
	}
	dataStore.AddImage(imageBuffer.Bytes(), "Preview")

	secrets := secret.New()
	controller := app.New(dataStore, settings, localization, secrets, platform.NewClipboard())

	testApp := test.NewApp()
	testApp.Settings().SetTheme(newHUDTheme())
	panel := NewPanel(testApp, controller, dataStore, settings, localization)
	panel.Show()

	canvas := panel.window.Canvas()
	capturable, ok := canvas.(interface{ Capture() image.Image })
	if !ok {
		t.Fatalf("canvas is not capturable: %T", canvas)
	}
	img := capturable.Capture()
	out, err := os.Create(filepath.Join(os.TempDir(), "smartpaste-panel-zh.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", out.Name())

	panel.search.SetText("hello@")
	if len(panel.results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(panel.results))
	}
	panel.search.SetText("")
	if len(panel.results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(panel.results))
	}
	panel.list.MoveSelection(1)
	if panel.list.Selected() != 1 {
		t.Fatalf("expected selection 1, got %d", panel.list.Selected())
	}
	panel.list.Select(0)
	panel.list.handleScroll(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -1)})
	if panel.list.offset != rowStep {
		t.Fatalf("expected offset %v after scrolling down, got %v", rowStep, panel.list.offset)
	}
	panel.list.handleScroll(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 1)})
	if panel.list.offset != 0 {
		t.Fatalf("expected offset 0 after scrolling up, got %v", panel.list.offset)
	}
}
