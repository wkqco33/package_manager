package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"ppm/internal/apperr"
)

// ZipArchiver는 .zip용 pkg.Archiver 구현체입니다.
type ZipArchiver struct{}

// Extract는 리더의 .zip 아카이브를 destDir로 풉니다.
func (a *ZipArchiver) Extract(r io.Reader, destDir string) error {
	// zip.NewReader는 ReaderAt이 필요하므로 먼저 버퍼링합니다.
	// 현재는 메모리 버퍼를 사용합니다.
	// 대용량 파일은 임시 파일이 더 적합할 수 있습니다.
	buf := new(bytes.Buffer)
	size, err := io.Copy(buf, r)
	if err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "failed to read zip into memory")
	}

	readerAt := bytes.NewReader(buf.Bytes())
	zr, err := zip.NewReader(readerAt, size)
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
	srcFile := filepath.Join(extractedDir, binName)
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		// 한 단계 하위 디렉터리까지 탐색
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

	// 기존 링크/파일 제거
	if _, err := os.Stat(targetLinkPath); err == nil {
		os.Remove(targetLinkPath)
	}

	if err := os.MkdirAll(filepath.Dir(targetLinkPath), 0755); err != nil {
		return err
	}

	// 원본 실행 권한 부여
	os.Chmod(srcFile, 0755)

	if runtime.GOOS == "windows" {
		// Windows에서는 심볼릭 링크 대신 파일을 복사합니다.
		// 필요하면 .bat 셔িম으로 대체할 수 있습니다.
		return a.copyFile(srcFile, targetLinkPath)
	}

	if err := os.Symlink(srcFile, targetLinkPath); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to link executable")
	}
	return nil
}

func (a *ZipArchiver) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
