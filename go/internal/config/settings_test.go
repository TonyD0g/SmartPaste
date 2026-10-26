package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsOverridesDataDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SMARTPASTE_DATA_DIRECTORY", dir)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.DataDir != dir {
		t.Fatalf("unexpected data dir: %s", paths.DataDir)
	}
	for _, candidate := range []string{paths.ImagesDir, paths.LanguagesDir} {
		if info, err := os.Stat(candidate); err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s: %v", candidate, err)
		}
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := OpenSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Get().Shortcut != ShortcutShiftV {
		t.Fatalf("unexpected default shortcut: %s", store.Get().Shortcut)
	}

	err = store.Update(func(settings *Settings) {
		settings.Shortcut = ShortcutShiftJ
		settings.Language = "zh-Hans"
		settings.CapturePaused = true
		settings.HasShownFirstLaunch = true
		settings.TimelineWidth = 700
	})
	if err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Get()
	if got.Shortcut != ShortcutShiftJ || got.Language != "zh-Hans" ||
		!got.CapturePaused || !got.HasShownFirstLaunch || got.TimelineWidth != 700 {
		t.Fatalf("unexpected settings: %+v", got)
	}
}

func TestInvalidSettingsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenSettings(path); err == nil {
		t.Fatal("expected an error for malformed settings")
	}
}
