package app

import (
	"errors"
	"fmt"
	"sync"
)

// RemovePackage is the filesystem operation used by PackageUninstaller.
// It is injectable so orchestration tests do not need to create real packages.
type RemovePackage func(packageName, installPath string) error

// PackageUninstaller coordinates bounded-concurrency package removal.
type PackageUninstaller struct {
	InstallPath string
	Remove      RemovePackage
	Concurrency int
}

// Uninstall removes all requested packages and returns every failure.
func (s PackageUninstaller) Uninstall(packageNames []string) error {
	if s.Remove == nil {
		return errors.New("package uninstaller requires a remove operation")
	}
	limit := s.Concurrency
	if limit <= 0 {
		limit = 1
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, limit)
	errCh := make(chan error, len(packageNames))
	for _, name := range packageNames {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := s.Remove(name, s.InstallPath); err != nil {
				errCh <- fmt.Errorf("%s: %w", name, err)
			}
		}()
	}

	wg.Wait()
	close(errCh)
	var failures []error
	for err := range errCh {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}
