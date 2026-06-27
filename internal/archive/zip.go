package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
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

	if err := os.MkdirAll(filepath.Dir(targetLinkPath), 0755); err != nil {
		return err
	}

	// 원본 실행 권한 부여
	os.Chmod(srcFile, 0755)

	if runtime.GOOS == "windows" {
		// Windows에서는 심볼릭 링크 대신 파일을 복사하며,
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
