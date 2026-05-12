package pkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ppm/internal/apperr"
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

func TestInstallWithPackage_BuildsFromGoSourceTarball(t *testing.T) {
	tempHome, _ := os.MkdirTemp("", "ppm-home-*")
	defer os.RemoveAll(tempHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", oldHome)

	binDir, _ := os.MkdirTemp("", "ppm-test-bin-*")
	defer os.RemoveAll(binDir)

	fetcher := &sourceArchiveFetcher{
		pkg: &Package{
			Name:    "owner/repo",
			Version: "v1.2.3",
			Source:  "https://example.com/source.tar.gz",
		},
		body: createGoSourceTarball(t),
	}

	p, err := fetcher.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	archiver := &sourceBuildArchiver{}
	if err := InstallWithPackage(p, fetcher, archiver, binDir); err != nil {
		t.Fatalf("InstallWithPackage failed: %v", err)
	}

	target := filepath.Join(binDir, "repo")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("Expected installed binary at %s: %v", target, err)
	}
}

type sourceArchiveFetcher struct {
	pkg  *Package
	body []byte
}

type sourceBuildArchiver struct{}

func (f *sourceArchiveFetcher) GetMetadata(name string) (*Package, error) {
	return f.pkg, nil
}

func (f *sourceArchiveFetcher) DownloadSource(p *Package) (io.ReadCloser, int64, error) {
	return io.NopCloser(bytes.NewReader(f.body)), int64(len(f.body)), nil
}

func (a *sourceBuildArchiver) Extract(r io.Reader, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
}

func (a *sourceBuildArchiver) Link(dir, name, target string) error {
	src := filepath.Join(dir, name)
	if _, err := os.Stat(src); err != nil {
		return apperr.New(apperr.CodeArchive, "executable %s not found in directory %s", name, dir)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte("linked"), 0755)
}

func createGoSourceTarball(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	files := map[string]string{
		"owner-repo-sha/go.mod":  "module example.com/test/repo\n\ngo 1.20\n",
		"owner-repo-sha/main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }\n",
	}

	for name, body := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("tar close failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}

	return buf.Bytes()
}
