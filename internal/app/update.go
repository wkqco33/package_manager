package app

import (
	"errors"
	"strings"

	"ppm/internal/pkg"
)

// UpdateResult describes the work performed by PackageUpdater.
type UpdateResult struct {
	Updated int
	Skipped int
	Legacy  int
}

// PackageUpdater resolves requested packages and delegates installation to the
// same installer service used by the install command.
type PackageUpdater struct {
	Fetcher     pkg.RegistryFetcher
	InstallPath string
	NewArchiver ArchiverFactory
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

	resolved, err := pkg.ResolveDependencies(targets, s.Fetcher)
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
