package platform

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"golang.design/x/clipboard"
)

var (
	clipboardOnce    sync.Once
	clipboardInitErr error
)

func InitClipboard() error {
	clipboardOnce.Do(func() {
		clipboardInitErr = clipboard.Init()
	})
	return clipboardInitErr
}

type Snapshot struct {
	Text  []byte
	Image []byte
}

type Clipboard struct {
	mu  sync.Mutex
	own map[[32]byte]time.Time
}

func NewClipboard() *Clipboard {
	return &Clipboard{own: make(map[[32]byte]time.Time)}
}

func (c *Clipboard) markOwn(items ...[]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for hash, at := range c.own {
		if now.Sub(at) > 5*time.Second {
			delete(c.own, hash)
		}
	}
	for _, item := range items {
		if len(item) == 0 {
			continue
		}
		c.own[sha256.Sum256(item)] = now
	}
}

func (c *Clipboard) isOwn(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	hash := sha256.Sum256(data)
	c.mu.Lock()
	defer c.mu.Unlock()
	at, ok := c.own[hash]
	if !ok {
		return false
	}
	if time.Since(at) > 5*time.Second {
		delete(c.own, hash)
		return false
	}
	return true
}

func (c *Clipboard) WriteText(text string) error {
	if clipboardInitErr != nil {
		return clipboardInitErr
	}
	c.markOwn([]byte(text))
	_, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(text))
	return err
}

func (c *Clipboard) WriteImage(png []byte) error {
	if clipboardInitErr != nil {
		return clipboardInitErr
	}
	c.markOwn(png)
	_, err := clipboard.Write(context.Background(), clipboard.FmtImage, png)
	return err
}

func (c *Clipboard) ReadText() (string, error) {
	if clipboardInitErr != nil {
		return "", clipboardInitErr
	}
	data, err := clipboard.Read(context.Background(), clipboard.FmtText)
	if errors.Is(err, clipboard.ErrNoData) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Clipboard) ReadImage() ([]byte, error) {
	if clipboardInitErr != nil {
		return nil, clipboardInitErr
	}
	data, err := clipboard.Read(context.Background(), clipboard.FmtImage)
	if errors.Is(err, clipboard.ErrNoData) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Clipboard) Snapshot() Snapshot {
	text, _ := c.ReadText()
	image, _ := c.ReadImage()
	return Snapshot{Text: []byte(text), Image: image}
}

func (c *Clipboard) Restore(snapshot Snapshot) {
	if clipboardInitErr != nil {
		return
	}
	items := make([]clipboard.Item, 0, 2)
	if len(snapshot.Text) > 0 {
		items = append(items, clipboard.Item{Format: clipboard.FmtText, Bytes: snapshot.Text})
	}
	if len(snapshot.Image) > 0 {
		items = append(items, clipboard.Item{Format: clipboard.FmtImage, Bytes: snapshot.Image})
	}
	if len(items) == 0 {
		return
	}
	for _, item := range items {
		c.markOwn(item.Bytes)
	}
	_, _ = clipboard.WriteAll(context.Background(), itemOptions(items)...)
}

func itemOptions(items []clipboard.Item) []clipboard.Option {
	options := make([]clipboard.Option, 0, len(items))
	for _, item := range items {
		options = append(options, item)
	}
	return options
}

func (c *Clipboard) Watch(
	ctx context.Context,
	paused func() bool,
	onText func(string, bool),
	onImage func([]byte, bool),
) {
	if clipboardInitErr != nil {
		return
	}
	for data := range clipboard.Watch(ctx, clipboard.FmtText, clipboard.FmtImage) {
		if data.Sensitive || c.isOwn(data.Bytes) {
			continue
		}
		if paused != nil && paused() {
			continue
		}
		switch data.Format {
		case clipboard.FmtText:
			if onText != nil {
				onText(string(data.Bytes), data.Sensitive)
			}
		case clipboard.FmtImage:
			if onImage != nil {
				onImage(data.Bytes, data.Sensitive)
			}
		}
	}
}
