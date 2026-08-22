package pkg

import "fmt"

// MetadataFetcher는 패키지 메타데이터 조회에 필요한 최소 의존성입니다.
// 다운로드까지 필요한 RegistryFetcher와 분리해 의존성 해석을 독립적으로 테스트할 수 있게 합니다.
type MetadataFetcher interface {
	GetMetadata(pkgName string) (*Package, error)
}

// ResolveDependencies는 깊이 우선 탐색과 위상 정렬을 사용해 패키지를
// 의존성 설치 순서로 반환합니다. 각 패키지는 결과에 한 번만 포함됩니다.
func ResolveDependencies(pkgNames []string, fetcher MetadataFetcher) ([]*Package, error) {
	resolved := make([]*Package, 0, len(pkgNames))
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if visited[name] {
			return nil
		}
		if name == "" {
			return fmt.Errorf("package name is empty")
		}

		visiting[name] = true
		p, err := fetcher.GetMetadata(name)
		if err != nil {
			return err
		}
		if p == nil {
			return fmt.Errorf("metadata for %s is nil", name)
		}

		for _, dependency := range p.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}

		delete(visiting, name)
		visited[name] = true
		resolved = append(resolved, p)
		return nil
	}

	for _, name := range pkgNames {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}
