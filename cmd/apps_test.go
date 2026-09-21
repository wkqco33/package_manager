package cmd

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/apps"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
)

// isolateTestHome은 테스트별 임시 홈을 사용하도록 모든 경로 환경 변수를 격리합니다.
// platform.GetPaths()는 HOME보다 PPM_CONFIG_DIR·XDG_CONFIG_HOME 같은 명시적 재정의를
// 우선하므로 이를 비우지 않으면, 해당 변수가 설정된 CI 러너에서 모든 테스트가 실제
// 홈·캐시 디렉터리를 공유해 실행 순서에 따라 서로 간섭합니다.
func isolateTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", "")
	for _, name := range []string{
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

func TestAppsCommandInjectsPackageDirectoryLookup(t *testing.T) {
	wantErr := errors.New("paths unavailable")
	command := newAppsCommand(appsDependencies{
		GetPackagesDir: func() (string, error) { return "", wantErr },
	})
	if err := command.Execute(nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected path error", err)
	}
}

func TestAppsCommandMarksInstalledApps(t *testing.T) {
	command := newAppsCommand(appsDependencies{
		GetPackagesDir: func() (string, error) { return "/tmp/pkgs", nil },
		ListInstalled: func(string) ([]*pkg.Package, error) {
			return []*pkg.Package{{Name: "cli/cli", Version: "v2.0.0"}}, nil
		},
		DefaultApps: func() []apps.DefaultApp {
			return []apps.DefaultApp{
				{Name: "cli/cli", BinName: "gh", Description: "GitHub CLI", Homepage: "https://cli.github.com"},
				{Name: "jqlang/jq", BinName: "jq", Description: "JSON processor", Homepage: "https://jqlang.github.io/jq"},
			}
		},
	})
	if err := command.Execute(nil); err != nil {
		t.Fatalf("apps command error = %v", err)
	}
}

type mockAppsCmdFetcher struct {
	downloaded []string
}

func (m *mockAppsCmdFetcher) GetMetadata(name string) (*pkg.Package, error) {
	return &pkg.Package{Name: name, Version: "v1.0.0", Source: name + ".tar.gz"}, nil
}

func (m *mockAppsCmdFetcher) DownloadSource(p *pkg.Package) (io.ReadCloser, int64, error) {
	m.downloaded = append(m.downloaded, p.Name)
	return io.NopCloser(strings.NewReader("content")), int64(len("content")), nil
}

type mockAppsCmdArchiver struct{}

func (a *mockAppsCmdArchiver) Extract(_ io.Reader, dest string) error {
	return os.MkdirAll(dest, 0755)
}

func (a *mockAppsCmdArchiver) Link(_, _, _ string) error {
	return nil
}

func TestAppsCommandInstallFlag_FiltersAndInstallsUninstalledApps(t *testing.T) {
	isolateTestHome(t)

	var installedTargets []string
	command := newAppsCommand(appsDependencies{
		GetPackagesDir: func() (string, error) { return "/tmp/pkgs", nil },
		ListInstalled: func(string) ([]*pkg.Package, error) {
			return []*pkg.Package{{Name: "cli/cli", Version: "v1.0.0"}}, nil
		},
		DefaultApps: func() []apps.DefaultApp {
			return []apps.DefaultApp{
				{Name: "cli/cli", BinName: "gh"},
				{Name: "jqlang/jq", BinName: "jq"},
			}
		},
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{InstallPath: t.TempDir()}, nil
		},
		NewFetcher: func(*config.Config) pkg.RegistryFetcher {
			return &mockAppsCmdFetcher{}
		},
		NewArchiver: func(source, binName string) pkg.Archiver {
			installedTargets = append(installedTargets, binName)
			return &mockAppsCmdArchiver{}
		},
	})

	if err := command.Execute([]string{"--install"}); err != nil {
		t.Fatalf("command error = %v", err)
	}

	// cli/cli는 이미 설치되어 있으므로 jqlang/jq만 설치되어야 함
	if len(installedTargets) != 1 || installedTargets[0] != "jq" {
		t.Fatalf("expected installed targets [jq], got %v", installedTargets)
	}
}

func TestAppsCommandInstallAllFlag(t *testing.T) {
	isolateTestHome(t)

	var installedTargets []string
	command := newAppsCommand(appsDependencies{
		GetPackagesDir: func() (string, error) { return "/tmp/pkgs", nil },
		ListInstalled: func(string) ([]*pkg.Package, error) {
			return []*pkg.Package{{Name: "cli/cli", Version: "v1.0.0"}}, nil
		},
		DefaultApps: func() []apps.DefaultApp {
			return []apps.DefaultApp{
				{Name: "cli/cli", BinName: "gh"},
				{Name: "jqlang/jq", BinName: "jq"},
			}
		},
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{InstallPath: t.TempDir()}, nil
		},
		NewFetcher: func(*config.Config) pkg.RegistryFetcher {
			return &mockAppsCmdFetcher{}
		},
		NewArchiver: func(source, binName string) pkg.Archiver {
			installedTargets = append(installedTargets, binName)
			return &mockAppsCmdArchiver{}
		},
	})

	if err := command.Execute([]string{"--install", "--all"}); err != nil {
		t.Fatalf("command error = %v", err)
	}

	// --all 이므로 둘 다 설치되어야 함
	if len(installedTargets) != 2 {
		t.Fatalf("expected 2 installed targets, got %d (%v)", len(installedTargets), installedTargets)
	}
}

func TestAppsCommandInstallSpecificApps(t *testing.T) {
	isolateTestHome(t)

	var installedTargets []string
	command := newAppsCommand(appsDependencies{
		GetPackagesDir: func() (string, error) { return "/tmp/pkgs", nil },
		ListInstalled: func(string) ([]*pkg.Package, error) {
			return nil, nil
		},
		DefaultApps: func() []apps.DefaultApp {
			return []apps.DefaultApp{
				{Name: "cli/cli", BinName: "gh"},
				{Name: "jqlang/jq", BinName: "jq"},
			}
		},
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{InstallPath: t.TempDir()}, nil
		},
		NewFetcher: func(*config.Config) pkg.RegistryFetcher {
			return &mockAppsCmdFetcher{}
		},
		NewArchiver: func(source, binName string) pkg.Archiver {
			installedTargets = append(installedTargets, binName)
			return &mockAppsCmdArchiver{}
		},
	})

	// 인자로 jq만 전달
	if err := command.Execute([]string{"--install", "jq"}); err != nil {
		t.Fatalf("command error = %v", err)
	}

	if len(installedTargets) != 1 || installedTargets[0] != "jq" {
		t.Fatalf("expected [jq], got %v", installedTargets)
	}
}

func TestAppsCommandInstallDryRun(t *testing.T) {
	isolateTestHome(t)

	archiverCalled := false
	command := newAppsCommand(appsDependencies{
		GetPackagesDir: func() (string, error) { return "/tmp/pkgs", nil },
		ListInstalled: func(string) ([]*pkg.Package, error) {
			return nil, nil
		},
		DefaultApps: func() []apps.DefaultApp {
			return []apps.DefaultApp{
				{Name: "cli/cli", BinName: "gh"},
			}
		},
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{InstallPath: t.TempDir()}, nil
		},
		NewFetcher: func(*config.Config) pkg.RegistryFetcher {
			return &mockAppsCmdFetcher{}
		},
		NewArchiver: func(source, binName string) pkg.Archiver {
			archiverCalled = true
			return &mockAppsCmdArchiver{}
		},
	})

	if err := command.Execute([]string{"--install", "--dry-run"}); err != nil {
		t.Fatalf("command error = %v", err)
	}

	if archiverCalled {
		t.Fatal("archiver should not be called in dry run")
	}
}
