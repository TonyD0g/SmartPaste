//go:build darwin

package platform

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics -framework CoreFoundation -framework AppKit
#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>

int sp_activate_pid(int pid);

typedef struct {
	char *role;
	char *title;
	char *description;
	char *help;
	char *placeholder;
	char *subrole;
	char *value;
	char *app;
	int secure;
} sp_focus_result;

static char *sp_copy_cfstring(CFStringRef value) {
	if (value == NULL) return NULL;
	CFIndex length = CFStringGetLength(value);
	CFIndex maximum = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
	char *buffer = (char *)malloc((size_t)maximum);
	if (buffer == NULL) return NULL;
	if (!CFStringGetCString(value, buffer, maximum, kCFStringEncodingUTF8)) {
		free(buffer);
		return NULL;
	}
	return buffer;
}

static char *sp_copy_attribute(AXUIElementRef element, CFStringRef attribute) {
	CFTypeRef value = NULL;
	if (AXUIElementCopyAttributeValue(element, attribute, &value) != kAXErrorSuccess || value == NULL) {
		return NULL;
	}
	char *result = NULL;
	if (CFGetTypeID(value) == CFStringGetTypeID()) {
		result = sp_copy_cfstring((CFStringRef)value);
	}
	CFRelease(value);
	return result;
}

static AXUIElementRef sp_focused_element(void) {
	AXUIElementRef system = AXUIElementCreateSystemWide();
	CFTypeRef focused = NULL;
	if (AXUIElementCopyAttributeValue(system, kAXFocusedUIElementAttribute, &focused) == kAXErrorSuccess && focused != NULL) {
		CFRelease(system);
		return (AXUIElementRef)focused;
	}
	CFRelease(system);

	pid_t pid = 0;
	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID
	);
	if (windows != NULL) {
		for (CFIndex i = 0; i < CFArrayGetCount(windows); i++) {
			CFDictionaryRef info = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
			CFNumberRef layer = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowLayer);
			int layerValue = 0;
			if (layer != NULL) CFNumberGetValue(layer, kCFNumberIntType, &layerValue);
			if (layerValue != 0) continue;
			CFNumberRef ownerPID = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowOwnerPID);
			if (ownerPID != NULL) {
				CFNumberGetValue(ownerPID, kCFNumberIntType, &pid);
				break;
			}
		}
		CFRelease(windows);
	}
	if (pid == 0) return NULL;

	AXUIElementRef application = AXUIElementCreateApplication(pid);
	focused = NULL;
	if (AXUIElementCopyAttributeValue(application, kAXFocusedUIElementAttribute, &focused) == kAXErrorSuccess && focused != NULL) {
		CFRelease(application);
		return (AXUIElementRef)focused;
	}
	CFRelease(application);
	return NULL;
}

static char *sp_frontmost_app_name(void) {
	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID
	);
	if (windows == NULL) return NULL;
	char *name = NULL;
	for (CFIndex i = 0; i < CFArrayGetCount(windows); i++) {
		CFDictionaryRef info = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		CFNumberRef layer = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowLayer);
		int layerValue = 0;
		if (layer != NULL) CFNumberGetValue(layer, kCFNumberIntType, &layerValue);
		if (layerValue != 0) continue;
		CFStringRef owner = (CFStringRef)CFDictionaryGetValue(info, kCGWindowOwnerName);
		if (owner != NULL) {
			name = sp_copy_cfstring(owner);
			break;
		}
	}
	CFRelease(windows);
	return name;
}

static int sp_frontmost_pid(void) {
	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID
	);
	if (windows == NULL) return 0;
	int pid = 0;
	for (CFIndex i = 0; i < CFArrayGetCount(windows); i++) {
		CFDictionaryRef info = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		CFNumberRef layer = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowLayer);
		int layerValue = 0;
		if (layer != NULL) CFNumberGetValue(layer, kCFNumberIntType, &layerValue);
		if (layerValue != 0) continue;
		CFNumberRef ownerPID = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowOwnerPID);
		if (ownerPID != NULL) CFNumberGetValue(ownerPID, kCFNumberIntType, &pid);
		break;
	}
	CFRelease(windows);
	return pid;
}

static sp_focus_result sp_read_focus(void) {
	sp_focus_result result;
	memset(&result, 0, sizeof(result));
	AXUIElementRef element = sp_focused_element();
	if (element == NULL) return result;

	result.role = sp_copy_attribute(element, kAXRoleAttribute);
	result.title = sp_copy_attribute(element, kAXTitleAttribute);
	result.description = sp_copy_attribute(element, kAXDescriptionAttribute);
	result.help = sp_copy_attribute(element, kAXHelpAttribute);
	result.placeholder = sp_copy_attribute(element, kAXPlaceholderValueAttribute);
	result.subrole = sp_copy_attribute(element, kAXSubroleAttribute);
	result.value = sp_copy_attribute(element, kAXValueAttribute);
	result.app = sp_frontmost_app_name();
	if (result.role != NULL && strcmp(result.role, "AXSecureTextField") == 0) {
		result.secure = 1;
	}
	if (result.subrole != NULL && strcmp(result.subrole, "AXSecureTextField") == 0) {
		result.secure = 1;
	}
	CFRelease(element);
	return result;
}

static void sp_free_focus_result(sp_focus_result *result) {
	if (result == NULL) return;
	free(result->role);
	free(result->title);
	free(result->description);
	free(result->help);
	free(result->placeholder);
	free(result->subrole);
	free(result->value);
	free(result->app);
	memset(result, 0, sizeof(*result));
}

static int sp_is_main_thread(void) { return pthread_main_np() ? 1 : 0; }

static int sp_ax_trusted(void) { return AXIsProcessTrusted() ? 1 : 0; }
static int sp_input_allowed(void) { return CGPreflightListenEventAccess() ? 1 : 0; }

static void sp_request_permissions(void) {
	CFStringRef keys[] = { CFSTR("AXTrustedCheckOptionPrompt") };
	const void *values[] = { kCFBooleanTrue };
	CFDictionaryRef options = CFDictionaryCreate(
		kCFAllocatorDefault,
		(const void **)keys,
		values,
		1,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	if (options != NULL) {
		AXIsProcessTrustedWithOptions(options);
		CFRelease(options);
	}
	CGRequestListenEventAccess();
}

static void sp_send_paste(void) {
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateCombinedSessionState);
	CGEventRef down = CGEventCreateKeyboardEvent(source, (CGKeyCode)9, true);
	CGEventSetFlags(down, kCGEventFlagMaskCommand);
	CGEventPost(kCGHIDEventTap, down);
	CGEventRef up = CGEventCreateKeyboardEvent(source, (CGKeyCode)9, false);
	CGEventSetFlags(up, kCGEventFlagMaskCommand);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down);
	CFRelease(up);
	if (source != NULL) CFRelease(source);
}

static char *sp_locale_id(void) {
	CFLocaleRef locale = CFLocaleCopyCurrent();
	if (locale == NULL) return NULL;
	CFStringRef identifier = CFLocaleGetIdentifier(locale);
	char *result = sp_copy_cfstring(identifier);
	CFRelease(locale);
	return result;
}
*/
import "C"

import (
	"os/exec"
	"strings"
	"unsafe"

	"smartpaste/internal/model"
)

func focusedField() (*model.Field, error) {
	result := C.sp_read_focus()
	defer C.sp_free_focus_result(&result)

	role := C.GoString(result.role)
	switch role {
	case "AXTextField", "AXTextArea", "AXComboBox", "AXSearchField":
	default:
		return nil, nil
	}
	if result.secure != 0 {
		return nil, nil
	}

	labelParts := make([]string, 0, 5)
	for _, value := range []*C.char{result.title, result.description, result.help, result.placeholder} {
		if text := strings.TrimSpace(C.GoString(value)); text != "" {
			labelParts = append(labelParts, text)
		}
	}
	if subrole := strings.ToLower(C.GoString(result.subrole)); subrole != "" {
		labelParts = append(labelParts, subrole)
	}

	return &model.Field{
		Label: strings.Join(labelParts, " · "),
		Role:  role,
		App:   C.GoString(result.app),
		Value: C.GoString(result.value),
	}, nil
}

func frontmostApp() Frontmost {
	name := C.sp_frontmost_app_name()
	defer func() {
		if name != nil {
			C.free(unsafe.Pointer(name))
		}
	}()
	return Frontmost{
		Name: C.GoString(name),
		PID:  int(C.sp_frontmost_pid()),
	}
}

func activateApp(app Frontmost) error {
	if app.PID > 0 && C.sp_activate_pid(C.int(app.PID)) != 0 {
		return nil
	}
	if strings.TrimSpace(app.Name) == "" {
		return ErrUnsupported
	}
	return exec.Command("open", "-a", app.Name).Run()
}

func simulatePaste() error {
	C.sp_send_paste()
	return nil
}

func IsMainThread() bool { return C.sp_is_main_thread() != 0 }

func focusSupported() bool { return true }
func pasteSupported() bool { return true }

func permissions() []Permission {
	return []Permission{
		{
			Key:         "accessibility",
			LabelKey:    "setup.permission.accessibility",
			Allowed:     C.sp_ax_trusted() != 0,
			Required:    true,
			SettingsURL: "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility",
		},
		{
			Key:         "input",
			LabelKey:    "setup.permission.input",
			Allowed:     C.sp_input_allowed() != 0,
			Required:    true,
			SettingsURL: "x-apple.systempreferences:com.apple.preference.security?Privacy_ListenEvent",
		},
	}
}

func openPermissionSettings(permission Permission) error {
	if permission.SettingsURL == "" {
		return ErrUnsupported
	}
	return exec.Command("open", permission.SettingsURL).Run()
}

func requestPermissions() {
	C.sp_request_permissions()
}

func systemLanguage() string {
	locale := C.sp_locale_id()
	defer func() {
		if locale != nil {
			C.free(unsafe.Pointer(locale))
		}
	}()
	return strings.ToLower(C.GoString(locale))
}

func isWayland() bool { return false }
