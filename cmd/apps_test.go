package cmd

import (
	"errors"
	"testing"

	"github.com/wkqco33/package_manager/internal/apps"
	"github.com/wkqco33/package_manager/internal/pkg"
)

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
