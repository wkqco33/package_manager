package archive

import (
	"strings"

	"ppm/internal/pkg"
)

// NewArchiver는 소스 파일명에 맞는 아카이버를 반환합니다.
func NewArchiver(filename string) pkg.Archiver {
	lowerName := strings.ToLower(filename)
	if strings.HasSuffix(lowerName, ".zip") {
		return &ZipArchiver{}
	}
	// .tar.gz, .tgz 및 알 수 없는 확장자는 TarArchiver 기본 사용
	return &TarArchiver{}
}
