package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"ppm/internal/apperr"
	"ppm/internal/pkg"
)

// TarArchiver implements pkg.Archiver for .tar.gz files
type TarArchiver struct{}

// Ensure TarArchiver implements pkg.Archiver
var _ pkg.Archiver = (*TarArchiver)(nil)

// Extract extracts a .tar.gz archive from a reader into destDir
func (a *TarArchiver) Extract(r io.Reader, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create dest directory")
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "failed to create gzip reader")
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	dirsCreated := make(map[string]bool)
	dirsCreated[destDir] = true

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apperr.Wrap(apperr.CodeArchive, err, "failed to read tar header")
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if !dirsCreated[target] {
				if err := os.MkdirAll(target, 0755); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create dir in archive")
				}
				dirsCreated[target] = true
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			parent := filepath.Dir(target)
			if !dirsCreated[parent] {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create parent dir")
				}
				dirsCreated[parent] = true
			}

			// Pass the mode from the tar header
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create file")
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write file")
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create symlink %s", header.Name)
			}
		}
	}
	return nil
}

// Link creates a symbolic link from binPath to targetDir/executable_name
func (a *TarArchiver) Link(extractedDir, binName, targetLinkPath string) error {
	// targetLinkPath is like ~/.local/bin/my-app
	// binName is what we extracted

	srcFile := filepath.Join(extractedDir, binName)
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		return apperr.New(apperr.CodeArchive, "executable %s not found in extracted directory %s", binName, extractedDir)
	}

	// Create symlink
	// If it already exists, remove it first
	if _, err := os.Stat(targetLinkPath); err == nil {
		os.Remove(targetLinkPath)
	}

	// Ensure target directory exists (~/.local/bin)
	if err := os.MkdirAll(filepath.Dir(targetLinkPath), 0755); err != nil {
		return err
	}

	// Make source executable
	os.Chmod(srcFile, 0755)

	if err := os.Symlink(srcFile, targetLinkPath); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to link executable")
	}
	return nil
}
