//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

void flashdockSetDockIconPNG(const void *data, int length) {
	NSData *nsData = nil;
	if (data != NULL && length > 0) {
		nsData = [[NSData alloc] initWithBytes:data length:(NSUInteger)length];
	}
	// AppKit 必须在主线程；Wails/Go 绑定线程直接调 setApplicationIconImage 会静默失败
	dispatch_async(dispatch_get_main_queue(), ^{
		@autoreleasepool {
			if (nsData == nil) {
				[NSApp setApplicationIconImage:nil];
				[[NSApp dockTile] display];
				return;
			}
			NSImage *image = [[NSImage alloc] initWithData:nsData];
			if (image != nil) {
				[NSApp setApplicationIconImage:image];
				[[NSApp dockTile] display];
				[image release];
			}
			[nsData release];
		}
	});
}

// 改写 Finder 里 .app 的自定义图标（等同「显示简介」粘贴图标），重启后仍显示。
// data 为空时清除自定义图标，恢复包内原 icns。
void flashdockSetFinderAppIconPNG(const char *appPath, const void *data, int length) {
	if (appPath == NULL || appPath[0] == '\0') {
		return;
	}
	NSString *path = [[NSString alloc] initWithUTF8String:appPath];
	if (path == nil) {
		return;
	}
	NSData *nsData = nil;
	if (data != NULL && length > 0) {
		nsData = [[NSData alloc] initWithBytes:data length:(NSUInteger)length];
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		@autoreleasepool {
			NSImage *image = nil;
			if (nsData != nil) {
				image = [[NSImage alloc] initWithData:nsData];
			}
			NSWorkspace *ws = [NSWorkspace sharedWorkspace];
			[ws setIcon:image forFile:path options:0];
			[ws noteFileSystemChanged:path];
			[image release];
			[nsData release];
			[path release];
		}
	});
}
*/
import "C"
import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func setApplicationDockIconPNG(pngBytes []byte) {
	if len(pngBytes) == 0 {
		C.flashdockSetDockIconPNG(nil, 0)
		return
	}
	C.flashdockSetDockIconPNG(unsafe.Pointer(&pngBytes[0]), C.int(len(pngBytes)))
}

func persistFinderAppIcon(pngBytes []byte, restoreDefault bool) {
	for _, p := range candidateFlashDockAppBundles() {
		cpath := C.CString(p)
		if restoreDefault || len(pngBytes) == 0 {
			C.flashdockSetFinderAppIconPNG(cpath, nil, 0)
		} else {
			C.flashdockSetFinderAppIconPNG(cpath, unsafe.Pointer(&pngBytes[0]), C.int(len(pngBytes)))
		}
		C.free(unsafe.Pointer(cpath))
	}
}

func flashDockAppBundleFromExecutable(exe string) string {
	exe = filepath.Clean(exe)
	macOS := filepath.Dir(exe)
	contents := filepath.Dir(macOS)
	bundle := filepath.Dir(contents)
	if filepath.Base(macOS) != "MacOS" || filepath.Base(contents) != "Contents" {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(bundle), ".app") {
		return ""
	}
	return bundle
}

func isFlashDockAppBundle(path string) bool {
	if path == "" {
		return false
	}
	exe := filepath.Join(path, "Contents", "MacOS", "FlashDock")
	st, err := os.Stat(exe)
	return err == nil && !st.IsDir()
}

func candidateFlashDockAppBundles() []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 3)
	add := func(p string) {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" || p == "." || p == "/" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		if !isFlashDockAppBundle(p) {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		add(flashDockAppBundleFromExecutable(exe))
	}
	add("/Applications/FlashDock.app")
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, "Applications", "FlashDock.app"))
	}
	return out
}
