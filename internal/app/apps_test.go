package app

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wkqco33/package_manager/internal/apps"
	"github.com/wkqco33/package_manager/internal/pkg"
)

type mockAppsFetcher struct {
	mu           sync.Mutex
	activeCount  int32
	maxActive    int32
	metadataMap  map[string]*pkg.Package
	failMetadata map[string]bool
	delay        time.Duration
}

func (m *mockAppsFetcher) GetMetadata(pkgName string) (*pkg.Package, error) {
	curr := atomic.AddInt32(&m.activeCount, 1)
	m.mu.Lock()
	if curr > m.maxActive {
		m.maxActive = curr
	}
	m.mu.Unlock()
	defer atomic.AddInt32(&m.activeCount, -1)

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failMetadata[pkgName] {
		return nil, fmt.Errorf("failed to fetch metadata for %s", pkgName)
	}
	if p, ok := m.metadataMap[pkgName]; ok {
		return p, nil
	}
	return &pkg.Package{
		Name:    pkgName,
		Version: "v1.0.0",
		Source:  "https://example.com/" + pkgName + ".tar.gz",
	}, nil
}

func (m *mockAppsFetcher) DownloadSource(p *pkg.Package) (io.ReadCloser, int64, error) {
	return io.NopCloser(strings.NewReader("archive-content")), int64(len("archive-content")), nil
}

func TestAppsInstallerRequiresDependencies(t *testing.T) {
	installer := AppsInstaller{}
	_, err := installer.InstallApps(nil, false)
	if err == nil || !strings.Contains(err.Error(), "fetcher") {
		t.Fatalf("expected missing fetcher error, got %v", err)
	}

	installer.Fetcher = &mockAppsFetcher{}
	_, err = installer.InstallApps(nil, false)
	if err == nil || !strings.Contains(err.Error(), "archiver") {
		t.Fatalf("expected missing archiver factory error, got %v", err)
	}
}

func TestAppsInstallerEmptyTargets(t *testing.T) {
	installer := AppsInstaller{
		Fetcher:     &mockAppsFetcher{},
		NewArchiver: func(source, binName string) pkg.Archiver { return &installerArchiver{} },
	}
	result, err := installer.InstallApps([]apps.DefaultApp{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Installed) != 0 {
		t.Fatalf("expected 0 installed, got %d", len(result.Installed))
	}
}

func TestAppsInstallerDryRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	fetcher := &mockAppsFetcher{
		metadataMap: map[string]*pkg.Package{
			"user/app1": {Name: "user/app1", Version: "v1.0.0", Source: "app1.tar.gz"},
			"user/app2": {Name: "user/app2", Version: "v2.0.0", Source: "app2.tar.gz"},
		},
	}
	archiverCalls := 0
	installer := AppsInstaller{
		Fetcher:     fetcher,
		InstallPath: t.TempDir(),
		NewArchiver: func(source, binName string) pkg.Archiver {
			archiverCalls++
			return &installerArchiver{}
		},
		Concurrency: 2,
	}

	targets := []apps.DefaultApp{
		{Name: "user/app1", BinName: "app1-bin"},
		{Name: "user/app2", BinName: "app2-bin"},
	}

	result, err := installer.InstallApps(targets, true)
	if err != nil {
		t.Fatalf("unexpected error on dry run: %v", err)
	}
	if archiverCalls != 0 {
		t.Fatalf("expected 0 archiver calls on dry run, got %d", archiverCalls)
	}
	if len(result.Installed) != 2 {
		t.Fatalf("expected 2 planned installs, got %d", len(result.Installed))
	}
}

func TestAppsInstallerConcurrentExecution(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	fetcher := &mockAppsFetcher{
		delay: 10 * time.Millisecond,
	}
	var linkedBins []string
	var mu sync.Mutex
	installer := AppsInstaller{
		Fetcher:     fetcher,
		InstallPath: t.TempDir(),
		NewArchiver: func(source, binName string) pkg.Archiver {
			return &mockTrackingArchiver{
				onLink: func(bin string) {
					mu.Lock()
					linkedBins = append(linkedBins, bin)
					mu.Unlock()
				},
			}
		},
		Concurrency: 3,
	}

	targets := []apps.DefaultApp{
		{Name: "user/app1", BinName: "b1"},
		{Name: "user/app2", BinName: "b2"},
		{Name: "user/app3", BinName: "b3"},
		{Name: "user/app4", BinName: "b4"},
		{Name: "user/app5", BinName: "b5"},
	}

	result, err := installer.InstallApps(targets, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Installed) != 5 {
		t.Fatalf("expected 5 installed, got %d", len(result.Installed))
	}
	if fetcher.maxActive > 3 {
		t.Fatalf("concurrency limit exceeded: maxActive=%d > Concurrency=3", fetcher.maxActive)
	}
	if len(linkedBins) != 5 {
		t.Fatalf("expected 5 linked binaries, got %d", len(linkedBins))
	}
}

func TestAppsInstallerPartialFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	fetcher := &mockAppsFetcher{
		failMetadata: map[string]bool{
			"user/app2": true,
		},
	}
	installer := AppsInstaller{
		Fetcher:     fetcher,
		InstallPath: t.TempDir(),
		NewArchiver: func(source, binName string) pkg.Archiver {
			return &installerArchiver{}
		},
		Concurrency: 2,
	}

	targets := []apps.DefaultApp{
		{Name: "user/app1", BinName: "b1"},
		{Name: "user/app2", BinName: "b2"},
		{Name: "user/app3", BinName: "b3"},
	}

	result, err := installer.InstallApps(targets, false)
	if err == nil {
		t.Fatal("expected error for failed package, got nil")
	}
	if len(result.Installed) != 2 {
		t.Fatalf("expected 2 successful installs, got %d", len(result.Installed))
	}
	if len(result.Failed) != 1 || result.Failed["user/app2"] == nil {
		t.Fatalf("expected user/app2 in failed list, got %v", result.Failed)
	}
}

type mockTrackingArchiver struct {
	onLink func(binName string)
}

func (m *mockTrackingArchiver) Extract(_ io.Reader, dest string) error {
	return os.MkdirAll(dest, 0755)
}

func (m *mockTrackingArchiver) Link(extractedDir, binName, targetLink string) error {
	if m.onLink != nil {
		m.onLink(binName)
	}
	return nil
}
