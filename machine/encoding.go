package machine

import (
	"bytes"
	"unicode/utf8"
)

// convertToUTF8 尝试将字节数组转换为 UTF-8 字符串
func convertToUTF8(data []byte) string {
	// 首先尝试直接作为 UTF-8
	if utf8.Valid(data) {
		return string(data)
	}

	// 如果不是有效的 UTF-8，逐字节处理
	var buf bytes.Buffer
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// 跳过无效字节
			data = data[1:]
			continue
		}
		buf.WriteRune(r)
		data = data[size:]
	}

	return buf.String()
}
