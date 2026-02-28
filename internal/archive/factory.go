package archive

import (
	"strings"

	"ppm/internal/pkg"
)

// NewArchiver는 소스 파일명에 맞는 아카이버를 반환합니다.
func NewArchiver(filename, binName string) pkg.Archiver {
	lowerName := strings.ToLower(filename)
	if strings.HasSuffix(lowerName, ".zip") {
		return &ZipArchiver{}
	}
	if strings.HasSuffix(lowerName, ".tar.gz") || strings.HasSuffix(lowerName, ".tgz") {
		return &TarArchiver{}
	}

	// 압축 형식이 아닌 경우 단일 바이너리로 간주
	return &BinaryArchiver{BinName: binName}
}
