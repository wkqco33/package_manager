package app

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPackageUninstallerRequiresRemoveOperation(t *testing.T) {
	if err := (PackageUninstaller{}).Uninstall([]string{"owner/repo"}); err == nil {
		t.Fatal("expected missing remove operation error")
	}
}

func TestPackageUninstallerRemovesAllPackagesWithBoundedConcurrency(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	var active, maxActive atomic.Int32
	service := PackageUninstaller{
		InstallPath: "/tmp/bin",
		Concurrency: 2,
		Remove: func(name, path string) error {
			if path != "/tmp/bin" {
				t.Errorf("install path = %q, want /tmp/bin", path)
			}
			mu.Lock()
			calls = append(calls, name)
			mu.Unlock()
			current := active.Add(1)
			for {
				previous := maxActive.Load()
				if current <= previous || maxActive.CompareAndSwap(previous, current) {
					break
				}
			}
			active.Add(-1)
			return nil
		},
	}

	if err := service.Uninstall([]string{"one", "two", "three"}); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if len(calls) != 3 {
		t.Fatalf("remove calls = %v, want all packages", calls)
	}
	if got := maxActive.Load(); got > 2 {
		t.Fatalf("max concurrent removals = %d, want at most 2", got)
	}
}

func TestPackageUninstallerAggregatesFailures(t *testing.T) {
	wantErr := errors.New("permission denied")
	service := PackageUninstaller{
		Remove: func(name, _ string) error {
			if name == "broken" {
				return wantErr
			}
			return nil
		},
	}

	err := service.Uninstall([]string{"ok", "broken"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped remove error", err)
	}
}
