//go:build darwin

package inputmethod

/*
#cgo LDFLAGS: -framework Carbon -framework CoreFoundation
#include <Carbon/Carbon.h>
#include <CoreFoundation/CoreFoundation.h>
#include <dispatch/dispatch.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>

static char *flashdockCopySourceID(TISInputSourceRef src) {
	if (src == NULL) {
		return NULL;
	}
	CFStringRef id = (CFStringRef)TISGetInputSourceProperty(src, kTISPropertyInputSourceID);
	if (id == NULL) {
		return NULL;
	}
	CFIndex len = CFStringGetLength(id);
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
	char *buf = (char *)malloc((size_t)maxSize);
	if (buf == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(id, buf, maxSize, kCFStringEncodingUTF8)) {
		free(buf);
		return NULL;
	}
	return buf;
}

static char *flashdockCurrentSourceID(void) {
	TISInputSourceRef src = TISCopyCurrentKeyboardInputSource();
	if (src == NULL) {
		return NULL;
	}
	char *id = flashdockCopySourceID(src);
	CFRelease(src);
	return id;
}

static int flashdockSelectSourceID(const char *sourceID) {
	if (sourceID == NULL || sourceID[0] == '\0') {
		return -1;
	}
	CFStringRef cfID = CFStringCreateWithCString(kCFAllocatorDefault, sourceID, kCFStringEncodingUTF8);
	if (cfID == NULL) {
		return -1;
	}
	const void *keys[] = { kTISPropertyInputSourceID };
	const void *vals[] = { cfID };
	CFDictionaryRef filter = CFDictionaryCreate(
		kCFAllocatorDefault,
		keys,
		vals,
		1,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	CFRelease(cfID);
	if (filter == NULL) {
		return -1;
	}
	CFArrayRef list = TISCreateInputSourceList(filter, false);
	CFRelease(filter);
	if (list == NULL || CFArrayGetCount(list) == 0) {
		if (list != NULL) {
			CFRelease(list);
		}
		return -1;
	}
	TISInputSourceRef src = (TISInputSourceRef)CFArrayGetValueAtIndex(list, 0);
	OSStatus st = TISSelectInputSource(src);
	CFRelease(list);
	return (int)st;
}

static int flashdockSelectASCII(void) {
	TISInputSourceRef ascii = TISCopyCurrentASCIICapableKeyboardInputSource();
	if (ascii == NULL) {
		return -1;
	}
	OSStatus st = TISSelectInputSource(ascii);
	CFRelease(ascii);
	return (int)st;
}

// TIS API 必须在主线程调用；Wails 绑定跑在后台 goroutine，直接调会 SIGSEGV。
static void flashdockOnMain(void (*fn)(void *), void *ctx) {
	if (pthread_main_np()) {
		fn(ctx);
		return;
	}
	dispatch_sync_f(dispatch_get_main_queue(), ctx, fn);
}

typedef struct {
	char *prevID;
	int status;
	const char *restoreID;
} flashdockIMCtx;

static void flashdockEnterMain(void *context) {
	flashdockIMCtx *ctx = (flashdockIMCtx *)context;
	ctx->prevID = flashdockCurrentSourceID();
	if (ctx->prevID == NULL) {
		ctx->status = -1;
		return;
	}
	ctx->status = flashdockSelectASCII();
	if (ctx->status != 0) {
		free(ctx->prevID);
		ctx->prevID = NULL;
	}
}

static void flashdockLeaveMain(void *context) {
	flashdockIMCtx *ctx = (flashdockIMCtx *)context;
	ctx->status = flashdockSelectSourceID(ctx->restoreID);
}

static char *flashdockEnterASCII(int *outStatus) {
	flashdockIMCtx ctx;
	memset(&ctx, 0, sizeof(ctx));
	ctx.status = -1;
	flashdockOnMain(flashdockEnterMain, &ctx);
	if (outStatus != NULL) {
		*outStatus = ctx.status;
	}
	return ctx.prevID;
}

static int flashdockLeaveASCII(const char *sourceID) {
	flashdockIMCtx ctx;
	memset(&ctx, 0, sizeof(ctx));
	ctx.status = -1;
	ctx.restoreID = sourceID;
	flashdockOnMain(flashdockLeaveMain, &ctx);
	return ctx.status;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

var savedSourceID string

func platformEnter() error {
	var status C.int
	cID := C.flashdockEnterASCII(&status)
	if cID == nil || status != 0 {
		if cID != nil {
			C.free(unsafe.Pointer(cID))
		}
		return fmt.Errorf("切换英文输入失败")
	}
	savedSourceID = C.GoString(cID)
	C.free(unsafe.Pointer(cID))
	return nil
}

func platformLeave() error {
	if savedSourceID == "" {
		return nil
	}
	cID := C.CString(savedSourceID)
	defer C.free(unsafe.Pointer(cID))
	st := C.flashdockLeaveASCII(cID)
	savedSourceID = ""
	if st != 0 {
		return fmt.Errorf("恢复输入法失败")
	}
	return nil
}
