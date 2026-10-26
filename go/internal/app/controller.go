package app

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"smartpaste/internal/config"
	"smartpaste/internal/i18n"
	"smartpaste/internal/model"
	"smartpaste/internal/platform"
	"smartpaste/internal/secret"
	"smartpaste/internal/store"
	"smartpaste/internal/suggest"
)

const (
	pollInterval         = 240 * time.Millisecond
	hotkeyRequestWindow  = 1600 * time.Millisecond
	regularRequestWindow = 13 * time.Second
	textRestoreDelay     = 220 * time.Millisecond
	imageRestoreDelay    = 420 * time.Millisecond
)

type Callbacks struct {
	ShowPanel    func()
	HidePanel    func()
	PanelVisible func() bool
	Refresh      func()
	Status       func(string)
}

type Controller struct {
	Store     *store.Store
	Settings  *config.SettingsStore
	I18n      *i18n.Manager
	Secrets   *secret.Store
	Client    *suggest.Client
	Clipboard *platform.Clipboard
	Callbacks Callbacks

	mu             sync.Mutex
	focused        *model.Field
	focusKey       string
	suggestion     *model.Clip
	ranked         []string
	isRequesting   bool
	pasteWhenReady bool
	generation     int

	focusWarned   bool
	keyWarned     bool
	lastFrontmost string
}

func New(
	store *store.Store,
	settings *config.SettingsStore,
	localization *i18n.Manager,
	secrets *secret.Store,
	clipboard *platform.Clipboard,
) *Controller {
	return &Controller{
		Store:     store,
		Settings:  settings,
		I18n:      localization,
		Secrets:   secrets,
		Client:    suggest.New(),
		Clipboard: clipboard,
	}
}

func (c *Controller) Start(ctx context.Context) {
	go c.watchClipboard(ctx)
	go c.pollFocus(ctx)
}

func (c *Controller) watchClipboard(ctx context.Context) {
	c.Clipboard.Watch(
		ctx,
		func() bool { return c.Settings.Get().CapturePaused },
		func(text string, _ bool) {
			if c.Store.AddText(text, c.sourceApp()) {
				c.notifyRefresh()
			}
		},
		func(image []byte, _ bool) {
			if c.Store.AddImage(image, c.sourceApp()) {
				c.notifyRefresh()
			}
		},
	)
}

func (c *Controller) sourceApp() string {
	name := platform.FrontmostApp().Name
	if strings.TrimSpace(name) == "" {
		return "Unknown"
	}
	return name
}

func (c *Controller) pollFocus(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.Callbacks.PanelVisible != nil && c.Callbacks.PanelVisible() {
				continue
			}
			c.observeFocus()
		}
	}
}

func (c *Controller) observeFocus() {
	field, err := platform.FocusedField()
	if err != nil {
		if !c.focusWarned {
			c.focusWarned = true
			c.notifyStatus(c.I18n.L("status.focus.unavailable"))
		}
		c.clearFocused()
		return
	}
	if field == nil {
		frontmost := platform.FrontmostApp().Name
		if frontmost != c.lastFrontmost {
			c.lastFrontmost = frontmost
			log.Printf("focus: none, frontmost=%q", frontmost)
		}
		c.clearFocused()
		return
	}
	c.focusWarned = false
	field = c.normalize(field)
	key := fieldKey(field)

	c.mu.Lock()
	same := c.focused != nil && c.focusKey == key
	c.mu.Unlock()
	if same {
		return
	}

	log.Printf("focused field: role=%q label=%q app=%q", field.Role, field.Label, field.App)

	c.mu.Lock()
	c.focused = field
	c.focusKey = key
	c.suggestion = nil
	c.ranked = nil
	c.isRequesting = false
	c.pasteWhenReady = false
	c.generation++
	c.mu.Unlock()

	c.notifyRefresh()
	c.beginSuggestion(field, false)
}

func (c *Controller) clearFocused() {
	c.mu.Lock()
	if c.focused == nil {
		c.mu.Unlock()
		return
	}
	c.focused = nil
	c.focusKey = ""
	c.suggestion = nil
	c.ranked = nil
	c.isRequesting = false
	c.pasteWhenReady = false
	c.generation++
	c.mu.Unlock()
	c.notifyRefresh()
}

func (c *Controller) HandleHotkey() {
	if c.Callbacks.PanelVisible != nil && c.Callbacks.PanelVisible() {
		if c.Callbacks.HidePanel != nil {
			c.Callbacks.HidePanel()
		}
		return
	}

	field, err := platform.FocusedField()
	log.Printf("hotkey: field=%v err=%v frontmost=%q textClips=%d", field != nil, err, platform.FrontmostApp().Name, len(c.Store.TextClips()))
	if err != nil || field == nil || len(c.Store.TextClips()) == 0 {
		if c.Callbacks.ShowPanel != nil {
			c.Callbacks.ShowPanel()
		}
		return
	}

	field = c.normalize(field)
	key := fieldKey(field)

	c.mu.Lock()
	same := c.focused != nil && c.focusKey == key
	suggestion := c.suggestion
	requesting := c.isRequesting
	c.mu.Unlock()

	if same && suggestion != nil {
		c.pasteClip(*suggestion)
		return
	}
	if same && requesting {
		c.mu.Lock()
		c.pasteWhenReady = true
		c.mu.Unlock()
		return
	}

	c.beginSuggestion(field, true)
}

func (c *Controller) beginSuggestion(field *model.Field, pasteAfterResponse bool) {
	c.mu.Lock()
	c.focused = field
	c.focusKey = fieldKey(field)
	c.suggestion = nil
	c.ranked = nil
	c.isRequesting = true
	c.pasteWhenReady = pasteAfterResponse
	c.generation++
	generation := c.generation
	c.mu.Unlock()

	c.notifyRefresh()

	candidates := suggest.CandidateValues(c.Store.TextClips())
	if len(candidates) == 0 {
		c.apply(generation, field, model.Outcome{NoMatch: true})
		return
	}

	apiKey, err := c.Secrets.Get()
	if err != nil || strings.TrimSpace(apiKey) == "" {
		if err != nil && !c.keyWarned {
			c.keyWarned = true
			c.notifyStatus(c.I18n.L("error.keyring"))
		}
		c.apply(generation, field, model.Outcome{Failed: true})
		return
	}
	c.keyWarned = false

	timeout := regularRequestWindow
	if pasteAfterResponse {
		timeout = hotkeyRequestWindow
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	outcome := c.Client.Suggest(ctx, apiKey, *field, candidates)
	c.apply(generation, field, outcome)
}

func (c *Controller) apply(generation int, field *model.Field, outcome model.Outcome) {
	c.mu.Lock()
	if generation != c.generation || c.focused == nil || fieldKey(c.focused) != fieldKey(field) {
		c.mu.Unlock()
		return
	}

	c.isRequesting = false
	shouldPaste := c.pasteWhenReady
	c.pasteWhenReady = false

	var toPaste *model.Clip
	fallback := false

	switch {
	case outcome.NoMatch || outcome.Failed || outcome.Clip == nil:
		c.ranked = nil
		if shouldPaste {
			fallback = true
		}
	default:
		c.ranked = outcome.RankedTexts
		if outcome.Confidence >= suggest.MinimumConfidence {
			if shouldPaste {
				toPaste = outcome.Clip
			} else {
				c.suggestion = outcome.Clip
			}
		} else if shouldPaste {
			fallback = true
		}
	}
	c.mu.Unlock()

	c.notifyRefresh()
	if toPaste != nil {
		c.pasteClip(*toPaste)
		return
	}
	if fallback {
		c.fallbackPaste(field)
	}
}

func (c *Controller) fallbackPaste(expected *model.Field) {
	if expected != nil {
		current, err := platform.FocusedField()
		if err != nil || current == nil || fieldKey(c.normalize(current)) != fieldKey(expected) {
			return
		}
	}
	_ = platform.SimulatePaste()
}

func (c *Controller) pasteClip(clip model.Clip) {
	snapshot := c.Clipboard.Snapshot()

	if clip.IsImage() {
		data, err := c.Store.Image(clip)
		if err != nil {
			c.notifyStatus(c.I18n.L("error.clipboard", err.Error()))
			return
		}
		if err := c.Clipboard.WriteImage(data); err != nil {
			c.notifyStatus(c.I18n.L("error.clipboard", err.Error()))
			return
		}
	} else {
		if err := c.Clipboard.WriteText(clip.Text); err != nil {
			c.notifyStatus(c.I18n.L("error.clipboard", err.Error()))
			return
		}
	}

	time.Sleep(25 * time.Millisecond)
	_ = platform.SimulatePaste()

	delay := textRestoreDelay
	if clip.IsImage() {
		delay = imageRestoreDelay
	}
	time.AfterFunc(delay, func() {
		c.Clipboard.Restore(snapshot)
	})

	c.mu.Lock()
	c.suggestion = nil
	c.isRequesting = false
	c.pasteWhenReady = false
	c.generation++
	c.mu.Unlock()
	c.notifyRefresh()
}

// PasteClip is used by the history panel after it has hidden itself and
// restored focus to the app the user was working in.
func (c *Controller) PasteClip(clip model.Clip, previous *platform.Frontmost) {
	if previous != nil {
		_ = platform.ActivateApp(*previous)
		time.Sleep(140 * time.Millisecond)
	}
	c.pasteClip(clip)
}

func (c *Controller) RankedTexts() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.ranked))
	copy(out, c.ranked)
	return out
}

func (c *Controller) Suggestion() *model.Clip {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.suggestion == nil {
		return nil
	}
	clone := *c.suggestion
	return &clone
}

func (c *Controller) Reset() {
	c.mu.Lock()
	c.focused = nil
	c.focusKey = ""
	c.suggestion = nil
	c.ranked = nil
	c.isRequesting = false
	c.pasteWhenReady = false
	c.generation++
	c.mu.Unlock()
	c.notifyRefresh()
}

func (c *Controller) ResetKeyWarning() {
	c.mu.Lock()
	c.keyWarned = false
	c.mu.Unlock()
}

func (c *Controller) normalize(field *model.Field) *model.Field {
	clone := *field
	if strings.TrimSpace(clone.Label) == "" {
		clone.Label = c.I18n.L("field.textField")
	}
	if strings.TrimSpace(clone.App) == "" {
		clone.App = "Unknown"
	}
	return &clone
}

func fieldKey(field *model.Field) string {
	return field.Role + "\x00" + field.Label + "\x00" + field.App
}

func (c *Controller) notifyRefresh() {
	if c.Callbacks.Refresh != nil {
		c.Callbacks.Refresh()
	}
}

func (c *Controller) notifyStatus(message string) {
	if c.Callbacks.Status != nil && message != "" {
		c.Callbacks.Status(message)
	}
}
