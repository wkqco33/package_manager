package pkg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ppm/internal/apperr"
	"ppm/internal/logger"
	"ppm/internal/ui"
)

// Package represents a package's metadata
type Package struct {
	Name        string
	Version     string
	Source      string
	Checksum    string
	Description string `json:"description"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
}

// RegistryFetcher interface for fetching metadata and downloading source from registries
type RegistryFetcher interface {
	GetMetadata(pkgName string) (*Package, error)
	DownloadSource(pkg *Package) (io.ReadCloser, int64, error) // returns reader and size to source archive
}

// Archiver interface for extracting archives and linking binaries
type Archiver interface {
	Extract(r io.Reader, destDir string) error
	Link(extractedDir, binName, targetLink string) error
}

// Install coordinates the installation of a package using standard interfaces
func Install(pkgName string, fetcher RegistryFetcher, archiver Archiver, installPath string) error {
	spinner := ui.NewSpinner("Fetching metadata for " + pkgName + "...")
	spinner.Start()
	p, err := fetcher.GetMetadata(pkgName)
	spinner.Stop()
	if err != nil {
		return apperr.Wrap(apperr.CodeRegistry, err, "metadata fetch error")
	}
	logger.Success("Metadata fetched successfully")

	logger.Info("Downloading and extracting %s version %s...", p.Name, p.Version)
	body, size, err := fetcher.DownloadSource(p)
	if err != nil {
		return apperr.Wrap(apperr.CodeNetwork, err, "download error")
	}

	bar := ui.NewProgressBar(size, 40, "Downloading")
	progressBody := &ui.ProgressReader{Reader: body, Bar: bar}
	defer progressBody.Close()

	// Extract to ~/.config/ppm/packages/<pkgName>-<version>
	home, _ := os.UserHomeDir()
	safeName := filepath.Base(p.Name) // e.g., owner/repo -> repo
	extractDir := filepath.Join(home, ".config", "ppm", "packages", fmt.Sprintf("%s-%s", safeName, p.Version))

	logger.Debug("Extracting archive", "dest", extractDir)
	if err := archiver.Extract(progressBody, extractDir); err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "extraction error")
	}

	// Link the binary. Target link = ~/.local/bin/<safeName>
	targetLink := filepath.Join(installPath, safeName)
	logger.Debug("Linking binary", "src", extractDir, "link", targetLink)

	// Default assume the binary inside archive is named exactly like the repo name (safeName)
	if err := archiver.Link(extractDir, safeName, targetLink); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "linking error")
	}

	logger.Success("Successfully installed %s!", p.Name)
	return nil
}
