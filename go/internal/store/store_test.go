package store

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smartpaste/internal/config"
)

func testStore(t *testing.T) (*Store, config.Paths) {
	t.Helper()
	t.Setenv("SMARTPASTE_DATA_DIRECTORY", t.TempDir())
	paths, err := config.ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	dataStore, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dataStore.Close)
	return dataStore, paths
}

func TestSensitiveContentIsRejected(t *testing.T) {
	dataStore, _ := testStore(t)

	for _, text := range []string{
		"4111 1111 1111 1111",
		"123-45-6789",
		"-----BEGIN RSA PRIVATE KEY-----",
		"密码: hunter2",
	} {
		if dataStore.AddText(text, "Test") {
			t.Fatalf("expected sensitive text to be rejected: %q", text)
		}
	}
	if !dataStore.AddText("contact me at hello@example.com", "Test") {
		t.Fatal("expected normal text to be stored")
	}
	if got := len(dataStore.Clips()); got != 2 {
		t.Fatalf("expected main clip plus indexed e-mail, got %d", got)
	}
}

func TestIndexValuesExtractsCommonFields(t *testing.T) {
	text := "Contact alice@example.com or +1 (415) 555-0134\nhttps://example.com/docs"
	values := IndexValues(text)
	required := []string{
		"alice@example.com",
		"https://example.com/docs",
	}
	for _, expected := range required {
		found := false
		for _, value := range values {
			if value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %q in %v", expected, values)
		}
	}
}

func TestImageScalingAndRemoval(t *testing.T) {
	dataStore, paths := testStore(t)

	source := image.NewRGBA(image.Rect(0, 0, 5000, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 5000; x++ {
			source.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatal(err)
	}
	if !dataStore.AddImage(buffer.Bytes(), "Preview") {
		t.Fatal("expected image to be stored")
	}

	clips := dataStore.Clips()
	if len(clips) != 1 || !clips[0].IsImage() {
		t.Fatalf("unexpected clips: %+v", clips)
	}
	if clips[0].ImagePixelWidth != 4096 {
		t.Fatalf("expected width 4096, got %d", clips[0].ImagePixelWidth)
	}
	if clips[0].ImagePixelHeight != 81 {
		t.Fatalf("expected height 81, got %d", clips[0].ImagePixelHeight)
	}
	imagePath := filepath.Join(paths.ImagesDir, clips[0].ImageFileName)
	if _, err := os.Stat(imagePath); err != nil {
		t.Fatalf("stored image missing: %v", err)
	}
	if thumb, ok := dataStore.Thumbnail(clips[0]); !ok || thumb == nil {
		t.Fatal("expected a thumbnail")
	}

	dataStore.Remove(clips)
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("expected image file to be removed, stat err=%v", err)
	}
}

func TestPersistenceUsesFoundationDateNumber(t *testing.T) {
	dataStore, paths := testStore(t)
	if !dataStore.AddText("persisted value", "Notes") {
		t.Fatal("expected text to be stored")
	}
	dataStore.Close()

	raw, err := os.ReadFile(paths.ClipsFile)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) == 0 {
		t.Fatal("expected a persisted clip")
	}
	if _, ok := decoded[0]["createdAt"].(float64); !ok {
		t.Fatalf("createdAt should be a reference-date number, got %T", decoded[0]["createdAt"])
	}
}

func TestLoadAcceptsBothDateFormats(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SMARTPASTE_DATA_DIRECTORY", dir)
	paths, err := config.ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	data := `[
	  {"id":"a","text":"number","sourceApp":"Test","createdAt":814343491.99},
	  {"id":"b","text":"text","sourceApp":"Test","createdAt":"2026-10-22T12:00:00Z"}
	]`
	if err := os.WriteFile(paths.ClipsFile, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	dataStore, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer dataStore.Close()

	clips := dataStore.Clips()
	if len(clips) != 2 {
		t.Fatalf("expected 2 clips, got %d", len(clips))
	}
	if clips[0].CreatedAt.IsZero() || clips[1].CreatedAt.IsZero() {
		t.Fatal("expected both timestamps to parse")
	}
	want := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	if !clips[1].CreatedAt.Equal(want) {
		t.Fatalf("unexpected RFC3339 date: %v", clips[1].CreatedAt)
	}
}
