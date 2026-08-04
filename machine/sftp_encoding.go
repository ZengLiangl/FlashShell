package machine

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func decodeBytesAsGB18030(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	r := transform.NewReader(bytes.NewReader(raw), simplifiedchinese.GB18030.NewDecoder())
	out, err := io.ReadAll(r)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

func encodeNameAsGB18030(name string) string {
	if name == "" {
		return name
	}
	r := transform.NewReader(strings.NewReader(name), simplifiedchinese.GB18030.NewEncoder())
	out, err := io.ReadAll(r)
	if err != nil {
		return name
	}
	return string(out)
}
