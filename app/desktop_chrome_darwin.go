//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -fblocks
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void flashdockGoShow(void);
extern void flashdockGoQuit(void);
extern int flashdockGoMinimizeToTray(void);

static NSWindow *flashdockDesktopWindow(void) {
	for (NSWindow *window in [NSApp windows]) {
		NSString *className = NSStringFromClass([window class]) ?: @"";
		if ([className isEqualToString:@"WailsWindow"]) {
			return window;
		}
	}
	return [NSApp mainWindow] ?: [NSApp keyWindow];
}

static NSStatusItem *flashdockStatusItem = nil;
static BOOL flashdockMiniaturizeObserverInstalled = NO;

@interface FlashDockStatusTarget : NSObject
@end

@implementation FlashDockStatusTarget
- (void)showMain:(id)sender {
	flashdockGoShow();
}
- (void)quitApp:(id)sender {
	flashdockGoQuit();
}
- (void)onDidMiniaturize:(NSNotification *)notification {
	if (!flashdockGoMinimizeToTray()) {
		return;
	}
	NSWindow *w = flashdockDesktopWindow();
	if (w == nil) {
		return;
	}
	[w deminiaturize:nil];
	[w orderOut:nil];
}
@end

static FlashDockStatusTarget *flashdockStatusTarget = nil;

static FlashDockStatusTarget *flashdockEnsureTarget(void) {
	if (flashdockStatusTarget == nil) {
		flashdockStatusTarget = [[FlashDockStatusTarget alloc] init];
	}
	return flashdockStatusTarget;
}

static void flashdockInstallMiniaturizeObserver(void) {
	if (flashdockMiniaturizeObserverInstalled) {
		return;
	}
	flashdockMiniaturizeObserverInstalled = YES;
	NSNotificationCenter *center = [NSNotificationCenter defaultCenter];
	[center addObserver:flashdockEnsureTarget()
	           selector:@selector(onDidMiniaturize:)
	               name:NSWindowDidMiniaturizeNotification
	             object:nil];
}

void flashdockHideDesktopWindow(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *w = flashdockDesktopWindow();
		if (w != nil) {
			[w orderOut:nil];
		}
	});
}

void flashdockShowDesktopWindow(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *w = flashdockDesktopWindow();
		[NSApp activateIgnoringOtherApps:YES];
		if (w != nil) {
			[w deminiaturize:nil];
			[w makeKeyAndOrderFront:nil];
		}
	});
}

void flashdockSetTrayEnabled(int on) {
	dispatch_async(dispatch_get_main_queue(), ^{
		flashdockInstallMiniaturizeObserver();
		if (!on) {
			if (flashdockStatusItem != nil) {
				[[NSStatusBar systemStatusBar] removeStatusItem:flashdockStatusItem];
				flashdockStatusItem = nil;
			}
			return;
		}
		if (flashdockStatusItem != nil) {
			return;
		}
		flashdockStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
		NSStatusBarButton *btn = [flashdockStatusItem button];
		[btn setTitle:@"FS"];
		[btn setToolTip:@"FlashShell"];
		[btn setTarget:flashdockEnsureTarget()];
		[btn setAction:@selector(showMain:)];

		NSMenu *menu = [[NSMenu alloc] init];
		NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"显示 FlashShell"
		                                                  action:@selector(showMain:)
		                                           keyEquivalent:@""];
		[showItem setTarget:flashdockEnsureTarget()];
		NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"退出"
		                                                  action:@selector(quitApp:)
		                                           keyEquivalent:@""];
		[quitItem setTarget:flashdockEnsureTarget()];
		[menu addItem:showItem];
		[menu addItem:[NSMenuItem separatorItem]];
		[menu addItem:quitItem];
		[flashdockStatusItem setMenu:menu];
	});
}

void flashdockSetTrayIconPNG(const unsigned char *data, int len) {
	if (data == NULL || len <= 0) {
		return;
	}
	NSData *payload = [NSData dataWithBytes:data length:(NSUInteger)len];
	NSImage *image = [[NSImage alloc] initWithData:payload];
	if (image == nil) {
		return;
	}
	[image setSize:NSMakeSize(18, 18)];
	[image setTemplate:YES];
	dispatch_async(dispatch_get_main_queue(), ^{
		if (flashdockStatusItem != nil) {
			[[flashdockStatusItem button] setImage:image];
			[[flashdockStatusItem button] setTitle:@""];
		}
	});
}

void flashdockEnsureDesktopHook(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		flashdockInstallMiniaturizeObserver();
	});
}
*/
import "C"

import (
	"unsafe"
)

func nativeEnsureWindowHook() {
	C.flashdockEnsureDesktopHook()
}

func nativeApplyWindowOpacity(opacity float64) {
	nativeSetWindowAlpha(opacity)
}

func nativeHideMainWindow(a *App) {
	C.flashdockHideDesktopWindow()
	wailsHide(a)
}

func nativeShowMainWindow(a *App) {
	wailsShow(a)
	C.flashdockShowDesktopWindow()
}

func nativeMainWindowVisible() bool {
	return nativeWindowIsVisible()
}

func nativeSetTrayEnabled(on bool) {
	flag := C.int(0)
	if on {
		flag = 1
	}
	C.flashdockSetTrayEnabled(flag)
	if on && len(embeddedDefaultAppIcon) > 0 {
		C.flashdockSetTrayIconPNG(
			(*C.uchar)(unsafe.Pointer(&embeddedDefaultAppIcon[0])),
			C.int(len(embeddedDefaultAppIcon)),
		)
	}
}

//export flashdockGoShow
func flashdockGoShow() {
	if desktopApp != nil {
		desktopApp.ShowMainWindow()
	}
}

//export flashdockGoQuit
func flashdockGoQuit() {
	desktopQuitFromTray()
}

//export flashdockGoMinimizeToTray
func flashdockGoMinimizeToTray() C.int {
	if desktopApp != nil && desktopApp.minimizeToTrayEnabled() {
		return 1
	}
	return 0
}
