package app

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/pkg"
)

type installerFetcher struct{}

func (installerFetcher) GetMetadata(string) (*pkg.Package, error) { return nil, nil }

func (installerFetcher) DownloadSource(*pkg.Package) (io.ReadCloser, int64, error) {
	return io.NopCloser(strings.NewReader("archive")), int64(len("archive")), nil
}

type installerArchiver struct {
	extracted bool
	linked    bool
}

func (a *installerArchiver) Extract(_ io.Reader, dest string) error {
	a.extracted = true
	return os.MkdirAll(dest, 0755)
}

func (a *installerArchiver) Link(_, _, _ string) error {
	a.linked = true
	return nil
}

func TestPackageInstallerRequiresDependencies(t *testing.T) {
	service := PackageInstaller{}
	if err := service.Install(nil); err == nil {
		t.Fatal("expected missing fetcher error")
	}

	service.Fetcher = installerFetcher{}
	if err := service.Install(nil); err == nil {
		t.Fatal("expected missing archiver factory error")
	}
}

// isolateTestHome은 테스트별 임시 홈을 사용하도록 모든 경로 환경 변수를 격리합니다.
// platform.GetPaths()는 HOME보다 APPDATA·LOCALAPPDATA·PPM_CONFIG_DIR·XDG_CONFIG_HOME
// 같은 명시적 경로를 우선하므로 이를 비우지 않으면, 해당 변수가 설정된 CI 러너에서
// 모든 테스트가 실제 홈·캐시 디렉터리를 공유해 실행 순서에 따라 서로 간섭합니다.
func isolateTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{
		"APPDATA",
		"LOCALAPPDATA",
		"PPM_CONFIG_DIR",
		"PPM_INSTALL_DIR",
		"PPM_CACHE_DIR",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
	} {
		t.Setenv(name, "")
	}
	return home
}

func TestPackageInstallerCreatesArchiverAndInstallsInOrder(t *testing.T) {
	isolateTestHome(t)

	var names []string
	archivers := make(map[string]*installerArchiver)
	service := PackageInstaller{
		Fetcher:     installerFetcher{},
		InstallPath: t.TempDir(),
		NewArchiver: func(source, binName string) pkg.Archiver {
			names = append(names, source+":"+binName)
			archiver := &installerArchiver{}
			archivers[binName] = archiver
			return archiver
		},
	}

	packages := []*pkg.Package{
		{Name: "owner/first", Version: "v1.0.0", Source: "first.tar.gz"},
		{Name: "owner/second", Version: "v1.0.0", Source: "second.tar.gz", BinName: "second-bin"},
	}
	if err := service.Install(packages); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	wantNames := []string{"first.tar.gz:first", "second.tar.gz:second-bin"}
	if len(names) != len(wantNames) || names[0] != wantNames[0] || names[1] != wantNames[1] {
		t.Fatalf("archiver factory calls = %v, want %v", names, wantNames)
	}
	for name, archiver := range archivers {
		if !archiver.extracted || !archiver.linked {
			t.Errorf("archiver for %s was not used for extraction and linking", name)
		}
	}
}
