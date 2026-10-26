package i18n

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	return New(dir, "zh-Hans")
}

func TestRenderPlaceholders(t *testing.T) {
	got := render("Hello %@, %ld items, %d%%", []any{"Alice", 3, 25})
	if got != "Hello Alice, 3 items, 25%" {
		t.Fatalf("unexpected render: %q", got)
	}
	if got := render("No placeholders", nil); got != "No placeholders" {
		t.Fatalf("unexpected render: %q", got)
	}
	if got := render("Missing %@", nil); got != "Missing %@" {
		t.Fatalf("unexpected render: %q", got)
	}
}

func TestBuiltInPacksAndExample(t *testing.T) {
	manager := newTestManager(t)
	if manager.CurrentCode() != chineseCode {
		t.Fatalf("expected zh-Hans, got %s", manager.CurrentCode())
	}
	if got := manager.L("menu.open", "⌘⇧V"); got != "打开 Smart Paste（⌘⇧V）" {
		t.Fatalf("unexpected translation: %q", got)
	}

	if _, err := os.Stat(filepath.Join(manager.dir, exampleFileName)); err != nil {
		t.Fatalf("expected example language pack: %v", err)
	}

	found := map[string]bool{}
	for _, pack := range manager.Available() {
		found[pack.Code] = true
	}
	if !found["en"] || !found[chineseCode] {
		t.Fatalf("missing built-in packs: %v", found)
	}
}

func TestInstallExternalPack(t *testing.T) {
	manager := newTestManager(t)
	source := filepath.Join(t.TempDir(), "fr.json")
	raw := `{"code":"fr","name":"Français","strings":{"menu.open":"Ouvrir (%@)"}}`
	if err := os.WriteFile(source, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	pack, err := manager.Install(source)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Code != "fr" {
		t.Fatalf("unexpected pack: %+v", pack)
	}
	if !manager.Select("fr") {
		t.Fatal("expected selection to succeed")
	}
	if got := manager.L("menu.open", "⌘⇧V"); got != "Ouvrir (⌘⇧V)" {
		t.Fatalf("unexpected translation: %q", got)
	}
	// Missing keys fall back to the English source.
	if got := manager.L("timeline.delete"); got != englishStrings["timeline.delete"] {
		t.Fatalf("expected English fallback, got %q", got)
	}
}

func TestInvalidPacks(t *testing.T) {
	manager := newTestManager(t)
	cases := []struct {
		name string
		raw  string
		key  string
	}{
		{"reserved", `{"code":"en","name":"English","strings":{"a":"b"}}`, "language.error.reservedCode"},
		{"invalid-code", `{"code":"1fr","name":"Bad","strings":{"a":"b"}}`, "language.error.invalidCode"},
		{"empty", `{"code":"fr","name":"Fr","strings":{}}`, "language.error.emptyTranslations"},
		{"missing-name", `{"code":"fr","strings":{"a":"b"}}`, "language.error.missingField"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), testCase.name+".json")
			if err := os.WriteFile(path, []byte(testCase.raw), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := manager.Install(path)
			if err == nil {
				t.Fatal("expected an error")
			}
			var fileErr *FileError
			if !errors.As(err, &fileErr) {
				t.Fatalf("unexpected error type: %T", err)
			}
			if fileErr.Key != testCase.key {
				t.Fatalf("expected %s, got %s", testCase.key, fileErr.Key)
			}
		})
	}
}
