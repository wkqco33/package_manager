package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

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
		target := filepath.Join(destDir, f.Name)

		// ZipSlip 취약점 방지
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) && target != destDir {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
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

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write file")
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
