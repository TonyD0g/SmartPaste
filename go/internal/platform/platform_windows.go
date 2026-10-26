//go:build windows

package platform

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"

	"smartpaste/internal/model"
)

const (
	uiaNamePropertyID                 = 30005
	uiaControlTypePropertyID          = 30003
	uiaLocalizedControlTypePropertyID = 30004
	uiaHelpTextPropertyID             = 30013
	uiaIsPasswordPropertyID           = 30019
	uiaValueValuePropertyID           = 30045
	uiaAriaRolePropertyID             = 30101
	uiaComboBoxControlTypeID          = 50003
	uiaEditControlTypeID              = 50004
	uiaDocumentControlTypeID          = 50030
)

var (
	clsidCUIAutomation = ole.NewGUID("{FF48DBA4-60EF-4201-AA87-54103EEF594E}")
	iidIUIAutomation   = ole.NewGUID("{30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}")

	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procKeybdEvent          = user32.NewProc("keybd_event")
)

const (
	swRestore     = 9
	vkControl     = 0x11
	vkV           = 0x56
	keyEventKeyUp = 0x0002
)

type iuiautomation struct {
	ole.IUnknown
}

type iuiautomationVtbl struct {
	ole.IUnknownVtbl
	CompareElements   uintptr
	CompareRuntimeIds uintptr
	GetRootElement    uintptr
	ElementFromHandle uintptr
	ElementFromPoint  uintptr
	GetFocusedElement uintptr
}

func (v *iuiautomation) VTable() *iuiautomationVtbl {
	return (*iuiautomationVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iuiautomation) focusedElement() (*iuiautomationElement, error) {
	var element *iuiautomationElement
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetFocusedElement,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&element)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return element, nil
}

type iuiautomationElement struct {
	ole.IUnknown
}

type iuiautomationElementVtbl struct {
	ole.IUnknownVtbl
	SetFocus                uintptr
	GetRuntimeId            uintptr
	FindFirst               uintptr
	FindAll                 uintptr
	FindFirstBuildCache     uintptr
	FindAllBuildCache       uintptr
	BuildUpdatedCache       uintptr
	GetCurrentPropertyValue uintptr
}

func (v *iuiautomationElement) VTable() *iuiautomationElementVtbl {
	return (*iuiautomationElementVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iuiautomationElement) property(id int32) (any, error) {
	var variant ole.VARIANT
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetCurrentPropertyValue,
		uintptr(unsafe.Pointer(v)),
		uintptr(id),
		uintptr(unsafe.Pointer(&variant)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	defer variant.Clear()
	return variant.Value(), nil
}

func (v *iuiautomationElement) stringProperty(id int32) string {
	value, err := v.property(id)
	if err != nil || value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func (v *iuiautomationElement) intProperty(id int32) int32 {
	value, err := v.property(id)
	if err != nil || value == nil {
		return 0
	}
	number, _ := value.(int32)
	return number
}

func (v *iuiautomationElement) boolProperty(id int32) bool {
	value, err := v.property(id)
	if err != nil {
		return false
	}
	flag, _ := value.(bool)
	return flag
}

type uiaRequest struct {
	reply chan uiaReply
}

type uiaReply struct {
	field *model.Field
	err   error
}

type uiaWorker struct {
	requests chan uiaRequest
}

var (
	uiaOnce   sync.Once
	uiaShared *uiaWorker
)

func startUIA() *uiaWorker {
	uiaOnce.Do(func() {
		worker := &uiaWorker{requests: make(chan uiaRequest, 4)}
		uiaShared = worker
		go worker.run()
	})
	return uiaShared
}

func (w *uiaWorker) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_ = ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	defer ole.CoUninitialize()

	var automation *iuiautomation
	initErr := error(nil)
	unknown, err := ole.CreateInstance(clsidCUIAutomation, iidIUIAutomation)
	if err != nil {
		initErr = err
	} else {
		automation = (*iuiautomation)(unsafe.Pointer(unknown))
	}

	for request := range w.requests {
		if initErr != nil {
			request.reply <- uiaReply{err: initErr}
			continue
		}
		request.reply <- uiaReply{field: readFocusedField(automation)}
	}
}

func focusedField() (*model.Field, error) {
	worker := startUIA()
	reply := make(chan uiaReply, 1)
	select {
	case worker.requests <- uiaRequest{reply: reply}:
	case <-time.After(2 * time.Second):
		return nil, errors.New("UI Automation request timed out")
	}
	select {
	case result := <-reply:
		return result.field, result.err
	case <-time.After(2 * time.Second):
		return nil, errors.New("UI Automation response timed out")
	}
}

func readFocusedField(automation *iuiautomation) *model.Field {
	element, err := automation.focusedElement()
	if err != nil || element == nil {
		return nil
	}
	defer element.Release()

	controlType := element.intProperty(uiaControlTypePropertyID)
	switch controlType {
	case uiaEditControlTypeID, uiaDocumentControlTypeID, uiaComboBoxControlTypeID:
	default:
		return nil
	}
	if element.boolProperty(uiaIsPasswordPropertyID) {
		return nil
	}

	role := controlTypeName(controlType)
	name := element.stringProperty(uiaNamePropertyID)
	help := element.stringProperty(uiaHelpTextPropertyID)
	localized := element.stringProperty(uiaLocalizedControlTypePropertyID)
	ariaRole := element.stringProperty(uiaAriaRolePropertyID)
	value := element.stringProperty(uiaValueValuePropertyID)

	parts := make([]string, 0, 4)
	for _, part := range []string{name, help, ariaRole, localized} {
		if text := strings.TrimSpace(part); text != "" {
			parts = append(parts, text)
		}
	}

	return &model.Field{
		Label: strings.Join(parts, " · "),
		Role:  role,
		App:   frontmostApp().Name,
		Value: value,
	}
}

func controlTypeName(id int32) string {
	switch id {
	case uiaComboBoxControlTypeID:
		return "ComboBox"
	case uiaEditControlTypeID:
		return "Edit"
	case uiaDocumentControlTypeID:
		return "Document"
	default:
		return fmt.Sprintf("Control(%d)", id)
	}
}

func frontmostApp() Frontmost {
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		return Frontmost{}
	}
	var pid uint32
	windows.GetWindowThreadProcessId(hwnd, &pid)
	return Frontmost{
		Name:   processName(pid),
		PID:    int(pid),
		Handle: uintptr(hwnd),
	}
}

func processName(pid uint32) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
		return ""
	}
	full := windows.UTF16ToString(buffer[:size])
	base := filepath.Base(full)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func activateApp(app Frontmost) error {
	if app.Handle == 0 {
		return ErrUnsupported
	}
	procShowWindow.Call(app.Handle, swRestore)
	ok, _, callErr := procSetForegroundWindow.Call(app.Handle)
	if ok == 0 {
		return fmt.Errorf("SetForegroundWindow: %v", callErr)
	}
	return nil
}

func simulatePaste() error {
	procKeybdEvent.Call(vkControl, 0, 0, 0)
	time.Sleep(12 * time.Millisecond)
	procKeybdEvent.Call(vkV, 0, 0, 0)
	time.Sleep(12 * time.Millisecond)
	procKeybdEvent.Call(vkV, 0, keyEventKeyUp, 0)
	procKeybdEvent.Call(vkControl, 0, keyEventKeyUp, 0)
	return nil
}

func focusSupported() bool { return true }
func pasteSupported() bool { return true }

func permissions() []Permission { return nil }
func openPermissionSettings(Permission) error {
	return ErrUnsupported
}
func requestPermissions()    {}
func systemLanguage() string { return "" }
func isWayland() bool        { return false }
