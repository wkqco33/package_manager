package pkg_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/wkqco33/package_manager/internal/archive"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/platform"
)

func TestInstallWithPackageUsesRealTarArchiver(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", "")

	installPath := t.TempDir()
	fetcher := &integrationFetcher{
		packageInfo: &pkg.Package{
			Name:    "owner/repo",
			Version: "v1.2.3",
			Source:  "https://example.com/source.tar.gz",
		},
		archive: createSourceArchive(t),
	}

	if err := pkg.InstallWithPackage(fetcher.packageInfo, fetcher, &archive.TarArchiver{}, installPath); err != nil {
		t.Fatalf("InstallWithPackage failed: %v", err)
	}

	target := filepath.Join(installPath, platform.ExecutableName("repo"))
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected installed binary at %s: %v", target, err)
	}
	packagesDir, err := config.GetPackagesDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(packagesDir, "repo-v1.2.3", "owner-repo-sha", "main.go")); err != nil {
		t.Fatalf("expected real tar extraction result: %v", err)
	}
}

type integrationFetcher struct {
	packageInfo *pkg.Package
	archive     []byte
}

func (f *integrationFetcher) GetMetadata(string) (*pkg.Package, error) {
	return f.packageInfo, nil
}

func (f *integrationFetcher) DownloadSource(*pkg.Package) (io.ReadCloser, int64, error) {
	return io.NopCloser(bytes.NewReader(f.archive)), int64(len(f.archive)), nil
}

func createSourceArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	files := map[string]string{
		"owner-repo-sha/go.mod":  "module example.com/test/repo\n\ngo 1.20\n",
		"owner-repo-sha/main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }\n",
	}
	for name, body := range files {
		header := &tar.Header{Name: name, Mode: 0644, Size: int64(len(body))}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
