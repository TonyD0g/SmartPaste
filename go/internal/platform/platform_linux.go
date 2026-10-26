//go:build linux || freebsd || openbsd || netbsd

package platform

import (
	"errors"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"smartpaste/internal/model"
)

const (
	atspiFocusedState = 1 << 11
	atspiRoleComboBox = 12
	atspiRolePassword = 40
	atspiRoleText     = 61
	atspiRoleEntry    = 79
	atspiRoleDocument = 82
	maxTreeDepth      = 9
	maxTreeNodes      = 600
	focusCacheTTL     = 250 * time.Millisecond
)

type atspiRef struct {
	Name string
	Path dbus.ObjectPath
}

type atspiNode struct {
	ref   atspiRef
	depth int
	app   string
}

type focusCache struct {
	mu    sync.Mutex
	at    time.Time
	field *model.Field
}

var cachedFocus focusCache

var (
	atspiOnce sync.Once
	atspiConn *dbus.Conn
	atspiErr  error
)

func atspiConnection() (*dbus.Conn, error) {
	atspiOnce.Do(func() {
		session, err := dbus.SessionBus()
		if err != nil {
			atspiErr = err
			return
		}
		busObject := session.Object("org.a11y.Bus", dbus.ObjectPath("/org/a11y/bus"))
		var address string
		if err := busObject.Call("org.a11y.Bus.GetAddress", 0).Store(&address); err != nil {
			atspiErr = err
			return
		}
		if address == "" {
			atspiErr = ErrUnsupported
			return
		}
		conn, err := dbus.Dial(address)
		if err != nil {
			atspiErr = err
			return
		}
		if err := conn.Auth(nil); err != nil {
			conn.Close()
			atspiErr = err
			return
		}
		if err := conn.Hello(); err != nil {
			conn.Close()
			atspiErr = err
			return
		}
		atspiConn = conn
	})
	return atspiConn, atspiErr
}

func focusedField() (*model.Field, error) {
	cachedFocus.mu.Lock()
	if cachedFocus.field != nil && time.Since(cachedFocus.at) < focusCacheTTL {
		field := cachedFocus.field
		cachedFocus.mu.Unlock()
		return field, nil
	}
	cachedFocus.mu.Unlock()

	conn, err := atspiConnection()
	if err != nil {
		return nil, err
	}

	root := conn.Object("org.a11y.atspi.Registry", dbus.ObjectPath("/org/a11y/atspi/accessible/root"))
	children, err := atspiChildren(root)
	if err != nil {
		return nil, err
	}

	activeName := strings.ToLower(frontmostApp().Name)
	sort.SliceStable(children, func(i, j int) bool {
		left := activeName != "" && strings.Contains(strings.ToLower(children[i].Name), activeName)
		right := activeName != "" && strings.Contains(strings.ToLower(children[j].Name), activeName)
		return left && !right
	})

	deadline := time.Now().Add(1200 * time.Millisecond)
	budget := maxTreeNodes
	queue := make([]atspiNode, 0, len(children))
	for _, child := range children {
		queue = append(queue, atspiNode{ref: child, depth: 0, app: child.Name})
	}

	for len(queue) > 0 && budget > 0 && time.Now().Before(deadline) {
		node := queue[0]
		queue = queue[1:]
		budget--

		object := conn.Object(node.ref.Name, node.ref.Path)
		role := atspiRole(object)
		states := atspiStates(object)
		focused := len(states) > 0 && states[0]&atspiFocusedState != 0

		if focused && editableRole(role) && role != atspiRolePassword {
			field := atspiField(conn, node, object, role)
			if field != nil {
				cachedFocus.mu.Lock()
				cachedFocus.field = field
				cachedFocus.at = time.Now()
				cachedFocus.mu.Unlock()
				return field, nil
			}
		}

		if node.depth >= maxTreeDepth {
			continue
		}
		childRefs, err := atspiChildren(object)
		if err != nil {
			continue
		}
		for _, child := range childRefs {
			queue = append(queue, atspiNode{ref: child, depth: node.depth + 1, app: node.app})
		}
	}

	cachedFocus.mu.Lock()
	cachedFocus.field = nil
	cachedFocus.at = time.Now()
	cachedFocus.mu.Unlock()
	return nil, nil
}

func atspiField(conn *dbus.Conn, node atspiNode, object dbus.BusObject, role uint32) *model.Field {
	name := atspiString(object, "org.a11y.atspi.Accessible.GetName")
	description := atspiString(object, "org.a11y.atspi.Accessible.GetDescription")
	roleName := atspiString(object, "org.a11y.atspi.Accessible.GetLocalizedRoleName")
	if roleName == "" {
		roleName = atspiString(object, "org.a11y.atspi.Accessible.GetRoleName")
	}

	value := ""
	textObject := conn.Object(node.ref.Name, node.ref.Path)
	if err := textObject.Call("org.a11y.atspi.Text.GetText", 0, int32(0), int32(-1)).Store(&value); err != nil {
		value = ""
	}

	parts := make([]string, 0, 3)
	for _, part := range []string{name, description, roleName} {
		if text := strings.TrimSpace(part); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "Text field")
	}

	return &model.Field{
		Label: strings.Join(parts, " · "),
		Role:  roleName,
		App:   node.app,
		Value: value,
	}
}

func editableRole(role uint32) bool {
	switch role {
	case atspiRoleComboBox, atspiRoleText, atspiRoleEntry, atspiRoleDocument:
		return true
	default:
		return false
	}
}

func atspiChildren(object dbus.BusObject) ([]atspiRef, error) {
	var children []atspiRef
	if err := object.Call("org.a11y.atspi.Accessible.GetChildren", 0).Store(&children); err != nil {
		return nil, err
	}
	return children, nil
}

func atspiRole(object dbus.BusObject) uint32 {
	var role uint32
	if err := object.Call("org.a11y.atspi.Accessible.GetRole", 0).Store(&role); err != nil {
		return 0
	}
	return role
}

func atspiStates(object dbus.BusObject) []uint32 {
	var states []uint32
	if err := object.Call("org.a11y.atspi.Accessible.GetState", 0).Store(&states); err != nil {
		return nil
	}
	return states
}

func atspiString(object dbus.BusObject, method string) string {
	var value string
	if err := object.Call(method, 0).Store(&value); err != nil {
		return ""
	}
	return value
}

func frontmostApp() Frontmost {
	id := commandOutput("xdotool", "getactivewindow")
	if id == "" {
		return Frontmost{}
	}
	name := commandOutput("xdotool", "getwindowclassname", id)
	if name == "" {
		name = commandOutput("xdotool", "getwindowname", id)
	}
	pid, _ := strconv.Atoi(commandOutput("xdotool", "getwindowpid", id))
	return Frontmost{Name: name, PID: pid, ID: id}
}

func activateApp(app Frontmost) error {
	if app.ID != "" {
		if err := exec.Command("xdotool", "windowactivate", "--sync", app.ID).Run(); err == nil {
			return nil
		}
	}
	if app.Name != "" {
		if err := exec.Command("wmctrl", "-a", app.Name).Run(); err == nil {
			return nil
		}
	}
	return ErrUnsupported
}

func simulatePaste() error {
	candidates := [][]string{
		{"xdotool", "key", "--clearmodifiers", "ctrl+v"},
		{"wtype", "-M", "ctrl", "-k", "v", "-m", "ctrl"},
		{"ydotool", "key", "29:1", "47:1", "47:0", "29:0"},
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		if err := exec.Command(candidate[0], candidate[1:]...).Run(); err == nil {
			return nil
		}
	}
	return errors.New("no paste tool found (install xdotool, wtype or ydotool)")
}

func focusSupported() bool { return true }
func pasteSupported() bool { return true }

func permissions() []Permission {
	return []Permission{
		{Key: "clipboard", LabelKey: "setup.permission.clipboard", Allowed: ClipboardSupported(), Required: true},
		{Key: "focus", LabelKey: "setup.permission.focus", Allowed: atspiAvailable(), Required: false},
		{Key: "hotkey", LabelKey: "setup.permission.hotkey", Allowed: !isWayland(), Required: false},
	}
}

func atspiAvailable() bool {
	_, err := atspiConnection()
	return err == nil
}

func openPermissionSettings(Permission) error { return ErrUnsupported }
func requestPermissions()                     {}
func systemLanguage() string                  { return "" }

func isWayland() bool {
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") {
		return true
	}
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

func commandOutput(name string, args ...string) string {
	if _, err := exec.LookPath(name); err != nil {
		return ""
	}
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
