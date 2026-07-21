//go:build darwin

package notifier

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework AppKit
#include <stdlib.h>
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>

// axSessionFindPID returns the pid of the first running app with bundleID, or -1.
static int axSessionFindPID(const char *bundleID) {
	@autoreleasepool {
		NSString *bid = [NSString stringWithUTF8String:bundleID];
		NSArray *apps = [NSRunningApplication runningApplicationsWithBundleIdentifier:bid];
		if (!apps || apps.count == 0) return -1;
		return (int)((NSRunningApplication *)apps[0]).processIdentifier;
	}
}

// axSessionActivate brings the app with pid to the foreground.
static void axSessionActivate(int pid) {
	@autoreleasepool {
		NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:(pid_t)pid];
		if (!app) return;
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
		[app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
#pragma clang diagnostic pop
	}
}

// promptForAXTrust triggers the system Accessibility permission prompt so the
// responsible app (ClaudeNotifier when run from a notification click) appears
// in System Settings > Privacy & Security > Accessibility.
static int promptForAXTrust(void) {
	@autoreleasepool {
		NSDictionary *opts = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
		return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts) ? 1 : 0;
	}
}

// enableElectronAX asks the Electron app to build its accessibility tree.
// Electron exposes web content to AX clients only after this (or after any
// assistive client connects).
static void enableElectronAX(int pid) {
	AXUIElementRef appEl = AXUIElementCreateApplication((pid_t)pid);
	if (!appEl) return;
	AXUIElementSetAttributeValue(appEl, CFSTR("AXManualAccessibility"), kCFBooleanTrue);
	CFRelease(appEl);
}

// titleMatchesSession reports whether an element title identifies the target
// conversation. Sidebar buttons carry the conversation title, optionally
// prefixed with a status ("Running <title>"), so match exact or suffix with a
// word boundary. Context-menu buttons ("More options for <title>") share the
// suffix and are excluded by role (they are AXPopUpButton, not AXButton).
static BOOL titleMatchesSession(NSString *title, NSString *target) {
	// Never treat a "Mark as unread <title>" action button as the conversation
	// row — pressing it toggles unread instead of opening (Home-area rows).
	if ([title hasPrefix:@"Mark as unread"]) return NO;
	if ([title isEqualToString:target]) return YES;
	if ([title hasSuffix:[@" " stringByAppendingString:target]]) return YES;
	return NO;
}

// copyButtonName returns the accessible name of a button element: AXTitle when
// present, otherwise AXDescription (icon-only buttons like the Home/Code area
// switchers expose their name there). Caller releases.
static NSString *copyButtonName(AXUIElementRef el) {
	CFTypeRef ref = NULL;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXTitle"), &ref) == kAXErrorSuccess && ref) {
		if (CFGetTypeID(ref) == CFStringGetTypeID() && CFStringGetLength((CFStringRef)ref) > 0) {
			return [(__bridge NSString *)ref autorelease];
		}
		CFRelease(ref);
	}
	ref = NULL;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXDescription"), &ref) == kAXErrorSuccess && ref) {
		if (CFGetTypeID(ref) == CFStringGetTypeID()) {
			return [(__bridge NSString *)ref autorelease];
		}
		CFRelease(ref);
	}
	return nil;
}

// findAndPressSessionButton walks el looking for an AXButton whose title
// matches target and presses it. Returns 1 when pressed, 0 when not found.
static int findAndPressSessionButton(AXUIElementRef el, NSString *target, int depth, int *budget) {
	if (depth > 40 || *budget <= 0) return 0;
	(*budget)--;

	CFTypeRef roleRef = NULL;
	NSString *role = nil;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXRole"), &roleRef) == kAXErrorSuccess && roleRef) {
		if (CFGetTypeID(roleRef) == CFStringGetTypeID()) role = (__bridge NSString *)roleRef;
	}

	if (role && [role isEqualToString:@"AXButton"]) {
		NSString *name = copyButtonName(el);
		if (name && titleMatchesSession(name, target)) {
			if (roleRef) CFRelease(roleRef);
			AXUIElementPerformAction(el, CFSTR("AXScrollToVisible"));
			AXError pressErr = AXUIElementPerformAction(el, CFSTR("AXPress"));
			return pressErr == kAXErrorSuccess ? 1 : 0;
		}
	}
	if (roleRef) CFRelease(roleRef);

	CFTypeRef childrenRef = NULL;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXChildren"), &childrenRef) != kAXErrorSuccess || !childrenRef) {
		return 0;
	}
	int pressed = 0;
	CFArrayRef children = (CFArrayRef)childrenRef;
	CFIndex count = CFArrayGetCount(children);
	for (CFIndex i = 0; i < count && !pressed; i++) {
		AXUIElementRef child = (AXUIElementRef)CFArrayGetValueAtIndex(children, i);
		pressed = findAndPressSessionButton(child, target, depth + 1, budget);
	}
	CFRelease(childrenRef);
	return pressed;
}

// findAndPressExactButton walks el for an AXButton titled exactly target and
// presses it. Used for fixed navigation buttons like "Code" / "Home".
static int findAndPressExactButton(AXUIElementRef el, NSString *target, int depth, int *budget) {
	if (depth > 40 || *budget <= 0) return 0;
	(*budget)--;

	CFTypeRef roleRef = NULL;
	BOOL isButton = NO;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXRole"), &roleRef) == kAXErrorSuccess && roleRef) {
		isButton = CFGetTypeID(roleRef) == CFStringGetTypeID() &&
			[(__bridge NSString *)roleRef isEqualToString:@"AXButton"];
		CFRelease(roleRef);
	}
	if (isButton) {
		NSString *name = copyButtonName(el);
		if (name && [name isEqualToString:target]) {
			return AXUIElementPerformAction(el, CFSTR("AXPress")) == kAXErrorSuccess ? 1 : 0;
		}
	}

	CFTypeRef childrenRef = NULL;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXChildren"), &childrenRef) != kAXErrorSuccess || !childrenRef) {
		return 0;
	}
	int pressed = 0;
	CFArrayRef children = (CFArrayRef)childrenRef;
	CFIndex count = CFArrayGetCount(children);
	for (CFIndex i = 0; i < count && !pressed; i++) {
		pressed = findAndPressExactButton((AXUIElementRef)CFArrayGetValueAtIndex(children, i), target, depth + 1, budget);
	}
	CFRelease(childrenRef);
	return pressed;
}

// pressExactButtonInApp presses the first AXButton titled exactly targetTitle.
// Returns 1 pressed, 0 not found, -1 not trusted.
static int pressExactButtonInApp(int pid, const char *targetTitle) {
	@autoreleasepool {
		if (!AXIsProcessTrusted()) return -1;
		AXUIElementRef appEl = AXUIElementCreateApplication((pid_t)pid);
		if (!appEl) return 0;
		CFTypeRef windowsRef = NULL;
		if (AXUIElementCopyAttributeValue(appEl, CFSTR("AXWindows"), &windowsRef) != kAXErrorSuccess || !windowsRef) {
			CFRelease(appEl);
			return 0;
		}
		NSString *target = [NSString stringWithUTF8String:targetTitle];
		int pressed = 0;
		int budget = 200000;
		CFArrayRef windows = (CFArrayRef)windowsRef;
		CFIndex count = CFArrayGetCount(windows);
		for (CFIndex i = 0; i < count && !pressed; i++) {
			pressed = findAndPressExactButton((AXUIElementRef)CFArrayGetValueAtIndex(windows, i), target, 0, &budget);
		}
		CFRelease(windowsRef);
		CFRelease(appEl);
		return pressed;
	}
}

// findAndPressPrefixButton walks el for an AXButton whose accessible name
// STARTS WITH target and presses it. Permission-card buttons carry hotkey
// suffixes ("Always allow 2", "Allow once 3 ⌘ ⏎"), so exact match fails.
static int findAndPressPrefixButton(AXUIElementRef el, NSString *target, int depth, int *budget) {
	if (depth > 40 || *budget <= 0) return 0;
	(*budget)--;

	CFTypeRef roleRef = NULL;
	BOOL isButton = NO;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXRole"), &roleRef) == kAXErrorSuccess && roleRef) {
		isButton = CFGetTypeID(roleRef) == CFStringGetTypeID() &&
			[(__bridge NSString *)roleRef isEqualToString:@"AXButton"];
		CFRelease(roleRef);
	}
	if (isButton) {
		NSString *name = copyButtonName(el);
		if (name && [name hasPrefix:target]) {
			AXUIElementPerformAction(el, CFSTR("AXScrollToVisible"));
			return AXUIElementPerformAction(el, CFSTR("AXPress")) == kAXErrorSuccess ? 1 : 0;
		}
	}

	CFTypeRef childrenRef = NULL;
	if (AXUIElementCopyAttributeValue(el, CFSTR("AXChildren"), &childrenRef) != kAXErrorSuccess || !childrenRef) {
		return 0;
	}
	int pressed = 0;
	CFArrayRef children = (CFArrayRef)childrenRef;
	CFIndex count = CFArrayGetCount(children);
	for (CFIndex i = 0; i < count && !pressed; i++) {
		pressed = findAndPressPrefixButton((AXUIElementRef)CFArrayGetValueAtIndex(children, i), target, depth + 1, budget);
	}
	CFRelease(childrenRef);
	return pressed;
}

// pressPrefixButtonInApp presses the first AXButton whose name starts with
// targetPrefix. Returns 1 pressed, 0 not found, -1 not trusted.
static int pressPrefixButtonInApp(int pid, const char *targetPrefix) {
	@autoreleasepool {
		if (!AXIsProcessTrusted()) return -1;
		AXUIElementRef appEl = AXUIElementCreateApplication((pid_t)pid);
		if (!appEl) return 0;
		CFTypeRef windowsRef = NULL;
		if (AXUIElementCopyAttributeValue(appEl, CFSTR("AXWindows"), &windowsRef) != kAXErrorSuccess || !windowsRef) {
			CFRelease(appEl);
			return 0;
		}
		NSString *target = [NSString stringWithUTF8String:targetPrefix];
		int pressed = 0;
		int budget = 200000;
		CFArrayRef windows = (CFArrayRef)windowsRef;
		CFIndex count = CFArrayGetCount(windows);
		for (CFIndex i = 0; i < count && !pressed; i++) {
			pressed = findAndPressPrefixButton((AXUIElementRef)CFArrayGetValueAtIndex(windows, i), target, 0, &budget);
		}
		CFRelease(windowsRef);
		CFRelease(appEl);
		return pressed;
	}
}

// pressSessionButtonInApp searches all windows of pid for the conversation
// button and presses it. Returns 1 pressed, 0 not found, -1 not trusted.
static int pressSessionButtonInApp(int pid, const char *targetTitle) {
	@autoreleasepool {
		if (!AXIsProcessTrusted()) return -1;

		AXUIElementRef appEl = AXUIElementCreateApplication((pid_t)pid);
		if (!appEl) return 0;

		CFTypeRef windowsRef = NULL;
		if (AXUIElementCopyAttributeValue(appEl, CFSTR("AXWindows"), &windowsRef) != kAXErrorSuccess || !windowsRef) {
			CFRelease(appEl);
			return 0;
		}

		NSString *target = [NSString stringWithUTF8String:targetTitle];
		int pressed = 0;
		int budget = 200000;
		CFArrayRef windows = (CFArrayRef)windowsRef;
		CFIndex count = CFArrayGetCount(windows);
		for (CFIndex i = 0; i < count && !pressed; i++) {
			AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);
			pressed = findAndPressSessionButton(win, target, 0, &budget);
		}

		CFRelease(windowsRef);
		CFRelease(appEl);
		return pressed;
	}
}
*/
import "C"

import (
	"fmt"
	"os/exec"
	"time"
	"unsafe"

	"github.com/777genius/claude-notifications/internal/logging"
	"github.com/777genius/claude-notifications/internal/platform"
)

// RespondDesktopApproval answers a pending permission card in the Claude
// desktop app: it navigates to the conversation (sidebar press) and then
// presses the card's "Always allow" or "Allow once" button. The card buttons
// carry hotkey suffixes, so they are matched by prefix. Used by the approval
// notification's action buttons.
func RespondDesktopApproval(cliSessionID, scope string) error {
	var buttonPrefix string
	switch scope {
	case "always":
		buttonPrefix = "Always allow"
	case "once":
		buttonPrefix = "Allow once"
	default:
		return fmt.Errorf("unknown approval scope %q (want always|once)", scope)
	}

	// Bring the right conversation to front; ignore failure (the press loop
	// below still tries — the card may already be visible).
	if err := FocusDesktopSessionByCLIID(cliSessionID); err != nil {
		logging.Debug("respond-approval: focus failed (continuing): %v", err)
	}

	cBundleID := C.CString(platform.DesktopAppBundleID)
	defer C.free(unsafe.Pointer(cBundleID))
	pid := int(C.axSessionFindPID(cBundleID))
	if pid < 0 {
		return fmt.Errorf("Claude desktop app is not running")
	}

	cPrefix := C.CString(buttonPrefix)
	defer C.free(unsafe.Pointer(cPrefix))

	deadline := time.Now().Add(6 * time.Second)
	for {
		switch C.pressPrefixButtonInApp(C.int(pid), cPrefix) {
		case 1:
			logging.Debug("respond-approval: pressed %q for session %s", buttonPrefix, cliSessionID)
			return nil
		case -1:
			return fmt.Errorf("accessibility permission not granted")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no %q button found (card may already be answered)", buttonPrefix)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// FocusDesktopSessionByCLIID brings the Claude desktop app to the front and
// selects the conversation belonging to cliSessionID by pressing its sidebar
// item through the Accessibility API. The app is activated regardless of the
// outcome, so on any failure the behavior degrades to plain app focus.
func FocusDesktopSessionByCLIID(cliSessionID string) error {
	_, title := resolveDesktopSession(cliSessionID)
	return focusDesktopByTitle(title, "cli session "+cliSessionID)
}

// FocusDesktopSessionByWrapper focuses a conversation by the desktop app's own
// wrapper id (used for Cowork/Home task notifications, whose click has no CLI
// id to resolve from). title is passed through from the notification because
// Home tasks have no session record to re-resolve it from; falls back to a
// record lookup only if empty. Presses the sidebar item matching the title.
func FocusDesktopSessionByWrapper(wrapperID, title string) error {
	if title == "" {
		_, title = ResolveDesktopSessionByWrapper(wrapperID)
	}

	cBundleID := C.CString(platform.DesktopAppBundleID)
	defer C.free(unsafe.Pointer(cBundleID))
	pid := int(C.axSessionFindPID(cBundleID))
	if pid < 0 {
		if err := exec.Command("open", "-b", platform.DesktopAppBundleID).Run(); err != nil {
			return fmt.Errorf("Claude desktop app is not running and could not be launched: %w", err)
		}
		deadline := time.Now().Add(15 * time.Second)
		for pid < 0 {
			if time.Now().After(deadline) {
				return fmt.Errorf("Claude desktop app did not start in time")
			}
			time.Sleep(500 * time.Millisecond)
			pid = int(C.axSessionFindPID(cBundleID))
		}
		time.Sleep(2 * time.Second)
	}
	C.axSessionActivate(C.int(pid))

	if C.promptForAXTrust() == 0 {
		logging.Warn("focus-cowork: AX trust NOT granted for this process")
		return fmt.Errorf("accessibility permission not granted")
	}
	C.enableElectronAX(C.int(pid))
	time.Sleep(600 * time.Millisecond) // let the AX tree build after activation

	// Land on the Home area, where cloud/Home tasks live — reliable and
	// non-destructive (the task list is right there). We do NOT press the task
	// row: an unread task's only titled control is "Mark as unread <title>",
	// and pressing it toggles unread instead of opening. Best-effort exact-open
	// is attempted only for rows that expose a real, non-action title.
	cHome := C.CString("Home")
	homeResult := int(C.pressExactButtonInApp(C.int(pid), cHome))
	C.free(unsafe.Pointer(cHome))
	logging.Debug("focus-cowork: pid=%d Home-press result=%d (1=pressed 0=not found -1=untrusted)", pid, homeResult)

	if title == "" {
		return nil // on Home area; nothing more we can safely do
	}
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if C.pressSessionButtonInApp(C.int(pid), cTitle) == 1 {
			logging.Debug("Opened Home task %q", title)
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	logging.Debug("Home task %q not directly openable; left on Home area", title)
	return nil
}

// focusDesktopByTitle activates the Claude desktop app and presses the sidebar
// item whose name matches title, launching the app first if needed.
func focusDesktopByTitle(title, label string) error {
	cBundleID := C.CString(platform.DesktopAppBundleID)
	defer C.free(unsafe.Pointer(cBundleID))
	pid := int(C.axSessionFindPID(cBundleID))
	if pid < 0 {
		// App not running: launch it, then wait for it to come up so the
		// sidebar press below can still find the conversation.
		if err := exec.Command("open", "-b", platform.DesktopAppBundleID).Run(); err != nil {
			return fmt.Errorf("Claude desktop app is not running and could not be launched: %w", err)
		}
		launchDeadline := time.Now().Add(15 * time.Second)
		for pid < 0 {
			if time.Now().After(launchDeadline) {
				return fmt.Errorf("Claude desktop app did not start in time")
			}
			time.Sleep(500 * time.Millisecond)
			pid = int(C.axSessionFindPID(cBundleID))
		}
		// Give the freshly launched app a moment to render its first window.
		time.Sleep(2 * time.Second)
	}

	// Always bring the app forward first — matches previous click behavior
	// and gives Electron time to build its AX tree while animating.
	C.axSessionActivate(C.int(pid))

	if title == "" {
		return fmt.Errorf("no conversation title found for %s", label)
	}

	if C.promptForAXTrust() == 0 {
		return fmt.Errorf("accessibility permission not granted (grant it to Claude Notifier in System Settings > Privacy & Security > Accessibility)")
	}

	C.enableElectronAX(C.int(pid))

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	// The Electron AX tree builds asynchronously after enableElectronAX; the
	// sidebar also re-renders on app activation. Retry briefly. Claude Code
	// conversations are only listed in the sidebar while the "Code" area is
	// active, so after a couple of misses (user is in Home) switch areas and
	// keep looking.
	// A conversation lives in exactly one area's sidebar — Code sessions under
	// "Code", Home/Cowork tasks under "Home" — so if the first look misses,
	// cycle through both areas and retry rather than assuming Code.
	areas := []string{"Code", "Home"}
	deadline := time.Now().Add(10 * time.Second)
	areaIdx := 0
	for {
		switch C.pressSessionButtonInApp(C.int(pid), cTitle) {
		case 1:
			logging.Debug("Pressed sidebar item for conversation %q", title)
			return nil
		case -1:
			return fmt.Errorf("accessibility permission not granted")
		}
		// Miss: switch to the next area (its recents list re-renders) and retry.
		if areaIdx < len(areas) {
			cArea := C.CString(areas[areaIdx])
			if C.pressExactButtonInApp(C.int(pid), cArea) == 1 {
				logging.Debug("Conversation not found; switched app to %s area", areas[areaIdx])
			}
			C.free(unsafe.Pointer(cArea))
			areaIdx++
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("conversation %q not found in app UI (app left focused)", title)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
