package platform

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecutableName(t *testing.T) {
	got := ExecutableName("ppm")
	if runtime.GOOS == "windows" {
		if got != "ppm.exe" {
			t.Errorf("Windows에서는 ppm.exe를 기대했으나 %s", got)
		}
	} else {
		if got != "ppm" {
			t.Errorf("Unix에서는 ppm을 기대했으나 %s", got)
		}
	}
}

func TestGetPaths(t *testing.T) {
	paths, err := GetPaths()
	if err != nil {
		t.Fatalf("GetPaths failed: %v", err)
	}

	if paths.ConfigDir == "" || paths.BinDir == "" || paths.PackageDir == "" {
		t.Fatalf("경로가 비어 있습니다: %+v", paths)
	}

	// PackageDir는 항상 ConfigDir 하위의 packages 디렉터리여야 함
	want := filepath.Join(paths.ConfigDir, "packages")
	if paths.PackageDir != want {
		t.Errorf("PackageDir 기대값 %s, 실제 %s", want, paths.PackageDir)
	}

	// BinDir는 모든 플랫폼에서 .local/bin 으로 끝남
	if !strings.HasSuffix(paths.BinDir, filepath.Join(".local", "bin")) {
		t.Errorf("BinDir가 .local/bin으로 끝나야 하나 %s", paths.BinDir)
	}
}
