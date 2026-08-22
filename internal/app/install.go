// Package app contains application services that orchestrate domain operations
// without depending on the CLI framework.
package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/platform"
)

// ArchiverFactory creates the archive implementation appropriate for a source.
type ArchiverFactory func(source, binName string) pkg.Archiver

// PackageInstaller coordinates installation of already-resolved packages.
// Metadata resolution remains separate so both stages can be tested independently.
type PackageInstaller struct {
	Fetcher          pkg.RegistryFetcher
	InstallPath      string
	NewArchiver      ArchiverFactory
	AllowSourceBuild bool
	Offline          bool
	Atomic           bool
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

	var rollback *installRollback
	if s.Atomic {
		var err error
		rollback, err = newInstallRollback(packages, s.InstallPath)
		if err != nil {
			return err
		}
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
		if err := pkg.InstallWithPackageOptions(p, s.Fetcher, archiver, s.InstallPath, pkg.InstallOptions{AllowSourceBuild: s.AllowSourceBuild, Offline: s.Offline}); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", p.Name, err))
		}
	}

	if len(failures) > 0 {
		if rollback != nil {
			rollback.Restore()
		}
		return errors.Join(failures...)
	}
	return nil
}

type fileSnapshot struct {
	path    string
	missing bool
	link    string
	data    []byte
	mode    os.FileMode
}

type installRollback struct {
	files   []fileSnapshot
	newDirs []string
}

func newInstallRollback(packages []*pkg.Package, installPath string) (*installRollback, error) {
	packagesDir, err := config.GetPackagesDir()
	if err != nil {
		return nil, err
	}
	r := &installRollback{}
	seen := make(map[string]bool)
	for _, p := range packages {
		if p == nil {
			return nil, errors.New("atomic install cannot include nil package")
		}
		bin := filepath.Base(p.Name)
		if p.BinName != "" {
			bin = p.BinName
		}
		target := filepath.Join(installPath, platform.ExecutableName(bin))
		if !seen[target] {
			r.files = append(r.files, snapshotFile(target))
			seen[target] = true
		}
		dir := filepath.Join(packagesDir, filepath.Base(p.Name)+"-"+p.Version)
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			r.newDirs = append(r.newDirs, dir)
		}
	}
	return r, nil
}

func snapshotFile(path string) fileSnapshot {
	if link, err := os.Readlink(path); err == nil {
		return fileSnapshot{path: path, link: link}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{path: path, missing: true}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{path: path, missing: true}
	}
	return fileSnapshot{path: path, data: data, mode: info.Mode()}
}

func (r *installRollback) Restore() {
	for _, dir := range r.newDirs {
		_ = os.RemoveAll(dir)
	}
	for _, f := range r.files {
		_ = os.Remove(f.path)
		if f.missing {
			continue
		}
		if f.link != "" {
			_ = os.Symlink(f.link, f.path)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
			continue
		}
		if err := os.WriteFile(f.path, f.data, f.mode.Perm()); err == nil {
			_ = os.Chmod(f.path, f.mode)
		}
	}
}
