//go:build darwin

package i18n

import (
	"os/exec"
	"strings"
)

func platformLanguageCode() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	locale := strings.ToLower(strings.TrimSpace(string(out)))
	if strings.HasPrefix(locale, "zh") {
		return chineseCode
	}
	if strings.HasPrefix(locale, "en") {
		return defaultCode
	}
	return ""
}
