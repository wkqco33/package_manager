# 변경 이력 (CHANGELOG.md)

## [Unreleased]

## [0.2.13] - 2026-09-21

### 개선 사항 (Refactoring)

- **소스 tarball 폴백 거부 메시지 구체화 ([package.go](internal/pkg/package.go), [github.go](internal/registry/github.go))**
  - 릴리스에 현재 플랫폼용 바이너리 asset이 없어 설치가 중단될 때, 감지된 플랫폼(`darwin/arm64` 등)과 릴리스에 존재하는 설치 가능한 asset 목록을 함께 안내합니다.
  - 체크섬·서명 사이드카는 목록에서 제외하고, 10개를 넘으면 `외 N개`로 요약합니다.
  - 진단 정보(`AssetDiagnostics`)는 `ppm-meta.json`과 `ppm.lock`에 직렬화하지 않습니다.

- **인증 소스 가시성 강화 (`ppm auth status` / `ppm auth logout`) ([auth.go](cmd/auth.go), [auth.go](internal/auth/auth.go), [config.go](internal/config/config.go))**
  - `ppm auth status`가 해석 우선순위대로 모든 인증 소스(`PPM_AUTH_TOKEN`, `GITHUB_TOKEN`, `config.yaml`, credential store, `gh`)의 감지 여부를 표시합니다.
  - `ppm auth logout`이 ppm이 저장한 토큰만 삭제된다는 점과, 남아 있는 `gh`·환경 변수 토큰의 해제 방법(`gh auth logout --hostname github.com`, `unset`)을 안내합니다.
  - `ppm auth login`이 `gh` 토큰을 재사용할 때 ppm에 별도 토큰을 저장하지 않음을 명시합니다.
  - `config.ReadConfigToken`을 추가해 `config.yaml`의 원시 `auth_token`과 해석된 토큰(`LoadConfig`)을 구분합니다.

### 테스트 (Tests)

- **테스트 경로 환경 격리 보강 ([package_test.go](internal/pkg/package_test.go), [install_integration_test.go](internal/pkg/install_integration_test.go), [apps_test.go](cmd/apps_test.go), [apps_test.go](internal/app/apps_test.go), [update_test.go](internal/app/update_test.go), [install_test.go](internal/app/install_test.go))**
  - `platform.GetPaths()`는 `HOME`보다 `APPDATA`·`LOCALAPPDATA`·`PPM_CONFIG_DIR`·`XDG_CONFIG_HOME` 같은 명시적 경로를 우선하므로, 테스트 헬퍼가 이 변수들도 함께 비우도록 수정했습니다.
  - 그렇지 않으면 해당 변수가 설정된 CI 러너에서 모든 테스트가 러너의 실제 홈·캐시 디렉터리를 공유해, 이전 실행의 잔여물 때문에 실행 순서에 따라 실패했습니다.
  - 패키지 디렉터리와 캐시 디렉터리가 격리된 임시 홈 아래에 있는지 검증하는 테스트를 추가했습니다.

## [0.2.11] - 2026-09-15

### 추가 사항 (Features)

- **기본 앱 자동 설치 및 고루틴 동시성 지원 (`ppm apps --install`) ([apps.go](cmd/apps.go), [apps.go](internal/app/apps.go))**
  - `--install` (`-i`) 플래그로 미설치된 기본 앱들을 자동 감지하여 한 번에 설치할 수 있습니다.
  - Bounded Concurrency(세마포어 채널) 워커 풀을 적용해 메타데이터 조회 및 아카이브 다운로드를 병렬 처리하여 설치 시간을 극적으로 단축했습니다.
  - `--all` (`-a`): 이미 설치된 앱을 포함하여 모든 기본 앱 설치/재설치 지원.
  - `--concurrency` (`-c`): 동시 설치 고루틴 수 지정 (기본값: 4).
  - `--dry-run`: 실제 설치 없이 설치 계획 확인 지원.
  - `ppm apps -i [package...]`: 원하는 기본 앱만 지정하여 설치 지원.
  - 기본 앱 목록에 `ai-monitoring`, `package_manager`, `pc_spec_checker` 추가 (총 14개).

- **기본 앱 패키지 소개 커맨드 ([apps.go](cmd/apps.go), [apps.go](internal/apps/apps.go))**
  - `ppm apps` 커맨드로 ppm으로 설치 가능한 기본 앱 패키지 목록을 소개합니다.
  - 각 앱의 설명·홈페이지와 함께 설치 상태를 표시하며, `--json` 플래그로 자동화에 활용할 수 있습니다.

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
