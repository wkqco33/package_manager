package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ActiveDirCollector identifies package directories currently referenced by
// installed binaries. It is injected to keep cleanup orchestration testable.
type ActiveDirCollector func(packagesDir, installPath string, entries []os.DirEntry) map[string]bool

// PackageCleaner coordinates removal of inactive package versions.
type PackageCleaner struct {
	ReadDir           func(string) ([]os.DirEntry, error)
	RemoveAll         func(string) error
	Remove            func(string) error
	Readlink          func(string) (string, error)
	CollectActiveDirs ActiveDirCollector
}

// CleanAllResult describes the cleanup performed by CleanAll.
type CleanAllResult struct {
	PackagesRemoved bool
	LinksRemoved    int
}

// CleanAll removes the package directory and related symbolic links.
func (s PackageCleaner) CleanAll(packagesDir, installPath string) (CleanAllResult, error) {
	removeAll := s.RemoveAll
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	readDir := s.ReadDir
	if readDir == nil {
		readDir = os.ReadDir
	}
	remove := s.Remove
	if remove == nil {
		remove = os.Remove
	}
	readlink := s.Readlink
	if readlink == nil {
		readlink = os.Readlink
	}

	if err := removeAll(packagesDir); err != nil {
		return CleanAllResult{}, fmt.Errorf("remove package directory: %w", err)
	}
	result := CleanAllResult{PackagesRemoved: true}
	entries, err := readDir(installPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("read install directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(installPath, entry.Name())
		target, err := readlink(path)
		if err != nil || !pathReferencesDirectory(target, packagesDir) {
			continue
		}
		if err := remove(path); err != nil {
			return result, fmt.Errorf("remove link %s: %w", path, err)
		}
		result.LinksRemoved++
	}
	return result, nil
}

func pathReferencesDirectory(path, directory string) bool {
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	directoryAbs, err := filepath.Abs(directory)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(directoryAbs, pathAbs)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// CleanUnused removes package directories that are not active.
func (s PackageCleaner) CleanUnused(packagesDir, installPath string) (int, error) {
	readDir := s.ReadDir
	if readDir == nil {
		readDir = os.ReadDir
	}
	removeAll := s.RemoveAll
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	if s.CollectActiveDirs == nil {
		return 0, fmt.Errorf("package cleaner requires an active directory collector")
	}

	entries, err := readDir(packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read package directory: %w", err)
	}

	activeDirs := s.CollectActiveDirs(packagesDir, installPath, entries)
	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() || activeDirs[entry.Name()] {
			continue
		}
		if err := removeAll(filepath.Join(packagesDir, entry.Name())); err != nil {
			return removed, fmt.Errorf("remove package %s: %w", entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}
