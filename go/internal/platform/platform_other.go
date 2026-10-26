//go:build !darwin && !windows && !linux && !freebsd && !openbsd && !netbsd

package platform

import (
	"smartpaste/internal/model"
)

func focusedField() (*model.Field, error) { return nil, ErrUnsupported }
func frontmostApp() Frontmost             { return Frontmost{} }
func activateApp(Frontmost) error         { return ErrUnsupported }
func simulatePaste() error                { return ErrUnsupported }
func focusSupported() bool                { return false }
func pasteSupported() bool                { return false }
func permissions() []Permission           { return nil }
func openPermissionSettings(Permission) error {
	return ErrUnsupported
}
func requestPermissions()    {}
func systemLanguage() string { return "" }
func isWayland() bool        { return false }

func CanLiveSwitchHotkey() bool { return false }
