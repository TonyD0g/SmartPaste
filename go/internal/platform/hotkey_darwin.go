//go:build darwin

package platform

import "golang.design/x/hotkey"

func hotkeyModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}
}

func hotkeyKey(mode string) hotkey.Key {
	if normalizeMode(mode) == "shiftJ" {
		return hotkey.KeyJ
	}
	return hotkey.KeyV
}

func shortcutUsesCommand() bool { return true }

func CanLiveSwitchHotkey() bool { return false }
