//go:build windows

package inputmethod

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	imeCModeNative    = 0x0001
	imeCModeFullShape = 0x0008
	iaceChildren      = 0x0001
	iaceDefault       = 0x0010
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	imm32  = windows.NewLazySystemDLL("imm32.dll")

	procGetForegroundWindow     = user32.NewProc("GetForegroundWindow")
	procGetFocus                = user32.NewProc("GetFocus")
	procImmGetContext           = imm32.NewProc("ImmGetContext")
	procImmReleaseContext       = imm32.NewProc("ImmReleaseContext")
	procImmGetOpenStatus        = imm32.NewProc("ImmGetOpenStatus")
	procImmSetOpenStatus        = imm32.NewProc("ImmSetOpenStatus")
	procImmGetConversionStatus  = imm32.NewProc("ImmGetConversionStatus")
	procImmSetConversionStatus  = imm32.NewProc("ImmSetConversionStatus")
	procImmAssociateContextEx   = imm32.NewProc("ImmAssociateContextEx")
)

type winSavedState struct {
	mode       string // "status" | "associate"
	hwnd       uintptr
	open       bool
	conversion uint32
	sentence   uint32
}

var saved winSavedState

func platformEnter() error {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return fmt.Errorf("无法获取前台窗口")
	}
	focus, _, _ := procGetFocus.Call()
	target := focus
	if target == 0 {
		target = hwnd
	}

	himc, _, _ := procImmGetContext.Call(target)
	if himc != 0 {
		openR, _, _ := procImmGetOpenStatus.Call(himc)
		var conv, sent uint32
		procImmGetConversionStatus.Call(himc, uintptr(unsafe.Pointer(&conv)), uintptr(unsafe.Pointer(&sent)))

		saved = winSavedState{
			mode:       "status",
			hwnd:       target,
			open:       openR != 0,
			conversion: conv,
			sentence:   sent,
		}

		// 关闭 IME 开状态，并去掉中文/全角组词位 → 保持输入法选中但英文态
		procImmSetOpenStatus.Call(himc, 0)
		newConv := conv &^ (imeCModeNative | imeCModeFullShape)
		procImmSetConversionStatus.Call(himc, uintptr(newConv), uintptr(sent))
		procImmReleaseContext.Call(target, himc)
		return nil
	}

	// WebView 等场景拿不到 HIMC：对本窗口子树解除 IME 关联
	ok, _, _ := procImmAssociateContextEx.Call(hwnd, 0, iaceChildren)
	if ok == 0 {
		return fmt.Errorf("关闭中文组词失败")
	}
	saved = winSavedState{mode: "associate", hwnd: hwnd}
	return nil
}

func platformLeave() error {
	defer func() { saved = winSavedState{} }()
	switch saved.mode {
	case "status":
		if saved.hwnd == 0 {
			return nil
		}
		himc, _, _ := procImmGetContext.Call(saved.hwnd)
		if himc == 0 {
			return nil
		}
		procImmSetConversionStatus.Call(himc, uintptr(saved.conversion), uintptr(saved.sentence))
		openVal := uintptr(0)
		if saved.open {
			openVal = 1
		}
		procImmSetOpenStatus.Call(himc, openVal)
		procImmReleaseContext.Call(saved.hwnd, himc)
		return nil
	case "associate":
		if saved.hwnd == 0 {
			return nil
		}
		ok, _, _ := procImmAssociateContextEx.Call(saved.hwnd, 0, iaceDefault)
		if ok == 0 {
			return fmt.Errorf("恢复输入法失败")
		}
		return nil
	default:
		return nil
	}
}
