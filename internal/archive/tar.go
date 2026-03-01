package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
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
	var candidates []string

	// 모든 파일을 스캔하여 실행 가능한 파일 후보를 수집
	filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		// 깊이 제한 (최대 3단계까지 탐색)
		rel, _ := filepath.Rel(extractedDir, path)
		if len(strings.Split(filepath.ToSlash(rel), "/")) > 3 {
			return nil
		}

		// 실행 가능한지 확인
		isExec := false
		if runtime.GOOS == "windows" {
			isExec = strings.HasSuffix(strings.ToLower(info.Name()), ".exe")
		} else {
			// Unix: 실행 비트(x) 확인
			isExec = (info.Mode() & 0111) != 0
		}

		if isExec {
			candidates = append(candidates, path)
		} else if info.Name() == binName {
			// 이름이 정확히 일치하면 권한이 없더라도 후보에 포함 (나중에 Chmod 할 것이므로)
			candidates = append(candidates, path)
		}

		return nil
	})

	var srcFile string
	if len(candidates) > 0 {
		// 우선순위 1: binName과 정확히 일치하는 파일
		for _, c := range candidates {
			if filepath.Base(c) == binName {
				srcFile = c
				break
			}
		}

		// 우선순위 2: binName이 이름에 포함된 파일 (대소문자 무시)
		if srcFile == "" {
			for _, c := range candidates {
				if strings.Contains(strings.ToLower(filepath.Base(c)), strings.ToLower(binName)) {
					srcFile = c
					break
				}
			}
		}

		// 우선순위 3: 실행 파일이 단 하나뿐인 경우 그것을 선택
		if srcFile == "" && len(candidates) == 1 {
			srcFile = candidates[0]
		}

		// 우선순위 4: binName이 "ppm"인 경우 "ppm"이 포함된 파일 우선 (자체 프로젝트 배려)
		if srcFile == "" && (binName == "ppm" || binName == "package_manager") {
			for _, c := range candidates {
				base := strings.ToLower(filepath.Base(c))
				if base == "ppm" || base == "ppm.exe" {
					srcFile = c
					break
				}
			}
		}
	}

	if srcFile == "" {
		msg := fmt.Sprintf("executable %s not found in extracted directory %s", binName, extractedDir)
		if len(candidates) > 0 {
			var names []string
			for _, c := range candidates {
				rel, _ := filepath.Rel(extractedDir, c)
				names = append(names, rel)
			}
			msg += fmt.Sprintf(" (found other executables: %s)", strings.Join(names, ", "))
		}
		return apperr.New(apperr.CodeArchive, "%s", msg)
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
