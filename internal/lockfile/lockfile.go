package lockfile

import (
	"fmt"
	"os"

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
		return err
	}
	tmp, err := os.CreateTemp(".", ".ppm-lock-*.tmp")
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
	return os.Rename(tmpName, path)
}
