package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/wkqco33/package_manager/internal/apperr"
)

// Manifest는 패키지 ppm.json 파일의 데이터 스키마입니다.
type Manifest struct {
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Homepage     string            `json:"homepage"`
	BinName      string            `json:"bin_name"`
	Dependencies map[string]string `json:"dependencies"`
}

// InitOptions는 매니페스트 초기화 옵션입니다.
type InitOptions struct {
	Dir         string
	BinName     string
	Description string
	Author      string
	Homepage    string
	Force       bool
}

// Validate는 매니페스트 데이터의 유효성을 검증합니다.
func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.BinName) == "" {
		return apperr.New(apperr.CodeInvalidInput, "bin_name은 필수입니다")
	}
	if strings.ContainsAny(m.BinName, `/\\`) || m.BinName == "." || m.BinName == ".." {
		return apperr.New(apperr.CodeInvalidInput, "bin_name에 경로 구분자나 특수 경로를 사용할 수 없습니다")
	}
	for name, constraint := range m.Dependencies {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(constraint) == "" {
			return apperr.New(apperr.CodeInvalidInput, "의존성 이름과 버전 제약은 비어 있을 수 없습니다")
		}
	}
	return nil
}

// Load는 주어진 경로의 ppm.json 파일을 읽고 파싱합니다.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeFileSystem, err, "manifest를 읽을 수 없습니다")
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, apperr.Wrap(apperr.CodeInvalidInput, err, "잘못된 JSON 형식입니다")
	}
	return &m, nil
}

// Save는 주어진 경로에 ppm.json 파일을 저장합니다.
func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.CodeInvalidInput, err, "manifest 직렬화 실패")
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "manifest 파일 저장 실패")
	}
	return nil
}

// Init은 지정된 디렉토리에 ppm.json 매니페스트를 생성합니다.
func Init(opts InitOptions) (*Manifest, string, error) {
	targetDir := opts.Dir
	if strings.TrimSpace(targetDir) == "" {
		targetDir = "."
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, "", apperr.Wrap(apperr.CodeFileSystem, err, "디렉토리 경로를 확인할 수 없습니다")
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, "", apperr.Wrap(apperr.CodeFileSystem, err, "디렉토리 생성에 실패했습니다")
	}

	targetPath := filepath.Join(absDir, "ppm.json")
	if _, err := os.Stat(targetPath); err == nil && !opts.Force {
		return nil, targetPath, apperr.New(apperr.CodeFileSystem, "%s 파일이 이미 존재합니다. 덮어쓰려면 --force 플래그를 사용하세요", targetPath)
	}

	binName := strings.TrimSpace(opts.BinName)
	if binName == "" {
		binName = filepath.Base(absDir)
		if binName == "." || binName == "/" || binName == "" {
			binName = "app"
		}
	}

	m := &Manifest{
		Description:  opts.Description,
		Author:       opts.Author,
		Homepage:     opts.Homepage,
		BinName:      binName,
		Dependencies: make(map[string]string),
	}

	if err := m.Validate(); err != nil {
		return nil, targetPath, err
	}

	if err := m.Save(targetPath); err != nil {
		return nil, targetPath, err
	}

	return m, targetPath, nil
}
