//go:build windows

package app

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var errNoIconFrames = errors.New("无法生成图标帧")

const (
	wmSetIcon      = 0x0080
	iconSmall      = 0
	iconBig        = 1
	imageIcon      = 1
	lrLoadFromFile = 0x0010
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procSendMessageW    = user32.NewProc("SendMessageW")
	procLoadImageW      = user32.NewProc("LoadImageW")
	procDestroyIcon     = user32.NewProc("DestroyIcon")
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowThread = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procGetWindow       = user32.NewProc("GetWindow")

	gwOwner = uintptr(4)

	winIconMu    sync.Mutex
	winIconSmall windows.Handle
	winIconBig   windows.Handle
	winIconTemp  string
)

func setApplicationDockIconPNG(pngBytes []byte) {
	if len(pngBytes) == 0 {
		return
	}
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		return
	}
	src, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return
	}
	icoBytes, err := encodeWindowsICO(src, []int{16, 32, 48, 256})
	if err != nil || len(icoBytes) == 0 {
		return
	}
	dir, err := appIconsDir()
	if err != nil {
		dir = os.TempDir()
	}
	icoPath := filepath.Join(dir, "runtime-app.ico")
	if err := os.WriteFile(icoPath, icoBytes, 0644); err != nil {
		return
	}

	small := loadIconFromFile(icoPath, 16)
	big := loadIconFromFile(icoPath, 32)
	if small == 0 && big == 0 {
		_ = os.Remove(icoPath)
		return
	}

	winIconMu.Lock()
	defer winIconMu.Unlock()
	if winIconTemp != "" && winIconTemp != icoPath {
		_ = os.Remove(winIconTemp)
	}
	winIconTemp = icoPath

	if small != 0 {
		sendSetIcon(hwnd, iconSmall, small)
		if winIconSmall != 0 && winIconSmall != small {
			destroyIcon(winIconSmall)
		}
		winIconSmall = small
	}
	if big != 0 {
		sendSetIcon(hwnd, iconBig, big)
		if winIconBig != 0 && winIconBig != big {
			destroyIcon(winIconBig)
		}
		winIconBig = big
	}
}

func sendSetIcon(hwnd windows.HWND, iconType uintptr, hicon windows.Handle) {
	procSendMessageW.Call(uintptr(hwnd), wmSetIcon, iconType, uintptr(hicon))
}

func destroyIcon(h windows.Handle) {
	if h != 0 {
		procDestroyIcon.Call(uintptr(h))
	}
}

func loadIconFromFile(path string, size int) windows.Handle {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	r, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(p)),
		imageIcon,
		uintptr(size),
		uintptr(size),
		lrLoadFromFile,
	)
	return windows.Handle(r)
}

func findMainWindowHWND() windows.HWND {
	pid := windows.GetCurrentProcessId()
	var found windows.HWND
	cb := syscall.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		var windowPID uint32
		procGetWindowThread.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowPID)))
		if windowPID != pid {
			return 1
		}
		vis, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if vis == 0 {
			return 1
		}
		owner, _, _ := procGetWindow.Call(uintptr(hwnd), gwOwner)
		if owner != 0 {
			return 1
		}
		found = hwnd
		return 0
	})
	procEnumWindows.Call(cb, 0)
	return found
}

func encodeWindowsICO(src image.Image, sizes []int) ([]byte, error) {
	type entry struct {
		w, h  int
		bmp   []byte
	}
	entries := make([]entry, 0, len(sizes))
	for _, size := range sizes {
		if size <= 0 {
			continue
		}
		bmp, err := rgbaToBMPIconData(resizeImageNearest(src, size, size))
		if err != nil {
			continue
		}
		entries = append(entries, entry{w: size, h: size, bmp: bmp})
	}
	if len(entries) == 0 {
		return nil, errNoIconFrames
	}

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(entries)))

	offset := 6 + 16*len(entries)
	for _, e := range entries {
		wb, hb := byte(e.w), byte(e.h)
		if e.w >= 256 {
			wb = 0
		}
		if e.h >= 256 {
			hb = 0
		}
		buf.WriteByte(wb)
		buf.WriteByte(hb)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(e.bmp)))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(e.bmp)
	}
	for _, e := range entries {
		buf.Write(e.bmp)
	}
	return buf.Bytes(), nil
}

func resizeImageNearest(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y*sh/h
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*sw/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// rgbaToBMPIconData 生成 ICO 内嵌的 32bpp BMP（含 AND mask）
func rgbaToBMPIconData(img *image.RGBA) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)

	// XOR bitmap: 32bpp bottom-up BGRA
	rowSize := w * 4
	xorSize := rowSize * h
	// AND mask: 1bpp, rows padded to 32 bits
	andRow := ((w + 31) / 32) * 4
	andSize := andRow * h

	bihSize := 40
	buf := make([]byte, bihSize+xorSize+andSize)
	binary.LittleEndian.PutUint32(buf[0:], uint32(bihSize))
	binary.LittleEndian.PutUint32(buf[4:], uint32(w))
	binary.LittleEndian.PutUint32(buf[8:], uint32(h*2)) // height includes mask
	binary.LittleEndian.PutUint16(buf[12:], 1)
	binary.LittleEndian.PutUint16(buf[14:], 32)
	binary.LittleEndian.PutUint32(buf[16:], 0)
	binary.LittleEndian.PutUint32(buf[20:], uint32(xorSize))

	xor := buf[bihSize:]
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		dstOff := y * rowSize
		srcOff := rgba.PixOffset(0, srcY)
		for x := 0; x < w; x++ {
			i := srcOff + x*4
			j := dstOff + x*4
			xor[j+0] = rgba.Pix[i+2] // B
			xor[j+1] = rgba.Pix[i+1] // G
			xor[j+2] = rgba.Pix[i+0] // R
			xor[j+3] = rgba.Pix[i+3] // A
		}
	}
	// AND mask 全 0：由 alpha 决定透明
	return buf, nil
}
