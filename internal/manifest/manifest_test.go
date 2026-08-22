package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wkqco33/package_manager/internal/manifest"
)

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       manifest.Manifest
		wantErr bool
	}{
		{
			name: "유효한 매니페스트",
			m: manifest.Manifest{
				BinName:      "my-tool",
				Description:  "테스트 도구",
				Author:       "홍길동",
				Homepage:     "https://github.com/my-org/my-tool",
				Dependencies: map[string]string{"my-org/lib": ">=1.0.0"},
			},
			wantErr: false,
		},
		{
			name: "bin_name 누락",
			m: manifest.Manifest{
				BinName: "",
			},
			wantErr: true,
		},
		{
			name: "bin_name에 슬래시 포함",
			m: manifest.Manifest{
				BinName: "path/to/tool",
			},
			wantErr: true,
		},
		{
			name: "bin_name에 역슬래시 포함",
			m: manifest.Manifest{
				BinName: `path\to\tool`,
			},
			wantErr: true,
		},
		{
			name: "bin_name이 . 또는 ..",
			m: manifest.Manifest{
				BinName: ".",
			},
			wantErr: true,
		},
		{
			name: "dependencies의 빈 키 또는 빈 제약",
			m: manifest.Manifest{
				BinName:      "my-tool",
				Dependencies: map[string]string{"": ">=1.0.0"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManifestInitAndSaveLoad(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Init 기본 생성
	opts := manifest.InitOptions{
		Dir:         tempDir,
		BinName:     "custom-bin",
		Description: "설명 테스트",
		Author:      "작성자",
		Homepage:    "https://example.com",
	}
	m, path, err := manifest.Init(opts)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	expectedPath := filepath.Join(tempDir, "ppm.json")
	if path != expectedPath {
		t.Errorf("Init() path = %v, want %v", path, expectedPath)
	}

	if m.BinName != "custom-bin" || m.Description != "설명 테스트" {
		t.Errorf("Init() returned manifest mismatch: %+v", m)
	}

	// 2. 이미 존재하는 파일에 Force 없이 Init 시 에러
	_, _, err = manifest.Init(opts)
	if err == nil {
		t.Fatal("expected error when ppm.json already exists without Force")
	}

	// 3. Force로 덮어쓰기
	opts.Force = true
	opts.Description = "새로운 설명"
	m2, _, err := manifest.Init(opts)
	if err != nil {
		t.Fatalf("Init(Force=true) error = %v", err)
	}
	if m2.Description != "새로운 설명" {
		t.Errorf("Init(Force=true) Description = %q, want '새로운 설명'", m2.Description)
	}

	// 4. Load 테스트
	loaded, err := manifest.Load(expectedPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.BinName != "custom-bin" || loaded.Description != "새로운 설명" {
		t.Errorf("Load() mismatch: %+v", loaded)
	}
}

func TestManifestInitDefaultBinName(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "my-awesome-app")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}

	opts := manifest.InitOptions{
		Dir: subDir,
	}
	m, _, err := manifest.Init(opts)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if m.BinName != "my-awesome-app" {
		t.Errorf("Init() BinName = %q, want 'my-awesome-app'", m.BinName)
	}
}
