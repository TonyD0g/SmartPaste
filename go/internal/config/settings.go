package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type ShortcutMode string

const (
	ShortcutShiftV ShortcutMode = "shiftV"
	ShortcutShiftJ ShortcutMode = "shiftJ"
)

type Settings struct {
	Shortcut            ShortcutMode `json:"shortcut"`
	Language            string       `json:"language"`
	CapturePaused       bool         `json:"clipboardCapturePaused"`
	HasShownFirstLaunch bool         `json:"hasShownFirstLaunchUI"`
	TimelineWidth       float64      `json:"timelineWidth"`
}

func DefaultSettings() Settings {
	return Settings{
		Shortcut:      ShortcutShiftV,
		TimelineWidth: 560,
	}
}

type SettingsStore struct {
	mu   sync.RWMutex
	path string
	data Settings
}

func OpenSettings(path string) (*SettingsStore, error) {
	store := &SettingsStore{path: path, data: DefaultSettings()}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, err
	}
	if store.data.Shortcut != ShortcutShiftJ {
		store.data.Shortcut = ShortcutShiftV
	}
	if store.data.TimelineWidth < 440 {
		store.data.TimelineWidth = 560
	}
	return store, nil
}

func (s *SettingsStore) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *SettingsStore) Update(fn func(*Settings)) error {
	s.mu.Lock()
	fn(&s.data)
	snapshot := s.data
	s.mu.Unlock()
	return writeJSONAtomic(s.path, snapshot)
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
