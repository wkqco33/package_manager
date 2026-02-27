package archive

import (
	"archive/tar"
	"bufio"
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

			bw := bufio.NewWriterSize(outFile, 64*1024)
			if _, err := io.Copy(bw, tr); err != nil {
				outFile.Close()
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write file")
			}
			if err := bw.Flush(); err != nil {
				outFile.Close()
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to flush file")
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
		// GitHub source tarballs extract into a single top-level subdirectory
		// (e.g., owner-repo-sha/). Search one level deep before giving up.
		entries, readErr := os.ReadDir(extractedDir)
		if readErr == nil && len(entries) == 1 && entries[0].IsDir() {
			candidate := filepath.Join(extractedDir, entries[0].Name(), binName)
			if _, statErr := os.Stat(candidate); statErr == nil {
				srcFile = candidate
			}
		}
		if _, statErr := os.Stat(srcFile); os.IsNotExist(statErr) {
			return apperr.New(apperr.CodeArchive, "executable %s not found in extracted directory %s", binName, extractedDir)
		}
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
