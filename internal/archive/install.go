package archive

import (
	"io"
	"os"

	"ppm/internal/apperr"
)

// installExecutable은 src의 실행 파일을 dst 경로로 복사합니다.
//
// dst가 현재 실행 중인 바이너리인 경우(예: ppm이 자기 자신을 업데이트할 때)
// Windows에서는 파일을 삭제하거나 덮어쓸 수 없습니다. 다만 "이름 변경(rename)"은
// 허용되므로, 기존 파일을 ".old"로 옮긴 뒤 새 바이너리를 원래 경로에 기록합니다.
// 남겨진 ".old" 파일은 다음 업데이트 시 정리됩니다.
func installExecutable(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			// 삭제 실패(실행 중인 바이너리 등) → 옆으로 rename 후 진행
			backup := dst + ".old"
			_ = os.Remove(backup) // 이전 업데이트에서 남은 잔여 파일 정리 (best-effort)
			if err := os.Rename(dst, backup); err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to replace running executable")
			}
		}
	}

	return copyFileContents(src, dst)
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
