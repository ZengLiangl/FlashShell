//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void flashdockSetDockIconPNG(const void *data, int length) {
	@autoreleasepool {
		if (data == NULL || length <= 0) {
			[NSApp setApplicationIconImage:nil];
			return;
		}
		NSData *nsData = [NSData dataWithBytes:data length:(NSUInteger)length];
		NSImage *image = [[NSImage alloc] initWithData:nsData];
		if (image != nil) {
			[NSApp setApplicationIconImage:image];
		}
	}
}
*/
import "C"
import "unsafe"

func setApplicationDockIconPNG(pngBytes []byte) {
	if len(pngBytes) == 0 {
		C.flashdockSetDockIconPNG(nil, 0)
		return
	}
	C.flashdockSetDockIconPNG(unsafe.Pointer(&pngBytes[0]), C.int(len(pngBytes)))
}
