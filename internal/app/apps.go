package app

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wkqco33/package_manager/internal/apps"
	"github.com/wkqco33/package_manager/internal/pkg"
)

const defaultAppsConcurrency = 4

// AppsInstaller는 기본 추천 앱 패키지들의 동시 설치를 조율하는 서비스입니다.
type AppsInstaller struct {
	Fetcher          pkg.RegistryFetcher
	InstallPath      string
	NewArchiver      ArchiverFactory
	Concurrency      int
	AllowSourceBuild bool
	Offline          bool
}

// AppsInstallResult는 기본 앱 설치 작업의 결과를 나타냅니다.
type AppsInstallResult struct {
	Installed []*pkg.Package
	Failed    map[string]error
}

type metadataTaskResult struct {
	index int
	app   apps.DefaultApp
	pkg   *pkg.Package
	err   error
}

// InstallApps는 전달된 기본 앱 목록을 Bounded Concurrency 고루틴으로 설치합니다.
func (s AppsInstaller) InstallApps(targets []apps.DefaultApp, dryRun bool) (*AppsInstallResult, error) {
	if s.Fetcher == nil {
		return nil, errors.New("apps installer requires a fetcher")
	}
	if s.NewArchiver == nil {
		return nil, errors.New("apps installer requires an archiver factory")
	}

	result := &AppsInstallResult{
		Failed: make(map[string]error),
	}
	if len(targets) == 0 {
		return result, nil
	}

	concurrency := s.Concurrency
	if concurrency <= 0 {
		concurrency = defaultAppsConcurrency
	}

	// 1단계: Bounded Concurrency 기반 메타데이터 병렬 조회
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	metaResults := make([]metadataTaskResult, len(targets))

	for i, target := range targets {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, t apps.DefaultApp) {
			defer wg.Done()
			defer func() { <-sem }()

			p, err := s.Fetcher.GetMetadata(t.Name)
			if err != nil {
				metaResults[idx] = metadataTaskResult{index: idx, app: t, err: err}
				return
			}
			if p == nil {
				metaResults[idx] = metadataTaskResult{index: idx, app: t, err: fmt.Errorf("%s: metadata is nil", t.Name)}
				return
			}
			// 사전에 정의된 BinName을 메타데이터에 반영
			if t.BinName != "" {
				p.BinName = t.BinName
			}
			metaResults[idx] = metadataTaskResult{index: idx, app: t, pkg: p}
		}(i, target)
	}

	wg.Wait()

	var packagesToInstall []*pkg.Package
	for _, mr := range metaResults {
		if mr.err != nil {
			result.Failed[mr.app.Name] = mr.err
		} else if mr.pkg != nil {
			packagesToInstall = append(packagesToInstall, mr.pkg)
		}
	}

	if dryRun {
		result.Installed = packagesToInstall
		if len(result.Failed) > 0 {
			var errList []error
			for _, err := range result.Failed {
				errList = append(errList, err)
			}
			return result, errors.Join(errList...)
		}
		return result, nil
	}

	// 2단계: 패키지 설치 수행
	// 개별 패키지별로 설치하여 일부 패키지가 실패해도 나머지 패키지는 설치되도록 보장
	installer := PackageInstaller{
		Fetcher:          s.Fetcher,
		InstallPath:      s.InstallPath,
		NewArchiver:      s.NewArchiver,
		AllowSourceBuild: s.AllowSourceBuild,
		Offline:          s.Offline,
	}

	for _, p := range packagesToInstall {
		if err := installer.Install([]*pkg.Package{p}); err != nil {
			result.Failed[p.Name] = err
		} else {
			result.Installed = append(result.Installed, p)
		}
	}

	if len(result.Failed) > 0 {
		var errList []error
		for _, err := range result.Failed {
			errList = append(errList, err)
		}
		return result, errors.Join(errList...)
	}

	return result, nil
}
