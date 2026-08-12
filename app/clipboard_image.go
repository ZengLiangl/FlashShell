package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SaveClipboardImageForUpload 将剪贴板图片（dataURL 或纯 base64）落盘为临时 PNG，供 SFTP 上传。
func (a *App) SaveClipboardImageForUpload(dataURL string) (string, error) {
	raw := strings.TrimSpace(dataURL)
	if raw == "" {
		return "", fmt.Errorf("剪贴板图片为空")
	}
	payload := raw
	if i := strings.Index(raw, ","); i >= 0 {
		payload = raw[i+1:]
	}
	payload = strings.TrimSpace(payload)
	bin, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %w", err)
	}
	if len(bin) == 0 {
		return "", fmt.Errorf("图片数据为空")
	}
	dir := filepath.Join(os.TempDir(), "flashdock-paste-images")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("paste-%d.png", time.Now().UnixNano())
	out := filepath.Join(dir, name)
	if err := os.WriteFile(out, bin, 0o644); err != nil {
		return "", err
	}
	return out, nil
}
