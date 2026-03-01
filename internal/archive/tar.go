package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	srcFile := ""

	// 1. 최상위 디렉토리에서 바로 확인
	candidate := filepath.Join(extractedDir, binName)
	if _, err := os.Stat(candidate); err == nil {
		srcFile = candidate
	}

	// 2. 하위 디렉토리 탐색 (최대 2단계)
	if srcFile == "" {
		filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			// 디렉토리 깊이 제한 (extractedDir 기준 +2)
			rel, _ := filepath.Rel(extractedDir, path)
			depth := len(strings.Split(rel, string(os.PathSeparator)))
			if depth > 3 {
				return nil
			}

			if !info.IsDir() && info.Name() == binName {
				srcFile = path
				return filepath.SkipDir
			}
			return nil
		})
	}

	// 3. 실행 파일 이름이 다를 경우를 대비해 실행 권한이 있는 파일 검색
	if srcFile == "" {
		filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			// 실행 권한 확인 (Unix-like) 또는 확장자 확인 (Windows)
			isExec := false
			if runtime.GOOS == "windows" {
				isExec = strings.HasSuffix(strings.ToLower(info.Name()), ".exe")
			} else {
				isExec = (info.Mode() & 0111) != 0
			}

			if isExec {
				// 우선순위: binName이 포함된 파일 > 그 외 실행 파일
				if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(binName)) {
					srcFile = path
					return filepath.SkipDir
				}
				// 아직 못 찾았다면 첫 번째 실행 파일을 후보로 등록
				if srcFile == "" {
					srcFile = path
				}
			}
			return nil
		})
	}

	if srcFile == "" || srcFile == extractedDir {
		return apperr.New(apperr.CodeArchive, "executable %s not found in extracted directory %s", binName, extractedDir)
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
