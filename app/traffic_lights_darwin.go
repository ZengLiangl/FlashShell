//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

// 对齐 Electron trafficLightPosition：通过调整 titleBarContainer 高度与边距，
// 让三颗红绿灯在自定义顶栏高度内垂直居中（而非只改 button frame）。

static CGFloat flashdockTrafficX = 12.0;
static CGFloat flashdockTrafficY = 12.0;
static BOOL flashdockTrafficObserverInstalled = NO;

static NSWindow *flashdockMainWindow(void) {
	for (NSWindow *window in [NSApp windows]) {
		NSString *className = NSStringFromClass([window class]) ?: @"";
		if ([className isEqualToString:@"WailsWindow"]) {
			return window;
		}
	}
	return [NSApp mainWindow] ?: [NSApp keyWindow];
}

// close.superview = 按钮行；再上一级才是 NSTitlebarContainerView
static NSView *flashdockTitleBarContainer(NSWindow *window) {
	NSButton *closeButton = [window standardWindowButton:NSWindowCloseButton];
	if (closeButton == nil || [closeButton superview] == nil) {
		return nil;
	}
	return [[closeButton superview] superview];
}

static void flashdockApplyTrafficLightPosition(NSWindow *window) {
	if (window == nil) {
		return;
	}
	if (([window styleMask] & NSWindowStyleMaskFullScreen) != 0) {
		return;
	}

	NSButton *closeButton = [window standardWindowButton:NSWindowCloseButton];
	NSButton *miniaturizeButton = [window standardWindowButton:NSWindowMiniaturizeButton];
	NSButton *zoomButton = [window standardWindowButton:NSWindowZoomButton];
	if (closeButton == nil || miniaturizeButton == nil || zoomButton == nil) {
		return;
	}

	NSView *titleBarContainer = flashdockTitleBarContainer(window);
	if (titleBarContainer == nil) {
		return;
	}

	CGFloat buttonWidth = NSWidth([closeButton frame]);
	CGFloat buttonHeight = NSHeight([closeButton frame]);
	CGFloat padding = NSMinX([miniaturizeButton frame]) - NSMaxX([closeButton frame]);
	if (padding <= 0) {
		padding = 6.0;
	}

	// 容器高度 = 按钮 + 上下各 margin.y，使红绿灯在顶栏内垂直居中
	CGFloat containerHeight = buttonHeight + 2.0 * flashdockTrafficY;
	NSRect windowFrame = [window frame];
	NSRect containerFrame = [titleBarContainer frame];
	containerFrame.size.height = containerHeight;
	containerFrame.origin.y = NSHeight(windowFrame) - containerHeight;
	[titleBarContainer setFrame:containerFrame];

	CGFloat startX = flashdockTrafficX;
	CGFloat buttonY = flashdockTrafficY; // AppKit：相对父视图底边

	[closeButton setFrameOrigin:NSMakePoint(startX, buttonY)];
	[miniaturizeButton setFrameOrigin:NSMakePoint(startX + buttonWidth + padding, buttonY)];
	[zoomButton setFrameOrigin:NSMakePoint(startX + 2.0 * (buttonWidth + padding), buttonY)];
}

@interface FlashDockTrafficLightObserver : NSObject
@end

@implementation FlashDockTrafficLightObserver
- (void)onWindowEvent:(NSNotification *)notification {
	NSWindow *window = nil;
	if ([[notification object] isKindOfClass:[NSWindow class]]) {
		window = (NSWindow *)[notification object];
	} else {
		window = flashdockMainWindow();
	}
	flashdockApplyTrafficLightPosition(window);
}
@end

static FlashDockTrafficLightObserver *flashdockTrafficObserver = nil;

static void flashdockInstallTrafficLightObserver(void) {
	if (flashdockTrafficObserverInstalled) {
		return;
	}
	flashdockTrafficObserverInstalled = YES;
	flashdockTrafficObserver = [[FlashDockTrafficLightObserver alloc] init];
	NSNotificationCenter *center = [NSNotificationCenter defaultCenter];
	NSArray<NSString *> *names = @[
		NSWindowDidResizeNotification,
		NSWindowDidExitFullScreenNotification,
		NSWindowDidBecomeMainNotification,
		NSWindowDidDeminiaturizeNotification,
	];
	for (NSString *name in names) {
		[center addObserver:flashdockTrafficObserver selector:@selector(onWindowEvent:) name:name object:nil];
	}
}

void flashdockSetTrafficLightPosition(double x, double y) {
	flashdockTrafficX = (CGFloat)x;
	flashdockTrafficY = (CGFloat)y;
	dispatch_async(dispatch_get_main_queue(), ^{
		flashdockInstallTrafficLightObserver();
		flashdockApplyTrafficLightPosition(flashdockMainWindow());
		// 布局稳定后再贴两次，避免被系统打回默认位置
		dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(50 * NSEC_PER_MSEC)), dispatch_get_main_queue(), ^{
			flashdockApplyTrafficLightPosition(flashdockMainWindow());
		});
		dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(250 * NSEC_PER_MSEC)), dispatch_get_main_queue(), ^{
			flashdockApplyTrafficLightPosition(flashdockMainWindow());
		});
	});
}
*/
import "C"

func setTrafficLightPosition(x, y float64) {
	C.flashdockSetTrafficLightPosition(C.double(x), C.double(y))
}
