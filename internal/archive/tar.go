package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"ppm/internal/apperr"
	"ppm/internal/pkg"
)

// TarArchiver는 .tar.gz용 pkg.Archiver 구현체입니다.
type TarArchiver struct{}

// TarArchiver가 pkg.Archiver를 구현하는지 확인
var _ pkg.Archiver = (*TarArchiver)(nil)

// Extract는 리더의 .tar.gz 아카이브를 destDir로 풉니다.
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
			// 상위 디렉터리 보장
			parent := filepath.Dir(target)
			if !dirsCreated[parent] {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create parent dir")
				}
				dirsCreated[parent] = true
			}

			// tar 헤더의 모드값 사용
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
			// 유닉스 계열에서는 심볼릭 링크가 자주 사용됩니다.
			// Windows는 제외하고 가능한 경우만 생성합니다.
			if runtime.GOOS != "windows" {
				if err := os.Symlink(header.Linkname, target); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create symlink %s", header.Name)
				}
			}
		}
	}
	return nil
}

// Link는 심볼릭 링크를 만들거나 바이너리를 복사합니다.
func (a *TarArchiver) Link(extractedDir, binName, targetLinkPath string) error {
	srcFile := filepath.Join(extractedDir, binName)
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		// GitHub 소스 tarball은 최상위 하위 디렉터리로 풀리는 경우가 많음.
		// 모든 하위 디렉터리를 한 단계 깊이까지 탐색합니다.
		entries, readErr := os.ReadDir(extractedDir)
		if readErr == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					candidate := filepath.Join(extractedDir, entry.Name(), binName)
					if _, statErr := os.Stat(candidate); statErr == nil {
						srcFile = candidate
						break
					}
				}
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
		return a.copyFile(srcFile, targetLinkPath)
	}

	if err := os.Symlink(srcFile, targetLinkPath); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to link executable")
	}
	return nil
}

func (a *TarArchiver) copyFile(src, dst string) error {
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
