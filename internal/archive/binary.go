package archive

import (
	"io"
	"os"
	"path/filepath"
	"runtime"

	"ppm/internal/apperr"
	"ppm/internal/pkg"
)

// BinaryArchiver는 압축되지 않은 단일 바이너리용 pkg.Archiver 구현체입니다.
type BinaryArchiver struct {
	BinName string
}

// BinaryArchiver가 pkg.Archiver를 구현하는지 확인
var _ pkg.Archiver = (*BinaryArchiver)(nil)

// Extract는 리더의 내용을 destDir/BinName 파일로 직접 저장합니다.
func (a *BinaryArchiver) Extract(r io.Reader, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create dest directory")
	}

	target := filepath.Join(destDir, a.BinName)
	outFile, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create binary file")
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, r); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write binary file")
	}

	return nil
}

// Link는 단일 바이너리를 심볼릭 링크하거나 복사합니다. (TarArchiver.Link와 유사함)
func (a *BinaryArchiver) Link(extractedDir, binName, targetLinkPath string) error {
	srcFile := filepath.Join(extractedDir, a.BinName)
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		return apperr.New(apperr.CodeArchive, "executable %s not found in directory %s", a.BinName, extractedDir)
	}

	if err := os.MkdirAll(filepath.Dir(targetLinkPath), 0755); err != nil {
		return err
	}

	// 원본 실행 권한 부여
	os.Chmod(srcFile, 0755)

	if runtime.GOOS == "windows" {
		// 실행 중인 바이너리도 안전하게 교체합니다.
		return installExecutable(srcFile, targetLinkPath)
	}

	// 기존 링크/파일 제거 후 심볼릭 링크 생성
	if _, err := os.Stat(targetLinkPath); err == nil {
		os.Remove(targetLinkPath)
	}
	if err := os.Symlink(srcFile, targetLinkPath); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to link executable")
	}
	return nil
}
