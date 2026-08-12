package app

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"FlashDock/data"

	"embed"

	xdraw "golang.org/x/image/draw"
)

//go:embed assets/appicon.png
var embeddedDefaultAppIcon []byte

//go:embed assets/dock-*.png
var dockPresetFS embed.FS

// AppIconPresetInfo 前端展示用的图标预设
type AppIconPresetInfo struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Preview  string `json:"preview"` // data:image/png;base64,...
	IsCustom bool   `json:"isCustom"`
}

type appIconPresetDef struct {
	ID       string
	Label    string
	FileName string // dock-*.png；空表示使用 appicon.png
}

var appIconPresetDefs = []appIconPresetDef{
	{ID: "default", Label: "默认"},
	{ID: "helm", Label: "闪舵", FileName: "dock-helm.png"},
	{ID: "pipeline", Label: "任务流水线", FileName: "dock-pipeline.png"},
	{ID: "shell", Label: "Shell 终端", FileName: "dock-shell.png"},
	{ID: "split", Label: "分屏", FileName: "dock-split.png"},
	{ID: "broadcast", Label: "广播", FileName: "dock-broadcast.png"},
	{ID: "sftp", Label: "SFTP 传输", FileName: "dock-sftp.png"},
	{ID: "tunnel", Label: "SSH 隧道", FileName: "dock-tunnel.png"},
	{ID: "yaml", Label: "YAML 配置", FileName: "dock-yaml.png"},
	{ID: "parallel", Label: "双模并行", FileName: "dock-parallel.png"},
	{ID: "secure", Label: "安全信任", FileName: "dock-secure.png"},
}

func appIconsDir() (string, error) {
	home, err := data.ConfigHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "icons")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// CustomAppIconPath 用户自定义 Dock 图标落盘路径
func CustomAppIconPath() (string, error) {
	dir, err := appIconsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "custom.png"), nil
}

func customAppIconExists() bool {
	path, err := CustomAppIconPath()
	if err != nil {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func resolveAppIconPreset(preset string) string {
	preset = data.NormalizeAppIconPreset(preset)
	if preset == "custom" && !customAppIconExists() {
		return "default"
	}
	return preset
}

func pngDataURL(pngBytes []byte) string {
	if len(pngBytes) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
}

// dockPreviewSize 设置页预览边长。原图可达数百 KB～1MB，直接 base64 会拖慢设置打开。
const dockPreviewSize = 96

var (
	appIconPresetCacheMu sync.Mutex
	appIconPresetCache   []AppIconPresetInfo
)

func invalidateAppIconPresetCache() {
	appIconPresetCacheMu.Lock()
	appIconPresetCache = nil
	appIconPresetCacheMu.Unlock()
}

// warmAppIconPresetCache 启动时后台生成缩略图，避免首次打开设置卡顿
func warmAppIconPresetCache() {
	go func() {
		_ = listAppIconPresets()
	}()
}

func resizePNGPreview(src image.Image, size int) []byte {
	if src == nil || size <= 0 {
		return nil
	}
	sb := src.Bounds()
	if sb.Dx() <= 0 || sb.Dy() <= 0 {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, sb, xdraw.Over, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil
	}
	return buf.Bytes()
}

func previewDataURLFromPNG(pngBytes []byte) string {
	if len(pngBytes) == 0 {
		return ""
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		// 解码失败时退回原图 data URL（自定义图标可能非标准）
		return pngDataURL(pngBytes)
	}
	thumb := resizePNGPreview(img, dockPreviewSize)
	if len(thumb) == 0 {
		return pngDataURL(pngBytes)
	}
	return pngDataURL(thumb)
}

func presetPNGBytes(id string) ([]byte, error) {
	id = resolveAppIconPreset(id)
	if id == "custom" {
		path, err := CustomAppIconPath()
		if err != nil {
			return nil, err
		}
		return os.ReadFile(path)
	}
	if id == "default" {
		if len(embeddedDefaultAppIcon) > 0 {
			return embeddedDefaultAppIcon, nil
		}
		return nil, fmt.Errorf("缺少默认图标")
	}
	for _, def := range appIconPresetDefs {
		if def.ID != id || def.FileName == "" {
			continue
		}
		return dockPresetFS.ReadFile("assets/" + def.FileName)
	}
	return embeddedDefaultAppIcon, nil
}

func listAppIconPresets() []AppIconPresetInfo {
	appIconPresetCacheMu.Lock()
	if appIconPresetCache != nil {
		cached := appIconPresetCache
		appIconPresetCacheMu.Unlock()
		return cached
	}
	appIconPresetCacheMu.Unlock()

	out := make([]AppIconPresetInfo, 0, len(appIconPresetDefs)+1)
	for _, def := range appIconPresetDefs {
		pngBytes, err := presetPNGBytes(def.ID)
		if err != nil || len(pngBytes) == 0 {
			continue
		}
		out = append(out, AppIconPresetInfo{
			ID:      def.ID,
			Label:   def.Label,
			Preview: previewDataURLFromPNG(pngBytes),
		})
	}
	if customAppIconExists() {
		if pngBytes, err := presetPNGBytes("custom"); err == nil {
			out = append(out, AppIconPresetInfo{
				ID:       "custom",
				Label:    "自定义",
				Preview:  previewDataURLFromPNG(pngBytes),
				IsCustom: true,
			})
		}
	}

	appIconPresetCacheMu.Lock()
	if appIconPresetCache != nil {
		cached := appIconPresetCache
		appIconPresetCacheMu.Unlock()
		return cached
	}
	appIconPresetCache = out
	appIconPresetCacheMu.Unlock()
	return out
}

func saveCustomAppIconFromFile(srcPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	if srcPath == "" {
		return fmt.Errorf("未选择文件")
	}
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("读取图片失败: %w", err)
	}
	img, err := decodeImageBytes(raw, srcPath)
	if err != nil {
		return err
	}
	dest, err := CustomAppIconPath()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("编码 PNG 失败: %w", err)
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("保存自定义图标失败: %w", err)
	}
	invalidateAppIconPresetCache()
	return nil
}

func decodeImageBytes(raw []byte, nameHint string) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err == nil {
		return img, nil
	}
	ext := strings.ToLower(filepath.Ext(nameHint))
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(bytes.NewReader(raw))
	case ".png":
		img, err = png.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return nil, fmt.Errorf("不支持的图片格式，请使用 PNG 或 JPG")
	}
	return img, nil
}
