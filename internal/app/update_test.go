package app

import (
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	isolateTestHome(t)

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

type concurrentMockFetcher struct {
	onGetMetadata func(string) (*pkg.Package, error)
}

func (f concurrentMockFetcher) GetMetadata(name string) (*pkg.Package, error) {
	if f.onGetMetadata != nil {
		return f.onGetMetadata(name)
	}
	return nil, nil
}

func (concurrentMockFetcher) DownloadSource(*pkg.Package) (io.ReadCloser, int64, error) {
	return io.NopCloser(strings.NewReader("archive")), int64(len("archive")), nil
}

func TestPackageUpdaterCheckConcurrently(t *testing.T) {
	var (
		currentConcurrency     int32
		maxConcurrencyObserved int32
	)

	fetcher := concurrentMockFetcher{
		onGetMetadata: func(name string) (*pkg.Package, error) {
			cur := atomic.AddInt32(&currentConcurrency, 1)
			for {
				max := atomic.LoadInt32(&maxConcurrencyObserved)
				if cur <= max || atomic.CompareAndSwapInt32(&maxConcurrencyObserved, max, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&currentConcurrency, -1)

			switch name {
			case "owner/pkg1":
				return &pkg.Package{Name: "owner/pkg1", Version: "v2.0.0"}, nil
			case "owner/pkg2":
				return &pkg.Package{Name: "owner/pkg2", Version: "v1.0.0"}, nil
			case "owner/pkg3":
				return &pkg.Package{Name: "owner/pkg3", Version: "v1.5.0"}, nil
			default:
				return nil, errors.New("unknown package")
			}
		},
	}

	service := PackageUpdater{
		Fetcher:     fetcher,
		Concurrency: 2,
	}

	installed := []*pkg.Package{
		{Name: "owner/pkg1", Version: "v1.0.0"},
		{Name: "owner/pkg2", Version: "v1.0.0"},
		{Name: "owner/pkg3", Version: "v1.0.0"},
		{Name: "legacy-pkg", Version: "v1.0.0"},
	}

	results, err := service.Check(installed)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("len(results) = %d, want 4", len(results))
	}

	if results[0].Package.Name != "owner/pkg1" || !results[0].HasUpdate || results[0].Latest.Version != "v2.0.0" {
		t.Errorf("results[0] unexpected: %+v", results[0])
	}
	if results[1].Package.Name != "owner/pkg2" || results[1].HasUpdate {
		t.Errorf("results[1] unexpected: %+v", results[1])
	}
	if results[2].Package.Name != "owner/pkg3" || !results[2].HasUpdate || results[2].Latest.Version != "v1.5.0" {
		t.Errorf("results[2] unexpected: %+v", results[2])
	}
	if results[3].Package.Name != "legacy-pkg" || results[3].HasUpdate || results[3].Latest != nil {
		t.Errorf("results[3] unexpected: %+v", results[3])
	}

	if maxConcurrencyObserved > 2 {
		t.Errorf("maxConcurrencyObserved = %d, want <= 2", maxConcurrencyObserved)
	}
	if maxConcurrencyObserved < 2 {
		t.Errorf("expected concurrency to be at least 2, got %d", maxConcurrencyObserved)
	}
}

func TestPackageUpdaterCheckPropagatesError(t *testing.T) {
	wantErr := errors.New("metadata network error")
	fetcher := concurrentMockFetcher{
		onGetMetadata: func(name string) (*pkg.Package, error) {
			if name == "owner/err" {
				return nil, wantErr
			}
			return &pkg.Package{Name: name, Version: "v1.0.0"}, nil
		},
	}

	service := PackageUpdater{
		Fetcher:     fetcher,
		Concurrency: 2,
	}

	_, err := service.Check([]*pkg.Package{
		{Name: "owner/ok", Version: "v1.0.0"},
		{Name: "owner/err", Version: "v1.0.0"},
	})
	if err == nil || !strings.Contains(err.Error(), "metadata network error") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestPackageUpdaterUpdatePrefetchesConcurrently(t *testing.T) {
	isolateTestHome(t)

	var (
		currentConcurrency     int32
		maxConcurrencyObserved int32
	)

	fetcher := concurrentMockFetcher{
		onGetMetadata: func(name string) (*pkg.Package, error) {
			cur := atomic.AddInt32(&currentConcurrency, 1)
			for {
				max := atomic.LoadInt32(&maxConcurrencyObserved)
				if cur <= max || atomic.CompareAndSwapInt32(&maxConcurrencyObserved, max, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&currentConcurrency, -1)

			return &pkg.Package{Name: name, Version: "v2.0.0", Source: name + ".tar.gz"}, nil
		},
	}

	var installed []string
	service := PackageUpdater{
		Fetcher:     fetcher,
		InstallPath: t.TempDir(),
		Concurrency: 2,
		NewArchiver: func(_, binName string) pkg.Archiver {
			installed = append(installed, binName)
			return &installerArchiver{}
		},
	}

	result, err := service.Update([]*pkg.Package{
		{Name: "owner/app1", Version: "v1.0.0"},
		{Name: "owner/app2", Version: "v1.0.0"},
	}, []string{"owner/app1", "owner/app2"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if result.Updated != 2 {
		t.Fatalf("result.Updated = %d, want 2", result.Updated)
	}
	if maxConcurrencyObserved < 2 {
		t.Errorf("expected concurrent prefetching, got max concurrency %d", maxConcurrencyObserved)
	}
}
