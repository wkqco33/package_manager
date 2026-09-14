package app

import (
	"errors"
	"strings"
	"sync"

	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/version"
)

const defaultUpdateConcurrency = 5

// CheckResult describes the update status of an installed package.
type CheckResult struct {
	Package   *pkg.Package
	Latest    *pkg.Package
	HasUpdate bool
}

// UpdateResult describes the work performed by PackageUpdater.
type UpdateResult struct {
	Updated int
	Skipped int
	Legacy  int
}

// PackageUpdater resolves requested packages and delegates installation to the
// same installer service used by the install command.
type PackageUpdater struct {
	Fetcher         pkg.RegistryFetcher
	MetadataFetcher pkg.MetadataFetcher
	InstallPath     string
	NewArchiver     ArchiverFactory
	Concurrency     int
}

// Check checks for available updates for the installed packages concurrently.
func (s PackageUpdater) Check(installed []*pkg.Package) ([]CheckResult, error) {
	fetcher := s.MetadataFetcher
	if fetcher == nil {
		fetcher = s.Fetcher
	}
	if fetcher == nil {
		return nil, errors.New("package updater requires a fetcher")
	}

	concurrency := s.Concurrency
	if concurrency <= 0 {
		concurrency = defaultUpdateConcurrency
	}

	results := make([]CheckResult, len(installed))
	type checkTask struct {
		index int
		pkg   *pkg.Package
	}

	var validTasks []checkTask
	for i, p := range installed {
		if p == nil {
			continue
		}
		if !strings.Contains(p.Name, "/") {
			results[i] = CheckResult{Package: p, Latest: nil, HasUpdate: false}
			continue
		}
		validTasks = append(validTasks, checkTask{index: i, pkg: p})
	}

	if len(validTasks) == 0 {
		return results, nil
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var errs []error

	for _, task := range validTasks {
		wg.Add(1)
		sem <- struct{}{}

		go func(t checkTask) {
			defer wg.Done()
			defer func() { <-sem }()

			latest, err := fetcher.GetMetadata(t.pkg.Name)
			if err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
				return
			}

			hasUpdate := latest != nil && version.Compare(t.pkg.Version, latest.Version) < 0
			results[t.index] = CheckResult{
				Package:   t.pkg,
				Latest:    latest,
				HasUpdate: hasUpdate,
			}
		}(task)
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return results, nil
}

type prefetchCache struct {
	underlying pkg.MetadataFetcher
	mu         sync.RWMutex
	cache      map[string]*pkg.Package
}

func (c *prefetchCache) GetMetadata(name string) (*pkg.Package, error) {
	c.mu.RLock()
	p, ok := c.cache[name]
	c.mu.RUnlock()
	if ok {
		return p, nil
	}
	p, err := c.underlying.GetMetadata(name)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.cache[name] = p
	c.mu.Unlock()
	return p, nil
}

func prefetchMetadata(targets []string, fetcher pkg.MetadataFetcher, concurrency int) pkg.MetadataFetcher {
	if len(targets) <= 1 {
		return fetcher
	}
	if concurrency <= 0 {
		concurrency = defaultUpdateConcurrency
	}

	cache := &prefetchCache{
		underlying: fetcher,
		cache:      make(map[string]*pkg.Package, len(targets)),
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		sem <- struct{}{}

		go func(name string) {
			defer wg.Done()
			defer func() { <-sem }()

			p, err := fetcher.GetMetadata(name)
			if err == nil && p != nil {
				cache.mu.Lock()
				cache.cache[name] = p
				cache.mu.Unlock()
			}
		}(target)
	}

	wg.Wait()
	return cache
}

// Update updates requested packages. When requested is empty, all modern
// owner/repository packages in installed are selected.
func (s PackageUpdater) Update(installed []*pkg.Package, requested []string) (UpdateResult, error) {
	if s.Fetcher == nil {
		return UpdateResult{}, errors.New("package updater requires a fetcher")
	}

	installedVersions := make(map[string]map[string]struct{})
	for _, p := range installed {
		if p == nil || p.Name == "" || p.Version == "" {
			continue
		}
		if installedVersions[p.Name] == nil {
			installedVersions[p.Name] = make(map[string]struct{})
		}
		installedVersions[p.Name][p.Version] = struct{}{}
	}

	targets := append([]string(nil), requested...)
	legacy := 0
	if len(targets) == 0 {
		seen := make(map[string]struct{})
		for _, p := range installed {
			if p == nil || !strings.Contains(p.Name, "/") {
				legacy++
				continue
			}
			if _, exists := seen[p.Name]; exists {
				continue
			}
			seen[p.Name] = struct{}{}
			targets = append(targets, p.Name)
		}
	}
	if len(targets) == 0 {
		return UpdateResult{Legacy: legacy}, nil
	}

	metadataFetcher := pkg.MetadataFetcher(s.Fetcher)
	if s.MetadataFetcher != nil {
		metadataFetcher = s.MetadataFetcher
	}
	metadataFetcher = prefetchMetadata(targets, metadataFetcher, s.Concurrency)
	resolved, err := pkg.ResolveDependencies(targets, metadataFetcher)
	if err != nil {
		return UpdateResult{}, err
	}

	toInstall := make([]*pkg.Package, 0, len(resolved))
	result := UpdateResult{Legacy: legacy}
	for _, latest := range resolved {
		if versions := installedVersions[latest.Name]; versions != nil {
			if _, exists := versions[latest.Version]; exists {
				result.Skipped++
				continue
			}
		}
		toInstall = append(toInstall, latest)
	}

	if len(toInstall) == 0 {
		return result, nil
	}
	installer := PackageInstaller{
		Fetcher:     s.Fetcher,
		InstallPath: s.InstallPath,
		NewArchiver: s.NewArchiver,
	}
	if err := installer.Install(toInstall); err != nil {
		return result, err
	}
	result.Updated = len(toInstall)
	return result, nil
}
