package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestTarArchiver_Extract(t *testing.T) {
	// Create a mock tar.gz in memory
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	files := []struct {
		Name string
		Body string
	}{
		{"test.txt", "hello world"},
		{"subdir/inner.txt", "inner content"},
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0644,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gzw.Close()

	// Temp dir for extraction
	tmpDir, err := os.MkdirTemp("", "ppm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	archiver := &TarArchiver{}
	if err := archiver.Extract(&buf, tmpDir); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify files
	for _, file := range files {
		path := filepath.Join(tmpDir, file.Name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("File %s not found: %v", file.Name, err)
			continue
		}
		if string(content) != file.Body {
			t.Errorf("File %s content mismatch: got %s, want %s", file.Name, string(content), file.Body)
		}
	}
}
