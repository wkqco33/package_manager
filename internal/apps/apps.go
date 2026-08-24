// Package apps는 ppm이 기본으로 소개하는 공개 앱 패키지 목록을 정의합니다.
package apps

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
	{Name: "wkqco33/cli_template", BinName: "wtemp", Description: "wcli 기반 Go CLI 프로젝트 템플릿 생성기", Homepage: "https://github.com/wkqco33/cli_template"},
	{Name: "wkqco33/cpp_generator", BinName: "cppgen", Description: "C++ 프로젝트 생성기 (CLI/GUI/라이브러리 템플릿)", Homepage: "https://github.com/seoyc/cpp_generator"},
	{Name: "wkqco33/go-updater", BinName: "gu", Description: "Go 언어 버전 관리 및 업데이트 도구", Homepage: "https://github.com/wkqco33/go-updater"},
	{Name: "wkqco33/iggen", BinName: "iggen", Description: ".gitignore 파일을 자동으로 생성해주는 CLI 도구", Homepage: "https://github.com/wkqco33/iggen"},
	{Name: "wkqco33/note_cli", BinName: "ncli", Description: "CLI 메모 작성 앱", Homepage: "https://github.com/wkqco33/note_cli"},
	{Name: "wkqco33/ollama_client", BinName: "ollac", Description: "Ollama 로컬 LLM을 활용한 CLI 클라이언트", Homepage: "https://github.com/wkqco33/ollama_client"},
	{Name: "wkqco33/pc_cleaner", BinName: "pcc", Description: "불필요한 파일 및 캐시를 정리하는 시스템 최적화 도구", Homepage: "https://github.com/wkqco33/pc_cleaner"},
	{Name: "wkqco33/port_finder", BinName: "poff", Description: "포트 사용 프로세스 탐색 및 종료 유틸리티", Homepage: "https://github.com/wkqco33/port_finder"},
	{Name: "wkqco33/seckey_gen", BinName: "scgen", Description: "보안 시크릿 키 생성 유틸리티", Homepage: "https://github.com/wkqco33/seckey_gen"},
	{Name: "wkqco33/tdraw", BinName: "tdraw", Description: "원격 접속 환경에서 터미널로 이미지를 확인하는 CLI 뷰어", Homepage: "https://github.com/wkqco33/tdraw"},
	{Name: "wkqco33/wpygen", BinName: "wpygen", Description: "uv 기반 Python 프로젝트 템플릿 생성기", Homepage: "https://github.com/wkqco33/wpygen"},
}
