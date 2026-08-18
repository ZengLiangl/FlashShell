//go:build windows

package app

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

const (
	rtIcon                   = 3
	rtGroupIcon              = 14
	loadLibraryAsDatafile    = 0x00000002
	shcneAssocChanged        = 0x08000000
	shcneUpdateItem          = 0x00002000
	shcnfIDList              = 0x0000
	shcnfPathW               = 0x0005
	shcnfFlush               = 0x1000
	windowsIconLangNeutral   = 0
	windowsIconLangUSEnglish = 1033
)

var (
	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	procSHChangeNotify      = shell32.NewProc("SHChangeNotify")
	procBeginUpdateResource = modKernel32.NewProc("BeginUpdateResourceW")
	procUpdateResource      = modKernel32.NewProc("UpdateResourceW")
	procEndUpdateResource   = modKernel32.NewProc("EndUpdateResourceW")
	procLoadLibraryExW      = modKernel32.NewProc("LoadLibraryExW")
	procFreeLibrary         = modKernel32.NewProc("FreeLibrary")
	procEnumResourceNamesW  = modKernel32.NewProc("EnumResourceNamesW")
	procFindResourceW       = modKernel32.NewProc("FindResourceW")
	procFindResourceExW     = modKernel32.NewProc("FindResourceExW")
	procLoadResource        = modKernel32.NewProc("LoadResource")
	procLockResource        = modKernel32.NewProc("LockResource")
	procSizeofResource      = modKernel32.NewProc("SizeofResource")

	windowsCOMOnce sync.Once
)

func persistFinderAppIcon(pngBytes []byte, restoreDefault bool) {
	exe := windowsCurrentExePath()
	if exe == "" {
		return
	}
	if restoreDefault {
		restoreWindowsAppFileIcon(exe)
		return
	}
	src, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return
	}
	icoBytes, err := encodeWindowsICO(src, uniquePositiveInts(16, 20, 24, 32, 40, 48, 64, 128, 256))
	if err != nil || len(icoBytes) == 0 {
		return
	}
	icoPath := persistedWindowsAppIconPath()
	if icoPath == "" {
		return
	}
	ensureOriginalExeIconBackup(exe)
	if err := os.WriteFile(icoPath, icoBytes, 0644); err != nil {
		return
	}
	for _, lnk := range listFlashDockShortcuts(exe) {
		_ = setWindowsShortcutIcon(lnk, icoPath)
	}
	_ = replaceExeIconResources(exe, icoBytes)
	notifyWindowsShellIconChanged(exe)
	notifyWindowsShellIconChanged(icoPath)
}

func restoreWindowsAppFileIcon(exe string) {
	for _, lnk := range listFlashDockShortcuts(exe) {
		_ = setWindowsShortcutIcon(lnk, exe)
	}
	if original := originalWindowsAppIconPath(); original != "" {
		if raw, err := os.ReadFile(original); err == nil && len(raw) > 0 {
			_ = replaceExeIconResources(exe, raw)
		}
	}
	if icoPath := persistedWindowsAppIconPath(); icoPath != "" {
		_ = os.Remove(icoPath)
	}
	notifyWindowsShellIconChanged(exe)
}

func windowsCurrentExePath() string {
	if p, err := resolveWindowsUpdateTarget(); err == nil {
		return p
	}
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return filepath.Clean(p)
}

func persistedWindowsAppIconPath() string {
	dir, err := appIconsDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "app.ico")
}

func originalWindowsAppIconPath() string {
	dir, err := appIconsDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "original-app.ico")
}

func ensureOriginalExeIconBackup(exe string) {
	dest := originalWindowsAppIconPath()
	if dest == "" {
		return
	}
	if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
		return
	}
	raw, err := extractExeGroupIconToICO(exe)
	if err != nil || len(raw) == 0 {
		return
	}
	_ = os.WriteFile(dest, raw, 0644)
}

func isFlashDockShortcutName(name string) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return strings.Contains(strings.ToLower(base), "flashdock")
}

func windowsShortcutSearchRoots(exePath string) []string {
	roots := make([]string, 0, 8)
	if dir := filepath.Dir(strings.TrimSpace(exePath)); dir != "" && dir != "." {
		roots = append(roots, dir)
	}
	if v := os.Getenv("APPDATA"); v != "" {
		roots = append(roots,
			filepath.Join(v, "Microsoft", "Windows", "Start Menu", "Programs"),
			filepath.Join(v, "Microsoft", "Internet Explorer", "Quick Launch", "User Pinned", "TaskBar"),
		)
	}
	if v := os.Getenv("ProgramData"); v != "" {
		roots = append(roots, filepath.Join(v, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if v := os.Getenv("PUBLIC"); v != "" {
		roots = append(roots, filepath.Join(v, "Desktop"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, "Desktop"))
		roots = append(roots, filepath.Join(home, "OneDrive", "Desktop"))
	}
	return roots
}

func listFlashDockShortcuts(exePath string) []string {
	exePath = filepath.Clean(exePath)
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
	add := func(p string) {
		p = filepath.Clean(p)
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			return
		}
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	for _, root := range windowsShortcutSearchRoots(exePath) {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".lnk") {
				return nil
			}
			if isFlashDockShortcutName(d.Name()) {
				add(path)
				return nil
			}
			if strings.EqualFold(filepath.Dir(path), filepath.Dir(exePath)) {
				if tgt := windowsShortcutTarget(path); tgt != "" && strings.EqualFold(filepath.Clean(tgt), exePath) {
					add(path)
				}
			}
			return nil
		})
	}
	return out
}

func ensureWindowsCOM() {
	windowsCOMOnce.Do(func() {
		_ = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	})
}

func setWindowsShortcutIcon(lnkPath, iconPath string) error {
	ensureWindowsCOM()
	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer shell.Release()
	sc, err := oleutil.CallMethod(shell, "CreateShortcut", lnkPath)
	if err != nil {
		return err
	}
	shortcut := sc.ToIDispatch()
	if shortcut == nil {
		return errors.New("无法打开快捷方式")
	}
	defer shortcut.Release()
	if _, err := oleutil.PutProperty(shortcut, "IconLocation", iconPath+",0"); err != nil {
		return err
	}
	_, err = oleutil.CallMethod(shortcut, "Save")
	return err
}

func windowsShortcutTarget(lnkPath string) string {
	ensureWindowsCOM()
	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return ""
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return ""
	}
	defer shell.Release()
	sc, err := oleutil.CallMethod(shell, "CreateShortcut", lnkPath)
	if err != nil {
		return ""
	}
	shortcut := sc.ToIDispatch()
	if shortcut == nil {
		return ""
	}
	defer shortcut.Release()
	val, err := oleutil.GetProperty(shortcut, "TargetPath")
	if err != nil {
		return ""
	}
	defer val.Clear()
	return strings.TrimSpace(val.ToString())
}

func notifyWindowsShellIconChanged(path string) {
	path = strings.TrimSpace(path)
	if path != "" {
		p, err := windows.UTF16PtrFromString(path)
		if err == nil {
			procSHChangeNotify.Call(shcneUpdateItem, shcnfPathW|shcnfFlush, uintptr(unsafe.Pointer(p)), 0)
		}
	}
	procSHChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0)
}

type icoResEntry struct {
	width, height, colorCount, reserved byte
	planes, bitCount                    uint16
	bytesInRes                          uint32
	id                                  uint16
	data                                []byte
}

func parseICOForResources(ico []byte) ([]icoResEntry, error) {
	if len(ico) < 6 {
		return nil, errors.New("ico 过短")
	}
	if binary.LittleEndian.Uint16(ico[2:]) != 1 {
		return nil, errors.New("不是图标 ico")
	}
	n := int(binary.LittleEndian.Uint16(ico[4:]))
	if n <= 0 {
		return nil, errors.New("ico 无帧")
	}
	out := make([]icoResEntry, 0, n)
	for i := 0; i < n; i++ {
		off := 6 + i*16
		if off+16 > len(ico) {
			break
		}
		dataOff := int(binary.LittleEndian.Uint32(ico[off+12:]))
		dataLen := int(binary.LittleEndian.Uint32(ico[off+8:]))
		if dataOff < 0 || dataLen <= 0 || dataOff+dataLen > len(ico) {
			continue
		}
		entry := icoResEntry{
			width:      ico[off],
			height:     ico[off+1],
			colorCount: ico[off+2],
			reserved:   ico[off+3],
			planes:     binary.LittleEndian.Uint16(ico[off+4:]),
			bitCount:   binary.LittleEndian.Uint16(ico[off+6:]),
			bytesInRes: uint32(dataLen),
			id:         uint16(i + 1),
			data:       append([]byte(nil), ico[dataOff:dataOff+dataLen]...),
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil, errors.New("ico 帧无效")
	}
	return out, nil
}

func buildGrpIconDir(entries []icoResEntry) []byte {
	buf := make([]byte, 6+14*len(entries))
	binary.LittleEndian.PutUint16(buf[2:], 1)
	binary.LittleEndian.PutUint16(buf[4:], uint16(len(entries)))
	for i, e := range entries {
		off := 6 + i*14
		buf[off] = e.width
		buf[off+1] = e.height
		buf[off+2] = e.colorCount
		buf[off+3] = e.reserved
		binary.LittleEndian.PutUint16(buf[off+4:], e.planes)
		binary.LittleEndian.PutUint16(buf[off+6:], e.bitCount)
		binary.LittleEndian.PutUint32(buf[off+8:], e.bytesInRes)
		binary.LittleEndian.PutUint16(buf[off+12:], e.id)
	}
	return buf
}

func buildICOFromGrpEntries(entries []icoResEntry) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(entries)))
	offset := 6 + 16*len(entries)
	for _, e := range entries {
		buf.WriteByte(e.width)
		buf.WriteByte(e.height)
		buf.WriteByte(e.colorCount)
		buf.WriteByte(e.reserved)
		_ = binary.Write(&buf, binary.LittleEndian, e.planes)
		_ = binary.Write(&buf, binary.LittleEndian, e.bitCount)
		_ = binary.Write(&buf, binary.LittleEndian, e.bytesInRes)
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(e.data)
	}
	for _, e := range entries {
		buf.Write(e.data)
	}
	return buf.Bytes()
}

func replaceExeIconResources(exe string, ico []byte) error {
	entries, err := parseICOForResources(ico)
	if err != nil {
		return err
	}
	exe16, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	hUpdate, _, callErr := procBeginUpdateResource.Call(uintptr(unsafe.Pointer(exe16)), 0)
	if hUpdate == 0 {
		return callErr
	}
	discard := uintptr(1)
	defer func() {
		procEndUpdateResource.Call(hUpdate, discard)
	}()

	oldGroups, oldIcons := enumExeIconResourceIDs(exe)
	for _, id := range oldIcons {
		if err := updateExeResource(hUpdate, rtIcon, id, nil); err != nil {
			return err
		}
	}
	for _, id := range oldGroups {
		if err := updateExeResource(hUpdate, rtGroupIcon, id, nil); err != nil {
			return err
		}
	}
	for _, e := range entries {
		if err := updateExeResource(hUpdate, rtIcon, e.id, e.data); err != nil {
			return err
		}
	}
	groupID := uint16(1)
	if len(oldGroups) > 0 {
		groupID = oldGroups[0]
	}
	if err := updateExeResource(hUpdate, rtGroupIcon, groupID, buildGrpIconDir(entries)); err != nil {
		return err
	}
	discard = 0
	return nil
}

func updateExeResource(hUpdate uintptr, resType, id uint16, data []byte) error {
	var ptr uintptr
	var size uintptr
	if len(data) > 0 {
		ptr = uintptr(unsafe.Pointer(&data[0]))
		size = uintptr(len(data))
	}
	r, _, err := procUpdateResource.Call(hUpdate, uintptr(resType), uintptr(id), windowsIconLangNeutral, ptr, size)
	if r == 0 {
		r, _, err = procUpdateResource.Call(hUpdate, uintptr(resType), uintptr(id), windowsIconLangUSEnglish, ptr, size)
	}
	if r == 0 {
		return err
	}
	return nil
}

func enumExeIconResourceIDs(exe string) (groups, icons []uint16) {
	hmod := loadExeAsDatafile(exe)
	if hmod == 0 {
		return nil, nil
	}
	defer procFreeLibrary.Call(hmod)
	return enumResourceIDs(hmod, rtGroupIcon), enumResourceIDs(hmod, rtIcon)
}

func loadExeAsDatafile(exe string) uintptr {
	exe16, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 0
	}
	hmod, _, _ := procLoadLibraryExW.Call(uintptr(unsafe.Pointer(exe16)), 0, loadLibraryAsDatafile)
	return hmod
}

func enumResourceIDs(hmod uintptr, resType uint16) []uint16 {
	var ids []uint16
	cb := syscall.NewCallback(func(_, _, lpName, _ uintptr) uintptr {
		if lpName>>16 == 0 {
			ids = append(ids, uint16(lpName))
		}
		return 1
	})
	procEnumResourceNamesW.Call(hmod, uintptr(resType), cb, 0)
	return ids
}

func extractExeGroupIconToICO(exe string) ([]byte, error) {
	hmod := loadExeAsDatafile(exe)
	if hmod == 0 {
		return nil, errors.New("无法读取 exe 资源")
	}
	defer procFreeLibrary.Call(hmod)
	groups := enumResourceIDs(hmod, rtGroupIcon)
	if len(groups) == 0 {
		return nil, errors.New("exe 无组图标")
	}
	raw := loadResourceBytes(hmod, rtGroupIcon, groups[0])
	entries, err := parseGrpIconDir(raw)
	if err != nil {
		return nil, err
	}
	out := make([]icoResEntry, 0, len(entries))
	for _, e := range entries {
		data := loadResourceBytes(hmod, rtIcon, e.id)
		if len(data) == 0 {
			continue
		}
		e.data = data
		e.bytesInRes = uint32(len(data))
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, errors.New("exe 图标帧为空")
	}
	return buildICOFromGrpEntries(out), nil
}

func parseGrpIconDir(raw []byte) ([]icoResEntry, error) {
	if len(raw) < 6 {
		return nil, errors.New("组图标过短")
	}
	n := int(binary.LittleEndian.Uint16(raw[4:]))
	if n <= 0 {
		return nil, errors.New("组图标无帧")
	}
	out := make([]icoResEntry, 0, n)
	for i := 0; i < n; i++ {
		off := 6 + i*14
		if off+14 > len(raw) {
			break
		}
		out = append(out, icoResEntry{
			width:      raw[off],
			height:     raw[off+1],
			colorCount: raw[off+2],
			reserved:   raw[off+3],
			planes:     binary.LittleEndian.Uint16(raw[off+4:]),
			bitCount:   binary.LittleEndian.Uint16(raw[off+6:]),
			bytesInRes: binary.LittleEndian.Uint32(raw[off+8:]),
			id:         binary.LittleEndian.Uint16(raw[off+12:]),
		})
	}
	if len(out) == 0 {
		return nil, errors.New("组图标帧无效")
	}
	return out, nil
}

func loadResourceBytes(hmod uintptr, resType, id uint16) []byte {
	hrs, _, _ := procFindResourceExW.Call(hmod, uintptr(resType), uintptr(id), windowsIconLangNeutral)
	if hrs == 0 {
		hrs, _, _ = procFindResourceExW.Call(hmod, uintptr(resType), uintptr(id), windowsIconLangUSEnglish)
	}
	if hrs == 0 {
		hrs, _, _ = procFindResourceW.Call(hmod, uintptr(id), uintptr(resType))
	}
	if hrs == 0 {
		return nil
	}
	hglob, _, _ := procLoadResource.Call(hmod, hrs)
	if hglob == 0 {
		return nil
	}
	ptr, _, _ := procLockResource.Call(hglob)
	size, _, _ := procSizeofResource.Call(hmod, hrs)
	if ptr == 0 || size == 0 {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(size))...)
}
