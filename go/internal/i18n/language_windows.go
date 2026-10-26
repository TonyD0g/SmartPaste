//go:build windows

package i18n

import "golang.org/x/sys/windows"

func platformLanguageCode() string {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")
	langID, _, _ := proc.Call()
	if langID == 0 {
		return ""
	}
	switch langID & 0x3ff {
	case 0x04:
		return chineseCode
	case 0x09:
		return defaultCode
	default:
		return ""
	}
}
