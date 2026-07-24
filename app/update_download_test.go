package app

import (
	"net/http"
	"testing"
)

func TestParseContentRangeTotal(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"bytes 100-199/1000", 1000},
		{"bytes 0-0/42", 42},
		{"bytes 10-20/*", 0},
		{"", 0},
		{"invalid", 0},
	}
	for _, tc := range cases {
		if got := parseContentRangeTotal(tc.in); got != tc.want {
			t.Fatalf("parseContentRangeTotal(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestResolveDownloadTotalPrefersKnownSize(t *testing.T) {
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 12, // 代理常返回偏小长度
		Header:        http.Header{"Content-Range": []string{"bytes 0-11/12"}},
	}
	if got := resolveDownloadTotal(1_000_000, 0, resp); got != 1_000_000 {
		t.Fatalf("prefer knownSize: got %d", got)
	}
	resp206 := &http.Response{
		StatusCode:    http.StatusPartialContent,
		ContentLength: 100,
		Header:        http.Header{"Content-Range": []string{"bytes 50-149/500"}},
	}
	if got := resolveDownloadTotal(0, 50, resp206); got != 500 {
		t.Fatalf("content-range total: got %d want 500", got)
	}
	if got := resolveDownloadTotal(0, 50, &http.Response{
		StatusCode:    http.StatusPartialContent,
		ContentLength: 100,
	}); got != 150 {
		t.Fatalf("offset+length: got %d want 150", got)
	}
}
