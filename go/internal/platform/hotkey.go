package platform

import (
	"strings"
	"sync"

	"golang.design/x/hotkey"
)

type HotkeyManager struct {
	mu      sync.Mutex
	hk      *hotkey.Hotkey
	cancel  chan struct{}
	events  chan struct{}
	done    chan struct{}
	onPress func()
	once    sync.Once
	retired []*hotkey.Hotkey
}

func NewHotkeyManager(onPress func()) *HotkeyManager {
	manager := &HotkeyManager{
		events:  make(chan struct{}, 4),
		done:    make(chan struct{}),
		onPress: onPress,
	}
	go manager.consume()
	return manager
}

func (m *HotkeyManager) consume() {
	for {
		select {
		case <-m.events:
			if m.onPress != nil {
				m.onPress()
			}
		case <-m.done:
			return
		}
	}
}

func (m *HotkeyManager) Register(mode string) error {
	m.mu.Lock()
	hasExisting := m.hk != nil
	m.mu.Unlock()
	if hasExisting && !CanLiveSwitchHotkey() {
		return ErrRestartRequired
	}
	m.Unregister()

	hk := hotkey.New(hotkeyModifiers(), hotkeyKey(mode))
	if err := hk.Register(); err != nil {
		return err
	}

	cancel := make(chan struct{})
	m.mu.Lock()
	m.hk = hk
	m.cancel = cancel
	m.mu.Unlock()

	go func() {
		for {
			select {
			case <-hk.Keydown():
				select {
				case m.events <- struct{}{}:
				default:
				}
			case <-cancel:
				return
			}
		}
	}()
	return nil
}

func (m *HotkeyManager) Unregister() {
	m.mu.Lock()
	hk := m.hk
	cancel := m.cancel
	m.hk = nil
	m.cancel = nil
	m.mu.Unlock()

	if cancel != nil {
		close(cancel)
	}
	if hk != nil {
		_ = hk.Unregister()
	}
}

func (m *HotkeyManager) Close() {
	m.mu.Lock()
	if m.hk != nil {
		// Keep the tap alive until the process exits. Unregistering from a
		// non-main thread can deadlock inside the hotkey library on macOS.
		m.retired = append(m.retired, m.hk)
	}
	cancel := m.cancel
	m.hk = nil
	m.cancel = nil
	m.mu.Unlock()

	if cancel != nil {
		close(cancel)
	}
	m.once.Do(func() { close(m.done) })
}

func ShortcutLabel(mode string) string {
	key := "V"
	if normalizeMode(mode) == "shiftJ" {
		key = "J"
	}
	if shortcutUsesCommand() {
		return "⌘⇧" + key
	}
	return "Ctrl+Shift+" + key
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "shiftj", "commandj", "ctrlshiftj":
		return "shiftJ"
	default:
		return "shiftV"
	}
}
