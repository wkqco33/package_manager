# 변경 이력 (CHANGE_LOG.md)

## [Unreleased]

### 추가 사항 (Features)

- **기본 앱 패키지 소개 커맨드 추가 ([apps.go](cmd/apps.go), [apps.go](internal/apps/apps.go))**
  - `ppm apps` 커맨드로 ppm으로 설치 가능한 기본 앱 패키지 목록을 소개합니다.
  - 각 앱의 설명·홈페이지와 함께 설치 상태를 표시하며, `--json` 플래그로 자동화에 활용할 수 있습니다.
  - 기본 앱 목록은 현재 PC에 실제 설치되어 ppm으로 설치 가능함이 확인된 앱들로 구성됩니다.

## [1.1.0] - 2026-07-16

### 개선 사항 (Refactoring)

- **Zip 압축 해제 성능 개선 ([zip.go](internal/archive/zip.go))**
  - Zip 아카이브 해제 시 대용량 패키지의 힙 메모리 점유를 방지하기 위해 임시 파일 스트리밍 방식(`os.CreateTemp` 및 `zip.NewReader`)을 사용합니다.
- **바이너리 설치 및 교체 안정성 강화 ([install.go](internal/archive/install.go))**
  - 실행 파일 교체 시 백업과 롤백을 사용해 Windows 및 Unix 환경의 파일 잠금과 복사 실패를 안전하게 처리합니다.
- **HTTP Client 타임아웃 구성 분리 ([github.go](internal/registry/github.go))**
  - API 요청과 소스 다운로드에 서로 다른 timeout을 적용합니다.
- **데이터 정합성 검증 도입 ([package.go](internal/pkg/package.go))**
  - `Package` 필수 속성 검증을 추가했습니다.
- **프로그레스바 렌더링 출력 개선 ([package.go](internal/pkg/package.go))**
  - 다운로드와 압축 해제 진행 상태가 올바르게 종료되도록 수정했습니다.
- **아카이브 실행 파일 링크 로직 단일화 ([install.go](internal/archive/install.go), [tar.go](internal/archive/tar.go), [zip.go](internal/archive/zip.go))**
  - 실행 파일 검색 및 링크 로직을 공통 함수로 통합했습니다.
- **GitHub HTTP 요청 생성 구조 개선 ([github.go](internal/registry/github.go))**
  - 요청 생성 및 인증 헤더 설정을 공통 헬퍼로 통합했습니다.
- **로컬 아카이브 캐싱 시스템 구현 ([platform.go](internal/platform/platform.go), [config.go](internal/config/config.go), [package.go](internal/pkg/package.go))**
  - 플랫폼별 캐시에 다운로드를 임시 파일로 저장한 뒤 성공 시 승격하여 캐시 오염을 방지합니다.
- **데이터 스키마 기반 의존성 관리 도입 ([package.go](internal/pkg/package.go), [github.go](internal/registry/github.go), [install.go](cmd/install.go), [update.go](cmd/update.go))**
  - 의존성의 깊이 우선 탐색, 설치 순서 정렬, 순환 의존성 검출을 지원합니다.
