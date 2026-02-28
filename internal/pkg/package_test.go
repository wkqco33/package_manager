package pkg

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockFetcher는 RegistryFetcher 목 구현체입니다.
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

// MockArchiver는 Archiver 목 구현체입니다.
type MockArchiver struct {
	extracted bool
	linked    bool
}

func (a *MockArchiver) Extract(r io.Reader, dest string) error {
	a.extracted = true
	return os.MkdirAll(dest, 0755)
}

func (a *MockArchiver) Link(dir, name, target string) error {
	a.linked = true
	return nil
}

func TestInstall(t *testing.T) {
	// Mock HOME to a temp directory for this test
	tempHome, _ := os.MkdirTemp("", "ppm-home-*")
	defer os.RemoveAll(tempHome)
	
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", oldHome)

	mockPkg := &Package{
		Name:    "test/repo",
		Version: "v1.0.0",
		Source:  "http://example.com/tarball.tar.gz",
	}
	fetcher := &MockFetcher{pkg: mockPkg}
	archiver := &MockArchiver{}

	// Create temp bin dir
	binDir, _ := os.MkdirTemp("", "ppm-test-bin-*")
	defer os.RemoveAll(binDir)

	// Normal installation
	err := Install("test/repo", fetcher, archiver, binDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !archiver.extracted {
		t.Error("Archiver.Extract was not called")
	}
	if !archiver.linked {
		t.Error("Archiver.Link was not called")
	}

	// Reset mock and try again (should skip extraction)
	archiver.extracted = false
	archiver.linked = false

	err = Install("test/repo", fetcher, archiver, binDir)
	if err != nil {
		t.Fatalf("Second install failed: %v", err)
	}

	if archiver.extracted {
		t.Error("Archiver.Extract should have been skipped")
	}
	if !archiver.linked {
		t.Error("Archiver.Link should still be called to ensure link")
	}
}

func TestSaveAndListMetadata(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ppm-meta-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgDir := filepath.Join(tmpDir, "test-repo-v1.0.0")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	p := &Package{
		Name:    "test/repo",
		Version: "v1.0.0",
		Source:  "http://example.com/tarball.tar.gz",
	}

	if err := saveMetadata(pkgDir, p); err != nil {
		t.Fatalf("saveMetadata failed: %v", err)
	}

	// Verify file exists
	metaFile := filepath.Join(pkgDir, "ppm-meta.json")
	if _, err := os.Stat(metaFile); os.IsNotExist(err) {
		t.Error("ppm-meta.json was not created")
	}

	// Test ListInstalled
	installed, err := ListInstalled(tmpDir)
	if err != nil {
		t.Fatalf("ListInstalled failed: %v", err)
	}

	if len(installed) != 1 {
		t.Errorf("Expected 1 package, got %d", len(installed))
	}
	if installed[0].Name != "test/repo" {
		t.Errorf("Expected package name test/repo, got %s", installed[0].Name)
	}
}

func TestListInstalledWithLegacy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ppm-legacy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a legacy package (dir without metadata)
	legacyDir := filepath.Join(tmpDir, "legacy-pkg")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}

	installed, err := ListInstalled(tmpDir)
	if err != nil {
		t.Fatalf("ListInstalled failed: %v", err)
	}

	foundLegacy := false
	for _, p := range installed {
		if p.Name == "legacy-pkg" && p.Version == "" {
			foundLegacy = true
			break
		}
	}

	if !foundLegacy {
		t.Error("Legacy package not found in list")
	}
}
