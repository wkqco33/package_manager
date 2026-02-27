package pkg

import (
	"io"
	"strings"
	"testing"
)

// MockFetcher implements RegistryFetcher
type MockFetcher struct {
	pkg *Package
	err error
}

func (f *MockFetcher) GetMetadata(name string) (*Package, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pkg, nil
}

func (f *MockFetcher) DownloadSource(p *Package) (io.ReadCloser, int64, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return io.NopCloser(strings.NewReader("mock data")), int64(9), nil
}

// MockArchiver implements Archiver
type MockArchiver struct {
	extracted bool
	linked    bool
}

func (a *MockArchiver) Extract(r io.Reader, dest string) error {
	a.extracted = true
	return nil
}

func (a *MockArchiver) Link(dir, name, target string) error {
	a.linked = true
	return nil
}

func TestInstall(t *testing.T) {
	mockPkg := &Package{
		Name:    "test/repo",
		Version: "v1.0.0",
		Source:  "http://example.com/tarball.tar.gz",
	}
	fetcher := &MockFetcher{pkg: mockPkg}
	archiver := &MockArchiver{}

	// /tmp/ppm-test-bin을 가상 설치 경로로 사용
	err := Install("test/repo", fetcher, archiver, "/tmp/ppm-test-bin")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !archiver.extracted {
		t.Error("Archiver.Extract was not called")
	}
	if !archiver.linked {
		t.Error("Archiver.Link was not called")
	}
}
