// Package app contains application services that orchestrate domain operations
// without depending on the CLI framework.
package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"ppm/internal/pkg"
)

// ArchiverFactory creates the archive implementation appropriate for a source.
type ArchiverFactory func(source, binName string) pkg.Archiver

// PackageInstaller coordinates installation of already-resolved packages.
// Metadata resolution remains separate so both stages can be tested independently.
type PackageInstaller struct {
	Fetcher     pkg.RegistryFetcher
	InstallPath string
	NewArchiver ArchiverFactory
}

// Install installs packages in the supplied order and reports all failures.
// Keeping the order deterministic is important because dependencies are resolved
// before this service is called.
func (s PackageInstaller) Install(packages []*pkg.Package) error {
	if s.Fetcher == nil {
		return errors.New("package installer requires a fetcher")
	}
	if s.NewArchiver == nil {
		return errors.New("package installer requires an archiver factory")
	}

	var failures []error
	for _, p := range packages {
		if p == nil {
			failures = append(failures, errors.New("cannot install nil package"))
			continue
		}

		binName := filepath.Base(p.Name)
		if p.BinName != "" {
			binName = p.BinName
		}
		archiver := s.NewArchiver(p.Source, binName)
		if archiver == nil {
			failures = append(failures, fmt.Errorf("%s: archiver factory returned nil", p.Name))
			continue
		}
		if err := pkg.InstallWithPackage(p, s.Fetcher, archiver, s.InstallPath); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", p.Name, err))
		}
	}

	return errors.Join(failures...)
}
