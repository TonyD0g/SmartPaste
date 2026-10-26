package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const AppName = "Smart Paste"

type Paths struct {
	DataDir      string
	ImagesDir    string
	LanguagesDir string
	ClipsFile    string
	SettingsFile string
	LogFile      string
}

func ResolvePaths() (Paths, error) {
	dataDir := os.Getenv("SMARTPASTE_DATA_DIRECTORY")
	if dataDir == "" {
		base, err := defaultDataDir()
		if err != nil {
			return Paths{}, err
		}
		dataDir = filepath.Join(base, AppName)
	}

	imagesDir := filepath.Join(dataDir, "Images")
	languagesDir := filepath.Join(dataDir, "Languages")

	for _, dir := range []string{dataDir, imagesDir, languagesDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Paths{}, fmt.Errorf("create %s: %w", dir, err)
		}
	}

	return Paths{
		DataDir:      dataDir,
		ImagesDir:    imagesDir,
		LanguagesDir: languagesDir,
		ClipsFile:    filepath.Join(dataDir, "clips.json"),
		SettingsFile: filepath.Join(dataDir, "settings.json"),
		LogFile:      logPath(),
	}, nil
}

func defaultDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		if dir := os.Getenv("APPDATA"); dir != "" {
			return dir, nil
		}
		return os.UserConfigDir()
	default:
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			return dir, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}

func logPath() string {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, "Library", "Logs", "Smart Paste.log")
		}
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("APPDATA")
		}
		if base != "" {
			return filepath.Join(base, AppName, "logs", "smart-paste.log")
		}
	default:
		base := os.Getenv("XDG_STATE_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				base = filepath.Join(home, ".local", "state")
			}
		}
		if base != "" {
			return filepath.Join(base, "smart-paste", "smart-paste.log")
		}
	}
	return filepath.Join(os.TempDir(), "smart-paste.log")
}
