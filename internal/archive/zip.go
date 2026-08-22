package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"ppm/internal/apperr"
)

// ZipArchiver는 .zip용 pkg.Archiver 구현체입니다.
type ZipArchiver struct{}

// Extract는 리더의 .zip 아카이브를 destDir로 풉니다.
func (a *ZipArchiver) Extract(r io.Reader, destDir string) error {
	// zip.NewReader는 ReaderAt이 필요하므로 임시 파일을 사용해 메모리 낭비를 줄입니다.
	tmpFile, err := os.CreateTemp("", "ppm-zip-*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create temporary zip file")
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	size, err := io.Copy(tmpFile, r)
	if err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "failed to write zip to temporary file")
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to seek temporary zip file")
	}

	zr, err := zip.NewReader(tmpFile, size)
	if err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "failed to create zip reader")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create dest directory")
	}

	for _, f := range zr.File {
		target, err := archiveTargetPath(destDir, f.Name)
		if err != nil {
			return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive path %q", f.Name)
		}
		if f.UncompressedSize64 > uint64(maxExtractedFileSize) {
			return apperr.New(apperr.CodeArchive, "archive member %q exceeds the maximum extracted file size", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := ensureNoSymlinkPath(destDir, target); err != nil {
				return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive directory %q", f.Name)
			}
			if err := os.MkdirAll(target, 0755); err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create archive directory")
			}
			continue
		}

		if err := ensureNoSymlinkPath(destDir, target); err != nil {
			return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive file %q", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create parent dir")
		}

		rc, err := f.Open()
		if err != nil {
			return apperr.Wrap(apperr.CodeArchive, err, "failed to open zip file member")
		}

		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create file")
		}

		n, copyErr := io.Copy(outFile, io.LimitReader(rc, maxExtractedFileSize+1))
		if copyErr != nil {
			outFile.Close()
			rc.Close()
			return apperr.Wrap(apperr.CodeFileSystem, copyErr, "failed to write file")
		}
		if n > maxExtractedFileSize {
			outFile.Close()
			rc.Close()
			return apperr.New(apperr.CodeArchive, "archive member %q exceeds the maximum extracted file size", f.Name)
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}

// Link는 심볼릭 링크를 만들거나 바이너리를 복사합니다.
func (a *ZipArchiver) Link(extractedDir, binName, targetLinkPath string) error {
	return findAndLinkExecutable(extractedDir, binName, targetLinkPath)
}
