//go:build windows

package platform

import "golang.design/x/hotkey"

func hotkeyModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}
}

func hotkeyKey(mode string) hotkey.Key {
	if normalizeMode(mode) == "shiftJ" {
		return hotkey.KeyJ
	}
	return hotkey.KeyV
}

func shortcutUsesCommand() bool { return false }

func CanLiveSwitchHotkey() bool { return true }
