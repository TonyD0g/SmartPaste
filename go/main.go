package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"smartpaste/internal/app"
	"smartpaste/internal/config"
	"smartpaste/internal/i18n"
	"smartpaste/internal/platform"
	"smartpaste/internal/secret"
	"smartpaste/internal/store"
	"smartpaste/internal/ui"
)

const version = "0.2.0-go"

func main() {
	paths, err := config.ResolvePaths()
	if err != nil {
		log.Fatalf("resolve data directory: %v", err)
	}
	setupLogging(paths.LogFile)
	log.Printf("Smart Paste %s starting on %s/%s", version, runtime.GOOS, runtime.GOARCH)

	listener, err := net.Listen("tcp", "127.0.0.1:47119")
	if err != nil {
		log.Printf("another Smart Paste instance is already running: %v", err)
		return
	}
	defer listener.Close()

	settings, err := config.OpenSettings(paths.SettingsFile)
	if err != nil {
		log.Fatalf("open settings: %v", err)
	}
	localization := i18n.New(paths.LanguagesDir, settings.Get().Language)

	fyneApp := fyneapp.New()
	if err := platform.InitClipboard(); err != nil {
		log.Printf("clipboard initialization failed: %v", err)
	}

	dataStore, err := store.Open(paths)
	if err != nil {
		log.Fatalf("open history: %v", err)
	}

	secrets := secret.New()
	controller := app.New(dataStore, settings, localization, secrets, platform.NewClipboard())
	application := ui.New(fyneApp, controller, dataStore, settings, localization, secrets, paths)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals
		fyne.Do(application.Quit)
	}()

	application.Run()
}

func setupLogging(path string) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
}
