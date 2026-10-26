package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	maximumFileSize = 2 * 1024 * 1024
	exampleFileName = "example.json"
	defaultCode     = "en"
	chineseCode     = "zh-Hans"
)

var codePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{0,31}$`)

type Pack struct {
	Code    string
	Name    string
	Strings map[string]string
	BuiltIn bool
}

type FileError struct {
	Key  string
	Args []any
}

func (e *FileError) Error() string { return e.Key }

type Manager struct {
	mu      sync.RWMutex
	dir     string
	packs   map[string]Pack
	current Pack
	changes chan struct{}
}

func New(languagesDir, preferred string) *Manager {
	manager := &Manager{
		dir:     languagesDir,
		packs:   make(map[string]Pack),
		changes: make(chan struct{}, 1),
	}
	for _, pack := range []Pack{englishPack(), chinesePack()} {
		manager.packs[strings.ToLower(pack.Code)] = pack
	}

	_ = writeExampleIfNeeded(languagesDir)
	for _, pack := range loadExternal(languagesDir) {
		manager.packs[strings.ToLower(pack.Code)] = pack
	}

	manager.current = manager.resolve(preferred)
	if manager.current.Code == "" {
		manager.current = manager.resolve(systemLanguageCode())
	}
	if manager.current.Code == "" {
		manager.current = manager.packs[defaultCode]
	}
	return manager
}

func (m *Manager) CurrentCode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current.Code
}

func (m *Manager) CurrentName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current.Name
}

func (m *Manager) Available() []Pack {
	m.mu.RLock()
	defer m.mu.RUnlock()

	builtIn := []Pack{m.packs[defaultCode], m.packs[strings.ToLower(chineseCode)]}
	seen := map[string]struct{}{defaultCode: {}, strings.ToLower(chineseCode): {}}
	external := make([]Pack, 0, len(m.packs))
	for key, pack := range m.packs {
		if _, ok := seen[key]; ok {
			continue
		}
		external = append(external, pack)
	}
	sort.Slice(external, func(i, j int) bool {
		return strings.ToLower(external[i].Name) < strings.ToLower(external[j].Name)
	})
	return append(builtIn, external...)
}

func (m *Manager) Changes() <-chan struct{} { return m.changes }

func (m *Manager) Select(code string) bool {
	m.mu.Lock()
	pack := m.resolveLocked(code)
	if pack.Code == "" {
		m.mu.Unlock()
		return false
	}
	if pack.Code == m.current.Code {
		m.mu.Unlock()
		return true
	}
	m.current = pack
	m.mu.Unlock()
	m.notify()
	return true
}

func (m *Manager) Install(path string) (Pack, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, &FileError{Key: "language.error.unreadable"}
	}
	if len(raw) > maximumFileSize {
		return Pack{}, &FileError{Key: "language.error.tooLarge"}
	}
	pack, err := parsePack(raw)
	if err != nil {
		return Pack{}, err
	}

	destination := filepath.Join(m.dir, pack.Code+".json")
	if err := writeFileAtomic(destination, raw); err != nil {
		return Pack{}, &FileError{Key: "language.error.unreadable"}
	}

	m.mu.Lock()
	m.packs[strings.ToLower(pack.Code)] = pack
	if strings.EqualFold(pack.Code, m.current.Code) {
		m.current = pack
	}
	m.mu.Unlock()
	m.notify()
	return pack, nil
}

func (m *Manager) L(key string, args ...any) string {
	m.mu.RLock()
	template, ok := m.current.Strings[key]
	if !ok {
		template = m.packs[defaultCode].Strings[key]
	}
	m.mu.RUnlock()
	if template == "" {
		template = key
	}
	return render(template, args)
}

func (m *Manager) Error(err error) string {
	var fileErr *FileError
	if errors.As(err, &fileErr) {
		return m.L(fileErr.Key, fileErr.Args...)
	}
	return err.Error()
}

func (m *Manager) notify() {
	select {
	case m.changes <- struct{}{}:
	default:
	}
}

func (m *Manager) resolve(code string) Pack {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resolveLocked(code)
}

func (m *Manager) resolveLocked(code string) Pack {
	code = strings.TrimSpace(code)
	if code == "" {
		return Pack{}
	}
	if pack, ok := m.packs[strings.ToLower(code)]; ok {
		return pack
	}
	if strings.HasPrefix(strings.ToLower(code), "zh") {
		return m.packs[strings.ToLower(chineseCode)]
	}
	if strings.HasPrefix(strings.ToLower(code), "en") {
		return m.packs[defaultCode]
	}
	return Pack{}
}

func loadExternal(dir string) []Pack {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	packs := make([]Pack, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil || len(raw) > maximumFileSize {
			continue
		}
		pack, err := parsePack(raw)
		if err != nil {
			continue
		}
		packs = append(packs, pack)
	}
	return packs
}

func parsePack(raw []byte) (Pack, error) {
	var root struct {
		Code    string            `json:"code"`
		Name    string            `json:"name"`
		Strings map[string]string `json:"strings"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return Pack{}, &FileError{Key: "language.error.invalidFormat"}
	}

	code := strings.TrimSpace(root.Code)
	if code == "" {
		return Pack{}, &FileError{Key: "language.error.missingField", Args: []any{"code"}}
	}
	if !codePattern.MatchString(code) {
		return Pack{}, &FileError{Key: "language.error.invalidCode"}
	}
	if strings.EqualFold(code, defaultCode) || strings.EqualFold(code, chineseCode) {
		return Pack{}, &FileError{Key: "language.error.reservedCode", Args: []any{code}}
	}

	name := strings.TrimSpace(root.Name)
	if name == "" {
		return Pack{}, &FileError{Key: "language.error.missingField", Args: []any{"name"}}
	}
	if len(root.Strings) == 0 {
		return Pack{}, &FileError{Key: "language.error.emptyTranslations"}
	}

	return Pack{Code: code, Name: name, Strings: root.Strings}, nil
}

func writeExampleIfNeeded(dir string) error {
	path := filepath.Join(dir, exampleFileName)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	root := map[string]any{
		"_instructions": "This is an example language pack. Change \"code\" to your language code, " +
			"change \"name\" to the name shown in the Language menu, then translate every value " +
			"in \"strings\". Keep the placeholders %@ and %ld unchanged. Load your edited file " +
			"with Language > Load Language File..., or place it in this folder and restart Smart Paste.",
		"code":    "example",
		"name":    "Example (English Template)",
		"strings": englishStrings,
	}
	raw, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(raw, '\n'))
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lang-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func systemLanguageCode() string {
	for _, key := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(os.Getenv(key))
		if value == "" {
			continue
		}
		value = strings.SplitN(value, ":", 2)[0]
		value = strings.SplitN(value, ".", 2)[0]
		if strings.HasPrefix(value, "zh") {
			return chineseCode
		}
		if strings.HasPrefix(value, "en") {
			return defaultCode
		}
	}
	return platformLanguageCode()
}

// render supports the placeholders used by the original language packs:
// %@ (string), %ld / %d (integer) and %% (literal percent).
func render(template string, args []any) string {
	if !strings.Contains(template, "%") {
		return template
	}

	var out strings.Builder
	argument := 0
	for i := 0; i < len(template); i++ {
		if template[i] != '%' {
			out.WriteByte(template[i])
			continue
		}
		if i+1 < len(template) && template[i+1] == '%' {
			out.WriteByte('%')
			i++
			continue
		}

		consumed := 0
		if i+1 < len(template) && template[i+1] == '@' {
			consumed = 2
		} else if i+2 < len(template) && template[i+1] == 'l' && template[i+2] == 'd' {
			consumed = 3
		} else if i+1 < len(template) && template[i+1] == 'd' {
			consumed = 2
		}
		if consumed == 0 {
			out.WriteByte('%')
			continue
		}

		if argument < len(args) {
			out.WriteString(fmt.Sprint(args[argument]))
			argument++
		} else {
			out.WriteString(template[i : i+consumed])
		}
		i += consumed - 1
	}
	return out.String()
}
