package archive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wkqco33/package_manager/internal/apperr"
)

// installExecutable은 src의 실행 파일을 dst 경로로 복사합니다.
//
// dst가 현재 실행 중인 바이너리인 경우(예: ppm이 자기 자신을 업데이트할 때)
// Windows에서는 파일을 삭제하거나 덮어쓸 수 없습니다. 다만 "이름 변경(rename)"은
// 허용되므로, 기존 파일을 ".old"로 옮긴 뒤 새 바이너리를 원래 경로에 기록합니다.
// 남겨진 ".old" 파일은 다음 업데이트 시 정리됩니다.
func installExecutable(src, dst string) error {
	backup := dst + ".old"
	hasBackup := false

	if _, err := os.Stat(dst); err == nil {
		// 기존의 잔여 백업 파일이 있으면 미리 지웁니다.
		_ = os.Remove(backup)

		// 기존 바이너리를 백업 경로로 무조건 rename 시도 (실행 중이더라도 이름 변경 가능)
		if err := os.Rename(dst, backup); err == nil {
			hasBackup = true
		} else {
			// 만약 rename 실패 시 직접 삭제를 시도합니다.
			if err := os.Remove(dst); err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to move or remove existing executable")
			}
		}
	}

	// 새로운 바이너리를 대상 경로에 복사
	if err := copyFileContents(src, dst); err != nil {
		// 복사 도중 에러 발생 시, 백업이 존재하면 복원 시도
		if hasBackup {
			_ = os.Rename(backup, dst)
		}
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to copy executable contents")
	}

	// 교체가 성공하면 백업 파일 정리 (실패하더라도 전체 과정은 성공으로 간주)
	if hasBackup {
		_ = os.Remove(backup)
	}

	return nil
}

// copyFileContents는 src 파일의 내용을 dst로 복사합니다.
func copyFileContents(src, dst string) error {
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

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}
	return nil
}

// findAndLinkExecutable은 extractedDir에서 실행 파일 후보를 찾고 우선순위에 맞춰 targetLinkPath에 링크를 생성합니다.
func findAndLinkExecutable(extractedDir, binName, targetLinkPath string) error {
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
