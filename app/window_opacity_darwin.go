//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static NSWindow *flashdockOpacityWindow(void) {
	for (NSWindow *window in [NSApp windows]) {
		NSString *className = NSStringFromClass([window class]) ?: @"";
		if ([className isEqualToString:@"WailsWindow"]) {
			return window;
		}
	}
	return [NSApp mainWindow] ?: [NSApp keyWindow];
}

void flashdockSetWindowAlpha(double a) {
	NSWindow *w = flashdockOpacityWindow();
	if (w == nil) {
		return;
	}
	if (a >= 0.995) {
		[w setAlphaValue:1.0];
		[w setOpaque:YES];
		return;
	}
	if (a < 0.4) {
		a = 0.4;
	}
	[w setOpaque:NO];
	[w setAlphaValue:a];
}

int flashdockWindowIsVisible(void) {
	NSWindow *w = flashdockOpacityWindow();
	if (w == nil) {
		return 0;
	}
	return [w isVisible] ? 1 : 0;
}
*/
import "C"

func nativeSetWindowAlpha(opacity float64) {
	C.flashdockSetWindowAlpha(C.double(opacity))
}

func nativeWindowIsVisible() bool {
	return C.flashdockWindowIsVisible() != 0
}
