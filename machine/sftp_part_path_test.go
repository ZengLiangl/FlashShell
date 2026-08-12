package machine

import "testing"

func TestUploadPartPath(t *testing.T) {
	got := uploadPartPath("/home/u/a.txt")
	want := "/home/u/.a.txt.flashdock.part"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = uploadPartPath("a.txt")
	want = ".a.txt.flashdock.part"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDownloadPartPath(t *testing.T) {
	got := downloadPartPath("/tmp/foo.bin")
	want := "/tmp/foo.bin.flashdock.part"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
