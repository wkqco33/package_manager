package platform

import (
	"os"
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
	for _, name := range []string{"PPM_CONFIG_DIR", "PPM_INSTALL_DIR", "PPM_CACHE_DIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(name, "")
	}
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

func TestGetPathsHonorsPPMOverrides(t *testing.T) {
	t.Setenv("PPM_CONFIG_DIR", filepath.Join(t.TempDir(), "config"))
	t.Setenv("PPM_INSTALL_DIR", filepath.Join(t.TempDir(), "bin"))
	t.Setenv("PPM_CACHE_DIR", filepath.Join(t.TempDir(), "cache"))

	paths, err := GetPaths()
	if err != nil {
		t.Fatalf("GetPaths failed: %v", err)
	}
	if paths.ConfigDir != os.Getenv("PPM_CONFIG_DIR") {
		t.Errorf("ConfigDir = %q, want PPM_CONFIG_DIR", paths.ConfigDir)
	}
	if paths.BinDir != os.Getenv("PPM_INSTALL_DIR") {
		t.Errorf("BinDir = %q, want PPM_INSTALL_DIR", paths.BinDir)
	}
	if paths.CacheDir != os.Getenv("PPM_CACHE_DIR") {
		t.Errorf("CacheDir = %q, want PPM_CACHE_DIR", paths.CacheDir)
	}
}

func TestGetPathsHonorsXDGOverridesOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG 경로는 Unix 계열에서만 적용됩니다")
	}
	t.Setenv("PPM_CONFIG_DIR", "")
	t.Setenv("PPM_CACHE_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "xdg-cache"))

	paths, err := GetPaths()
	if err != nil {
		t.Fatalf("GetPaths failed: %v", err)
	}
	if want := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "ppm"); paths.ConfigDir != want {
		t.Errorf("ConfigDir = %q, want %q", paths.ConfigDir, want)
	}
	if want := filepath.Join(os.Getenv("XDG_CACHE_HOME"), "ppm"); paths.CacheDir != want {
		t.Errorf("CacheDir = %q, want %q", paths.CacheDir, want)
	}
}
