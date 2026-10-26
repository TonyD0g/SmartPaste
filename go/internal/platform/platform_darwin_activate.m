#import <AppKit/AppKit.h>

int sp_activate_pid(int pid) {
	@autoreleasepool {
		NSRunningApplication *application = [NSRunningApplication
			runningApplicationWithProcessIdentifier:(pid_t)pid];
		if (application == nil) {
			return 0;
		}
		return [application activateWithOptions:NSApplicationActivateAllWindows] ? 1 : 0;
	}
}
