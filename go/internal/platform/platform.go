package platform

import (
	"errors"
	"strings"

	"smartpaste/internal/model"
)

var (
	ErrUnsupported     = errors.New("not supported on this platform")
	ErrRestartRequired = errors.New("restart required to change the global shortcut")
)

type Frontmost struct {
	Name   string
	PID    int
	Handle uintptr
	ID     string
}

type Permission struct {
	Key         string
	LabelKey    string
	Description string
	Allowed     bool
	Required    bool
	SettingsURL string
}

// FocusedField returns the currently focused editable control, or nil when
// there is no usable focus target. Implementations filter password fields; the
// shared check below applies the same keyword filter as the Swift app.
func FocusedField() (*model.Field, error) {
	field, err := focusedField()
	if err != nil {
		return nil, err
	}
	if field == nil {
		return nil, nil
	}
	if looksSensitive(field) {
		return nil, nil
	}
	return field, nil
}

var sensitiveWords = []string{
	"password", "passcode", "secure", "credit card", "card number",
	"cvv", "cvc", "security code", "one-time", "one time", "otp",
	"social security", "ssn",
	"密码", "验证码", "信用卡", "安全码",
}

func looksSensitive(field *model.Field) bool {
	haystack := strings.ToLower(field.Label + " " + field.Role + " " + field.Value + " " + field.App)
	for _, word := range sensitiveWords {
		if strings.Contains(haystack, word) {
			return true
		}
	}
	return false
}

func ClipboardSupported() bool { return clipboardInitErr == nil }
func FocusSupported() bool     { return focusSupported() }
func PasteSupported() bool     { return pasteSupported() }

func FrontmostApp() Frontmost                   { return frontmostApp() }
func ActivateApp(app Frontmost) error           { return activateApp(app) }
func SimulatePaste() error                      { return simulatePaste() }
func Permissions() []Permission                 { return permissions() }
func OpenPermissionSettings(p Permission) error { return openPermissionSettings(p) }
func RequestPermissions()                       { requestPermissions() }
func SystemLanguage() string                    { return systemLanguage() }
func IsWayland() bool                           { return isWayland() }
