package lockfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wkqco33/package_manager/internal/pkg"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ppm.lock")
	input := []*pkg.Package{{Name: "owner/tool", Version: "v1.2.3", Source: "https://example.test/tool.tar.gz", Dependencies: []string{"owner/base"}}}
	if err := Save(path, input); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != CurrentVersion || len(got.Packages) != 1 || got.Packages[0].Name != input[0].Name {
		t.Fatalf("unexpected lockfile: %+v", got)
	}

	// 기존 파일을 덮어쓰는 업데이트 경로도 모든 OS에서 동작해야 합니다.
	updated := []*pkg.Package{{Name: "owner/new-tool", Version: "v2.0.0", Source: "https://example.test/new-tool"}}
	if err := Save(path, updated); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != 1 || got.Packages[0].Name != updated[0].Name {
		t.Fatalf("unexpected updated lockfile: %+v", got)
	}
}

func TestLoadRejectsUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ppm.lock")
	if err := Save(path, []*pkg.Package{{Name: "owner/tool", Version: "v1.0.0", Source: "https://example.test/tool"}}); err != nil {
		t.Fatal(err)
	}
	data := []byte("version: 99\npackages: []\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unsupported version error")
	}
}
