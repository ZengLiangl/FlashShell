package utils

import (
	"bytes"
	"io"
	"testing"
)

type stubConcurrentWriter struct {
	wrote       []byte
	concurrency int
	called      bool
}

func (s *stubConcurrentWriter) Write(p []byte) (int, error) {
	s.wrote = append(s.wrote, p...)
	return len(p), nil
}

func (s *stubConcurrentWriter) ReadFromWithConcurrency(r io.Reader, concurrency int) (int64, error) {
	s.called = true
	s.concurrency = concurrency
	return io.Copy(bytes.NewBuffer(nil), r)
}

func TestCopySFTPUpload_UsesConcurrentReadFrom(t *testing.T) {
	src := bytes.NewReader(bytes.Repeat([]byte("x"), 1024))
	dst := &stubConcurrentWriter{}

	n, err := CopySFTPUpload(dst, src)
	if err != nil {
		t.Fatalf("CopySFTPUpload: %v", err)
	}
	if n != 1024 {
		t.Fatalf("copied %d, want 1024", n)
	}
	if !dst.called {
		t.Fatal("expected ReadFromWithConcurrency to be used")
	}
	if dst.concurrency != 0 {
		t.Fatalf("concurrency arg = %d, want 0 (library default)", dst.concurrency)
	}
}

func TestCopySFTPUpload_FallsBackToCopyBuffer(t *testing.T) {
	src := bytes.NewReader([]byte("hello"))
	var dst bytes.Buffer

	n, err := CopySFTPUpload(&dst, src)
	if err != nil {
		t.Fatalf("CopySFTPUpload: %v", err)
	}
	if n != 5 || dst.String() != "hello" {
		t.Fatalf("got n=%d data=%q", n, dst.String())
	}
}

type stubWriterTo struct {
	payload []byte
	called  bool
}

func (s *stubWriterTo) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (s *stubWriterTo) WriteTo(w io.Writer) (int64, error) {
	s.called = true
	n, err := w.Write(s.payload)
	return int64(n), err
}

func TestCopySFTPDownload_UsesWriteTo(t *testing.T) {
	src := &stubWriterTo{payload: []byte("abcdef")}
	var dst bytes.Buffer

	n, err := CopySFTPDownload(&dst, src)
	if err != nil {
		t.Fatalf("CopySFTPDownload: %v", err)
	}
	if !src.called {
		t.Fatal("expected WriteTo to be used")
	}
	if n != 6 || dst.String() != "abcdef" {
		t.Fatalf("got n=%d data=%q", n, dst.String())
	}
}

func TestCopySFTPDownload_FallsBackToCopyBuffer(t *testing.T) {
	src := bytes.NewReader([]byte("hello"))
	var dst bytes.Buffer

	n, err := CopySFTPDownload(&dst, src)
	if err != nil {
		t.Fatalf("CopySFTPDownload: %v", err)
	}
	if n != 5 || dst.String() != "hello" {
		t.Fatalf("got n=%d data=%q", n, dst.String())
	}
}
