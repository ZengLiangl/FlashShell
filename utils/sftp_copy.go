package utils

import "io"

// concurrentReaderFrom matches github.com/pkg/sftp.File.ReadFromWithConcurrency.
// Using an interface keeps utils free of an sftp import while still engaging pipelined writes.
type concurrentReaderFrom interface {
	ReadFromWithConcurrency(r io.Reader, concurrency int) (int64, error)
}

// CopySFTPUpload copies src into dst, preferring SFTP pipelined concurrent writes when available.
// concurrency 0 lets pkg/sftp use MaxConcurrentRequestsPerFile (default 64).
func CopySFTPUpload(dst io.Writer, src io.Reader) (int64, error) {
	if rf, ok := dst.(concurrentReaderFrom); ok {
		return rf.ReadFromWithConcurrency(src, 0)
	}
	if rf, ok := dst.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}
	return CopyBuffer(dst, src)
}

// CopySFTPDownload copies src into dst, preferring *sftp.File.WriteTo (pipelined concurrent reads).
// Progress wrappers must sit on dst so src keeps its WriterTo implementation.
func CopySFTPDownload(dst io.Writer, src io.Reader) (int64, error) {
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	return CopyBuffer(dst, src)
}
