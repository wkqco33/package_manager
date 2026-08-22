package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPackageCleanerCleanAllRemovesPackageDirectoryAndRelatedLinks(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("symlink semantics differ on Windows")
	}
	packagesDir := t.TempDir()
	installDir := t.TempDir()
	targetDir := filepath.Join(packagesDir, "repo-v1")
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(installDir, "repo")
	if err := os.Symlink(filepath.Join(targetDir, "repo"), linkPath); err != nil {
		t.Fatal(err)
	}

	var removedLink string
	result, err := (PackageCleaner{
		ReadDir:   os.ReadDir,
		RemoveAll: os.RemoveAll,
		Readlink:  os.Readlink,
		Remove: func(path string) error {
			removedLink = path
			return os.Remove(path)
		},
	}).CleanAll(packagesDir, installDir)
	if err != nil {
		t.Fatalf("CleanAll failed: %v", err)
	}
	if !result.PackagesRemoved || result.LinksRemoved != 1 || removedLink != linkPath {
		t.Fatalf("result = %+v, removedLink = %q", result, removedLink)
	}
}

func TestPackageCleanerCleanAllReturnsRemovalError(t *testing.T) {
	wantErr := errors.New("remove failed")
	result, err := (PackageCleaner{
		RemoveAll: func(string) error { return wantErr },
	}).CleanAll("packages", "install")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped removal error", err)
	}
	if result.PackagesRemoved {
		t.Fatal("packages should not be marked removed after failure")
	}
}

func TestPackageCleanerRemovesOnlyInactiveDirectories(t *testing.T) {
	packagesDir := t.TempDir()
	for _, name := range []string{"active-v1", "old-v1"} {
		if err := os.Mkdir(filepath.Join(packagesDir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(packagesDir, "metadata.json"), []byte("metadata"), 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := (PackageCleaner{
		ReadDir:   os.ReadDir,
		RemoveAll: os.RemoveAll,
		CollectActiveDirs: func(_, _ string, _ []os.DirEntry) map[string]bool {
			return map[string]bool{"active-v1": true}
		},
	}).CleanUnused(packagesDir, t.TempDir())
	if err != nil {
		t.Fatalf("CleanUnused failed: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(packagesDir, "active-v1")); err != nil {
		t.Fatalf("active directory was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(packagesDir, "old-v1")); !os.IsNotExist(err) {
		t.Fatalf("inactive directory still exists: %v", err)
	}
}

func TestPackageCleanerHandlesMissingDirectory(t *testing.T) {
	removed, err := (PackageCleaner{
		CollectActiveDirs: func(string, string, []os.DirEntry) map[string]bool { return nil },
	}).CleanUnused(filepath.Join(t.TempDir(), "missing"), t.TempDir())
	if err != nil || removed != 0 {
		t.Fatalf("result = removed:%d err:%v, want no-op", removed, err)
	}
}

func TestPackageCleanerReturnsRemovalError(t *testing.T) {
	packagesDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(packagesDir, "old-v1"), 0755); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("remove failed")
	removed, err := (PackageCleaner{
		RemoveAll: func(string) error { return wantErr },
		CollectActiveDirs: func(string, string, []os.DirEntry) map[string]bool {
			return map[string]bool{}
		},
	}).CleanUnused(packagesDir, t.TempDir())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped removal error", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0 after failure", removed)
	}
}
