package lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/wkqco33/package_manager/internal/pkg"
)

const CurrentVersion = 1

type File struct {
	Version  int            `yaml:"version"`
	Packages []*pkg.Package `yaml:"packages"`
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("invalid lockfile: %w", err)
	}
	if f.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported lockfile version %d", f.Version)
	}
	for i, p := range f.Packages {
		if p == nil {
			return nil, fmt.Errorf("lockfile package %d is null", i)
		}
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("invalid package %d: %w", i, err)
		}
	}
	return &f, nil
}

func Save(path string, packages []*pkg.Package) error {
	data, err := yaml.Marshal(&File{Version: CurrentVersion, Packages: packages})
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
	}
	// 임시 파일은 최종 파일과 같은 디렉터리에 만들어야 Windows에서도
	// os.Rename이 디스크 간 이동으로 해석되지 않습니다.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ppm-lock-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { tmp.Close(); os.Remove(tmpName) }()
	if err := tmp.Chmod(0600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		// Windows는 대상 파일이 존재할 때 Rename을 허용하지 않으므로
		// 기존 파일을 먼저 제거한 뒤 동일 디렉터리의 임시 파일을 이동합니다.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to replace lockfile: %w", err)
		}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to finalize lockfile: %w", err)
	}
	return nil
}
