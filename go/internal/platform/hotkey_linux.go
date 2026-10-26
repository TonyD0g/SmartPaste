//go:build linux || freebsd || openbsd || netbsd

package platform

import "golang.design/x/hotkey"

// The X11 implementation of golang.design/x/hotkey only exposes its key and
// modifier constants when built with cgo. These are the stable X11 keysym and
// modifier-mask values for V, J, Shift and Control.
func hotkeyModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.Modifier(1 << 2), hotkey.Modifier(1 << 0)}
}

func hotkeyKey(mode string) hotkey.Key {
	if normalizeMode(mode) == "shiftJ" {
		return hotkey.Key(0x006a)
	}
	return hotkey.Key(0x0076)
}

func shortcutUsesCommand() bool { return false }

func CanLiveSwitchHotkey() bool { return true }
