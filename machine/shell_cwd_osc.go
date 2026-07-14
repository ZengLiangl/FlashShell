package machine

import (
	"bytes"
	"net/url"
	"strings"
)

// oscCwdFilter 从 PTY 输出中提取 cwd（OSC 7 / OSC 777），并剥离这些序列避免刷屏。
type oscCwdFilter struct {
	pending []byte
	onCwd   func(string)
}

func newOscCwdFilter(onCwd func(string)) *oscCwdFilter {
	return &oscCwdFilter{onCwd: onCwd}
}

// Feed 输入原始 PTY 字节，返回应展示给终端的数据。
func (f *oscCwdFilter) Feed(in []byte) []byte {
	if len(in) == 0 && len(f.pending) == 0 {
		return nil
	}
	data := append(f.pending, in...)
	f.pending = nil

	var out bytes.Buffer
	i := 0
	for i < len(data) {
		if data[i] != 0x1b {
			out.WriteByte(data[i])
			i++
			continue
		}
		// ESC
		if i+1 >= len(data) {
			f.pending = append([]byte{}, data[i:]...)
			break
		}
		if data[i+1] != ']' {
			out.WriteByte(data[i])
			i++
			continue
		}
		// OSC: ESC ]
		end, termLen := findOSCEnd(data, i+2)
		if end < 0 {
			f.pending = append([]byte{}, data[i:]...)
			break
		}
		payload := data[i+2 : end]
		if cwd, ok := parseOscCwdPayload(payload); ok {
			if f.onCwd != nil && cwd != "" {
				f.onCwd(cwd)
			}
			// 剥离整个 OSC（含终止符）
			i = end + termLen
			continue
		}
		// 非 cwd OSC，原样输出
		out.Write(data[i : end+termLen])
		i = end + termLen
	}
	return out.Bytes()
}

// findOSCEnd 返回 payload 结束下标（不含终止符），以及终止符长度。
// 终止符: BEL(0x07) 或 ST(ESC \)
func findOSCEnd(data []byte, start int) (end int, termLen int) {
	for j := start; j < len(data); j++ {
		if data[j] == 0x07 {
			return j, 1
		}
		if data[j] == 0x1b && j+1 < len(data) && data[j+1] == '\\' {
			return j, 2
		}
	}
	return -1, 0
}

func parseOscCwdPayload(payload []byte) (string, bool) {
	s := string(payload)
	// OSC 777 ; cwd ; /path
	if strings.HasPrefix(s, "777;cwd;") {
		return normalizeOscPath(s[len("777;cwd;"):]), true
	}
	// OSC 7 ; file://host/path  或  file:///path
	if strings.HasPrefix(s, "7;") {
		return parseOsc7FileURI(s[2:])
	}
	return "", false
}

func parseOsc7FileURI(uri string) (string, bool) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", false
	}
	if strings.HasPrefix(uri, "file://") {
		u, err := url.Parse(uri)
		if err != nil {
			// 降级：手动剥 file://host
			rest := strings.TrimPrefix(uri, "file://")
			if idx := strings.Index(rest, "/"); idx >= 0 {
				return normalizeOscPath(rest[idx:]), true
			}
			return "", false
		}
		p := u.Path
		if p == "" {
			p = "/"
		}
		// url.Path 可能把空格解码好了
		if dec, err := url.PathUnescape(p); err == nil {
			p = dec
		}
		return normalizeOscPath(p), true
	}
	if strings.HasPrefix(uri, "/") {
		return normalizeOscPath(uri), true
	}
	return "", false
}

func normalizeOscPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	// 去掉 URI 残留的查询串
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p != "/" {
		p = strings.TrimRight(p, "/")
	}
	return p
}
