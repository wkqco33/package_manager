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
	// 메모리 내 mock tar.gz 생성
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

	// 압축 해제용 임시 디렉터리
	tmpDir, err := os.MkdirTemp("", "ppm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	archiver := &TarArchiver{}
	if err := archiver.Extract(&buf, tmpDir); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// 파일 검증
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

func TestTarArchiver_Link(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ppm-link-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Case 1: Binary in top level
	binName := "my-app"
	binPath := filepath.Join(tmpDir, binName)
	if err := os.WriteFile(binPath, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(tmpDir, "bin")
	targetLink := filepath.Join(targetDir, binName)

	archiver := &TarArchiver{}
	if err := archiver.Link(tmpDir, binName, targetLink); err != nil {
		t.Fatalf("Link failed: %v", err)
	}

	if _, err := os.Lstat(targetLink); err != nil {
		t.Errorf("Link not created: %v", err)
	}

	// Case 2: Binary in nested directory (common for GitHub tarballs)
	nestedDir := filepath.Join(tmpDir, "owner-repo-sha")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedBinPath := filepath.Join(nestedDir, "nested-app")
	if err := os.WriteFile(nestedBinPath, []byte("nested-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	nestedTargetLink := filepath.Join(targetDir, "nested-app")
	if err := archiver.Link(tmpDir, "nested-app", nestedTargetLink); err != nil {
		t.Fatalf("Nested link failed: %v", err)
	}

	if _, err := os.Lstat(nestedTargetLink); err != nil {
		t.Errorf("Nested link not created: %v", err)
	}
}
