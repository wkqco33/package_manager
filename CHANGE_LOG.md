# 변경 이력 (CHANGE_LOG.md)

## [1.1.0] - 2026-07-16

### 개선 사항 (Refactoring)
- **Zip 압축 해제 성능 개선 ([zip.go](file:///home/wkqco/Workspace/utils/package_manager/internal/archive/zip.go))**
  - Zip 아카이브 해제 시 대용량 패키지의 힙 메모리 점유를 방지하기 위해, 메모리 버퍼링 방식에서 임시 파일 스트리밍 방식(`os.CreateTemp` 및 `zip.NewReader`)으로 변경하였습니다.
- **바이너리 설치 및 교체 안정성 강화 ([install.go](file:///home/wkqco/Workspace/utils/package_manager/internal/archive/install.go))**
  - 실행 파일 덮어쓰기 연산의 원자성을 보장하도록 개선했습니다. 대상 바이너리를 먼저 백업(`.old`)으로 rename 처리한 뒤 안전하게 덮어쓰고, 복사 도중 실패할 경우 롤백을 수행하는 방식으로 교체 프로세스를 강화하여 Windows 및 Unix 환경의 실행 중인 바이너리 락 오류를 예방합니다.
- **HTTP Client 타임아웃 구성 분리 ([github.go](file:///home/wkqco/Workspace/utils/package_manager/internal/registry/github.go))**
  - 단일 클라이언트의 일괄 30초 타임아웃을 API용(10초 빠른 실패)과 소스 다운로드용(10분 대기)으로 나누어 정의하여 전송 안전성을 향상시켰습니다.
- **데이터 정합성 검증 도입 ([package.go](file:///home/wkqco/Workspace/utils/package_manager/internal/pkg/package.go))**
  - `Package` 구조체에 필수 속성(Name, Version, Source 등)이 누락되었는지를 체크하는 `Validate` 검증 메서드를 추가하여 런타임 이전에 스키마 정합성을 확실히 검사하도록 변경했습니다.
- **프로그레스바 렌더링 출력 꼬임 현상 해결 ([package.go](file:///home/wkqco/Workspace/utils/package_manager/internal/pkg/package.go))**
  - 다운로드가 끝났음에도 `defer`로 실행이 지연되던 `progressBody.Close()`를 압축 해제(`Extract`)가 완료된 즉시 호출되도록 수정했습니다. 이로 인해 다운로드 직후 메타데이터 저장이나 바이너리 링크 로그가 출력될 때 화면에 진행 중 상태(예: 80%)의 바가 남아 100% 완료 바와 별도의 줄에 나누어 표기되던 버그를 완전히 해결했습니다.
- **아카이브 실행 파일 링크 로직 단일화 ([install.go](file:///home/wkqco/Workspace/utils/package_manager/internal/archive/install.go), [tar.go](file:///home/wkqco/Workspace/utils/package_manager/internal/archive/tar.go), [zip.go](file:///home/wkqco/Workspace/utils/package_manager/internal/archive/zip.go))**
  - `TarArchiver`와 `ZipArchiver`에 각기 중복 구현되어 있던 약 100라인 분량의 실행 파일 검색 및 링킹 로직을 `findAndLinkExecutable` 단일 공통 함수로 통합하여 코드 유지보수성을 향상시켰습니다.
- **GitHub HTTP 요청 생성 구조 개선 ([github.go](file:///home/wkqco/Workspace/utils/package_manager/internal/registry/github.go))**
  - 여러 페치 함수에 중복으로 흩어져 있던 `http.NewRequest` 빌드 및 인증/미디어타입 헤더 세팅 구문을 `newRequest` 비공개 헬퍼 메서드로 일원화하여 보일러플레이트 코드를 줄였습니다.
- **로컬 아카이브 캐싱 시스템 구현 ([platform.go](file:///home/wkqco/Workspace/utils/package_manager/internal/platform/platform.go), [config.go](file:///home/wkqco/Workspace/utils/package_manager/internal/config/config.go), [package.go](file:///home/wkqco/Workspace/utils/package_manager/internal/pkg/package.go))**
  - 플랫폼별 표준 캐시 경로(`CacheDir`)를 지정하고, 동일 버전의 아카이브가 캐시에 존재하는 경우 외부 다운로드 요청 없이 로컬에서 즉시 패키지를 압축 해제 및 설치하도록 구현했습니다. 캐시가 없을 때는 안전하게 임시 파일로 다운로드 후 캐시에 승격하여 캐시 오염을 원천 방지합니다.
- **데이터 스키마 기반 의존성 관리 도입 ([package.go](file:///home/wkqco/Workspace/utils/package_manager/internal/pkg/package.go), [github.go](file:///home/wkqco/Workspace/utils/package_manager/internal/registry/github.go), [install.go](file:///home/wkqco/Workspace/utils/package_manager/cmd/install.go), [update.go](file:///home/wkqco/Workspace/utils/package_manager/cmd/update.go))**
  - `Package` 스키마 및 `ppmMeta` 데이터 구조에 `dependencies` 항목을 추가했습니다.
  - `install` 및 `update` CLI 명령어 실행 시 의존 관계가 정의된 패키지들을 깊이 우선 탐색(DFS) 및 위상 정렬(Topology Sort) 알고리즘을 사용해 최적 순서로 도출하고, 상호 순환 의존성 오류 검출 기능과 함께 순차적으로 안정 설치되도록 구조를 대폭 보강했습니다.


