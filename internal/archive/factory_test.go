package archive

import (
	"testing"
)

func TestNewArchiver(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		wantType string
	}{
		{"zip", "release.zip", "*archive.ZipArchiver"},
		{"zip 대문자", "RELEASE.ZIP", "*archive.ZipArchiver"},
		{"tar.gz", "source.tar.gz", "*archive.TarArchiver"},
		{"tgz", "source.tgz", "*archive.TarArchiver"},
		{"확장자 없음(단일 바이너리)", "ppm", "*archive.BinaryArchiver"},
		{"기타 확장자", "binary.bin", "*archive.BinaryArchiver"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := NewArchiver(c.filename, "ppm")
			got := typeName(a)
			if got != c.wantType {
				t.Errorf("NewArchiver(%q) 타입 기대값 %s, 실제 %s", c.filename, c.wantType, got)
			}
		})
	}
}

func TestNewArchiverBinaryName(t *testing.T) {
	a := NewArchiver("standalone-binary", "my-tool")
	ba, ok := a.(*BinaryArchiver)
	if !ok {
		t.Fatalf("BinaryArchiver를 기대했으나 %T", a)
	}
	if ba.BinName != "my-tool" {
		t.Errorf("BinName 기대값 my-tool, 실제 %s", ba.BinName)
	}
}

func typeName(v interface{}) string {
	switch v.(type) {
	case *ZipArchiver:
		return "*archive.ZipArchiver"
	case *TarArchiver:
		return "*archive.TarArchiver"
	case *BinaryArchiver:
		return "*archive.BinaryArchiver"
	default:
		return "unknown"
	}
}
