// Package apps는 ppm이 기본으로 소개하는 공개 앱 패키지 목록을 정의합니다.
package apps

import "fmt"

// DefaultApp은 ppm이 소개하는 기본 앱 패키지입니다.
type DefaultApp struct {
	Name        string // owner/repo 형식의 패키지 이름
	BinName     string // 설치 후 생성되는 실행 파일명
	Description string // 한 줄 요약 설명
	Homepage    string // 프로젝트 홈페이지
}

// DefaultApps는 ppm이 기본으로 소개하는 앱 패키지 목록입니다.
// 현재 PC에 실제 설치되어 ppm으로 설치 가능함이 확인된 앱들로 구성됩니다.
var DefaultApps = []DefaultApp{
	{Name: "wkqco33/ai-monitoring", BinName: "pcam", Description: "AI를 이용해 시스템 상태 이상을 감지하고 분석하는 PC 모니터링 도구", Homepage: "https://github.com/wkqco33/ai-monitoring"},
	{Name: "wkqco33/cli_template", BinName: "wtemp", Description: "wcli 기반 Go CLI 프로젝트 템플릿 생성기", Homepage: "https://github.com/wkqco33/cli_template"},
	{Name: "wkqco33/cpp_generator", BinName: "cppgen", Description: "C++ 프로젝트 생성기 (CLI/GUI/라이브러리 템플릿)", Homepage: "https://github.com/seoyc/cpp_generator"},
	{Name: "wkqco33/go-updater", BinName: "gu", Description: "Go 언어 버전 관리 및 업데이트 도구", Homepage: "https://github.com/wkqco33/go-updater"},
	{Name: "wkqco33/iggen", BinName: "iggen", Description: ".gitignore 파일을 자동으로 생성해주는 CLI 도구", Homepage: "https://github.com/wkqco33/iggen"},
	{Name: "wkqco33/note_cli", BinName: "ncli", Description: "CLI 메모 작성 앱", Homepage: "https://github.com/wkqco33/note_cli"},
	{Name: "wkqco33/ollama_client", BinName: "ollac", Description: "Ollama 로컬 LLM을 활용한 CLI 클라이언트", Homepage: "https://github.com/wkqco33/ollama_client"},
	{Name: "wkqco33/package_manager", BinName: "ppm", Description: "GitHub 저장소 기반의 빠르고 안전한 프라이빗 패키지 매니저", Homepage: "https://github.com/wkqco33/package_manager"},
	{Name: "wkqco33/pc_cleaner", BinName: "pcc", Description: "불필요한 파일 및 캐시를 정리하는 시스템 최적화 도구", Homepage: "https://github.com/wkqco33/pc_cleaner"},
	{Name: "wkqco33/pc_spec_checker", BinName: "pcsc", Description: "Linux, macOS, Windows 시스템의 하드웨어 사양 정보를 수집하고 표시하는 크로스 플랫폼 CLI 도구", Homepage: "https://github.com/wkqco33/pc_spec_checker"},
	{Name: "wkqco33/port_finder", BinName: "poff", Description: "포트 사용 프로세스 탐색 및 종료 유틸리티", Homepage: "https://github.com/wkqco33/port_finder"},
	{Name: "wkqco33/seckey_gen", BinName: "scgen", Description: "보안 시크릿 키 생성 유틸리티", Homepage: "https://github.com/wkqco33/seckey_gen"},
	{Name: "wkqco33/tdraw", BinName: "tdraw", Description: "원격 접속 환경에서 터미널로 이미지를 확인하는 CLI 뷰어", Homepage: "https://github.com/wkqco33/tdraw"},
	{Name: "wkqco33/wpygen", BinName: "wpygen", Description: "uv 기반 Python 프로젝트 템플릿 생성기", Homepage: "https://github.com/wkqco33/wpygen"},
}

// FilterUninstalled는 설치된 패키지 목록과 대조하여 미설치된 기본 앱 목록을 반환합니다.
// installedNames에는 owner/repo 전체 이름 또는 repo/바이너리 이름이 포함될 수 있습니다.
func FilterUninstalled(all []DefaultApp, installedNames []string) []DefaultApp {
	installedSet := make(map[string]bool, len(installedNames)*2)
	for _, name := range installedNames {
		installedSet[name] = true
		if parts := splitOwnerRepo(name); len(parts) == 2 {
			installedSet[parts[1]] = true
		}
	}

	var uninstalled []DefaultApp
	for _, app := range all {
		parts := splitOwnerRepo(app.Name)
		repoName := app.Name
		if len(parts) == 2 {
			repoName = parts[1]
		}
		if !installedSet[app.Name] && !installedSet[repoName] && !installedSet[app.BinName] {
			uninstalled = append(uninstalled, app)
		}
	}
	return uninstalled
}

func splitOwnerRepo(name string) []string {
	for i := 0; i < len(name); i++ {
		if name[i] == '/' {
			return []string{name[:i], name[i+1:]}
		}
	}
	return nil
}

// FilterByName은 지정된 패키지 이름 또는 바이너리 이름에 매칭되는 기본 앱 목록을 반환합니다.
// 매칭되지 않는 이름이 있다면 에러를 반환합니다.
func FilterByName(all []DefaultApp, names []string) ([]DefaultApp, error) {
	if len(names) == 0 {
		return append([]DefaultApp(nil), all...), nil
	}

	appLookup := make(map[string]DefaultApp, len(all)*3)
	for _, app := range all {
		appLookup[app.Name] = app
		appLookup[app.BinName] = app
		if parts := splitOwnerRepo(app.Name); len(parts) == 2 {
			appLookup[parts[1]] = app
		}
	}

	seen := make(map[string]bool)
	var matched []DefaultApp
	for _, name := range names {
		app, found := appLookup[name]
		if !found {
			return nil, fmt.Errorf("알 수 없는 기본 앱 패키지: %s", name)
		}
		if !seen[app.Name] {
			seen[app.Name] = true
			matched = append(matched, app)
		}
	}
	return matched, nil
}
