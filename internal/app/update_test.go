package app

import (
	"io"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/pkg"
)

type updateFetcher struct {
	packages map[string]*pkg.Package
}

func (f updateFetcher) GetMetadata(name string) (*pkg.Package, error) {
	return f.packages[name], nil
}

func (updateFetcher) DownloadSource(*pkg.Package) (io.ReadCloser, int64, error) {
	return io.NopCloser(strings.NewReader("archive")), int64(len("archive")), nil
}

func TestPackageUpdaterUpdatesDependenciesAndReportsResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("APPDATA", "")

	fetcher := updateFetcher{packages: map[string]*pkg.Package{
		"owner/app": {Name: "owner/app", Version: "v2.0.0", Source: "app.tar.gz", Dependencies: []string{"owner/dep"}},
		"owner/dep": {Name: "owner/dep", Version: "v1.0.0", Source: "dep.tar.gz"},
	}}
	var installed []string
	service := PackageUpdater{
		Fetcher:     fetcher,
		InstallPath: t.TempDir(),
		NewArchiver: func(_, binName string) pkg.Archiver {
			installed = append(installed, binName)
			return &installerArchiver{}
		},
	}

	result, err := service.Update([]*pkg.Package{
		{Name: "owner/app", Version: "v1.0.0"},
	}, []string{"owner/app"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if result.Updated != 2 || result.Skipped != 0 || len(installed) != 2 {
		t.Fatalf("result = %+v, installed = %v, want two updates", result, installed)
	}
}

func TestPackageUpdaterSkipsCurrentVersionAndCountsLegacyPackages(t *testing.T) {
	fetcher := updateFetcher{packages: map[string]*pkg.Package{
		"owner/app": {Name: "owner/app", Version: "v1.0.0", Source: "app.tar.gz"},
	}}
	service := PackageUpdater{
		Fetcher:     fetcher,
		InstallPath: t.TempDir(),
		NewArchiver: func(_, _ string) pkg.Archiver { return &installerArchiver{} },
	}

	result, err := service.Update([]*pkg.Package{
		{Name: "owner/app", Version: "v1.0.0"},
		{Name: "legacy", Version: ""},
	}, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if result.Updated != 0 || result.Skipped != 1 || result.Legacy != 1 {
		t.Fatalf("result = %+v, want skipped=1 and legacy=1", result)
	}
}
