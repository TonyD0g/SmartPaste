package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"smartpaste/internal/config"
	"smartpaste/internal/model"
)

const (
	maxTextRunes = 20_000
	debounceSave = 180 * time.Millisecond
)

type Store struct {
	mu     sync.RWMutex
	paths  config.Paths
	clips  []model.Clip
	thumbs sync.Map

	saveCh    chan struct{}
	changes   chan struct{}
	done      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func Open(paths config.Paths) (*Store, error) {
	store := &Store{
		paths:   paths,
		saveCh:  make(chan struct{}, 1),
		changes: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	store.wg.Add(1)
	go store.saveLoop()
	return store, nil
}

func (s *Store) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		s.wg.Wait()
		s.saveNow()
	})
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.paths.ClipsFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var decoded []model.Clip
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}

	kept := decoded[:0]
	for _, clip := range decoded {
		if clip.ImageFileName == "" {
			kept = append(kept, clip)
			continue
		}
		if !safeImageName(clip.ImageFileName) {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.paths.ImagesDir, clip.ImageFileName)); err == nil {
			kept = append(kept, clip)
		}
	}
	s.clips = kept
	if len(kept) != len(decoded) {
		s.scheduleSave()
	}
	return nil
}

func (s *Store) Clips() []model.Clip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Clip, len(s.clips))
	copy(out, s.clips)
	return out
}

func (s *Store) TextClips() []model.Clip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Clip, 0, len(s.clips))
	for _, clip := range s.clips {
		if !clip.IsImage() {
			out = append(out, clip)
		}
	}
	return out
}

func (s *Store) Changes() <-chan struct{} { return s.changes }

func (s *Store) notify() {
	select {
	case s.changes <- struct{}{}:
	default:
	}
}

func (s *Store) AddText(rawText, sourceApp string) bool {
	text := strings.TrimSpace(rawText)
	if text == "" || len([]rune(text)) > maxTextRunes || LooksSensitive(text) {
		return false
	}

	s.mu.Lock()
	if len(s.clips) > 0 && !s.clips[0].IsImage() && s.clips[0].Text == text {
		s.mu.Unlock()
		return false
	}

	createdAt := model.NewTimestamp(time.Now())
	values := IndexValues(text)
	inserted := make([]model.Clip, 0, 1+len(values))
	inserted = append(inserted, model.Clip{
		ID:            newUUID(),
		Text:          text,
		SourceApp:     sourceApp,
		CreatedAt:     createdAt,
		IndexedValues: values,
	})
	for _, value := range values {
		if value == text {
			continue
		}
		inserted = append(inserted, model.Clip{
			ID:            newUUID(),
			Text:          value,
			SourceApp:     sourceApp,
			CreatedAt:     createdAt,
			IndexedValues: []string{value},
		})
	}
	s.clips = append(inserted, s.clips...)
	s.mu.Unlock()

	s.scheduleSave()
	s.notify()
	return true
}

func (s *Store) AddImage(data []byte, sourceApp string) bool {
	pngData, width, height, err := prepareImage(data)
	if err != nil {
		return false
	}

	name := strings.ToLower(newUUID()) + ".png"
	if err := writeFileAtomic(filepath.Join(s.paths.ImagesDir, name), pngData); err != nil {
		return false
	}

	s.mu.Lock()
	s.clips = append([]model.Clip{{
		ID:               newUUID(),
		SourceApp:        sourceApp,
		CreatedAt:        model.NewTimestamp(time.Now()),
		ImageFileName:    name,
		ImagePixelWidth:  width,
		ImagePixelHeight: height,
	}}, s.clips...)
	s.mu.Unlock()

	s.scheduleSave()
	s.notify()
	return true
}

func (s *Store) Image(clip model.Clip) ([]byte, error) {
	if !clip.IsImage() || !safeImageName(clip.ImageFileName) {
		return nil, errors.New("clip has no image")
	}
	return os.ReadFile(filepath.Join(s.paths.ImagesDir, clip.ImageFileName))
}

func (s *Store) Thumbnail(clip model.Clip) (image.Image, bool) {
	if !clip.IsImage() || !safeImageName(clip.ImageFileName) {
		return nil, false
	}
	if cached, ok := s.thumbs.Load(clip.ImageFileName); ok {
		return cached.(image.Image), true
	}
	data, err := os.ReadFile(filepath.Join(s.paths.ImagesDir, clip.ImageFileName))
	if err != nil {
		return nil, false
	}
	img, err := thumbnail(data)
	if err != nil {
		return nil, false
	}
	s.thumbs.Store(clip.ImageFileName, img)
	return img, true
}

func (s *Store) Remove(clips []model.Clip) {
	if len(clips) == 0 {
		return
	}
	ids := make(map[string]struct{}, len(clips))
	for _, clip := range clips {
		ids[clip.ID] = struct{}{}
	}
	removedImages := make([]string, 0, len(clips))

	s.mu.Lock()
	kept := s.clips[:0]
	for _, clip := range s.clips {
		if _, remove := ids[clip.ID]; remove {
			if clip.IsImage() && safeImageName(clip.ImageFileName) {
				removedImages = append(removedImages, clip.ImageFileName)
			}
			continue
		}
		kept = append(kept, clip)
	}
	s.clips = kept
	s.mu.Unlock()

	for _, name := range removedImages {
		_ = os.Remove(filepath.Join(s.paths.ImagesDir, name))
		s.thumbs.Delete(name)
	}
	s.scheduleSave()
	s.notify()
}

func (s *Store) Clear() {
	s.mu.Lock()
	removed := s.clips
	s.clips = nil
	s.mu.Unlock()

	for _, clip := range removed {
		if clip.IsImage() && safeImageName(clip.ImageFileName) {
			_ = os.Remove(filepath.Join(s.paths.ImagesDir, clip.ImageFileName))
			s.thumbs.Delete(clip.ImageFileName)
		}
	}
	s.scheduleSave()
	s.notify()
}

func (s *Store) scheduleSave() {
	select {
	case s.saveCh <- struct{}{}:
	default:
	}
}

func (s *Store) saveLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.done:
			return
		case <-s.saveCh:
			timer := time.NewTimer(debounceSave)
			select {
			case <-timer.C:
			case <-s.done:
				timer.Stop()
				return
			}
			s.saveNow()
		}
	}
}

func (s *Store) saveNow() {
	s.mu.RLock()
	snapshot := make([]model.Clip, len(s.clips))
	copy(snapshot, s.clips)
	s.mu.RUnlock()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	_ = writeFileAtomic(s.paths.ClipsFile, data)
}

func newUUID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return hex.EncodeToString(raw[:])
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], raw[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], raw[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], raw[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], raw[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], raw[10:16])
	return string(dst)
}
