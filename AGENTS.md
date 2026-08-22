# AGENTS.md - ppm 개발 가이드

이 문서는 `ppm (Private Package Manager)` 프로젝트에 참여하는 모든 AI 에이전트 및 개발자를 위한 개발 지침입니다. 프로젝트의 일관된 코드 품질과 아키텍처 원칙을 유지하기 위해 아래 내용을 반드시 숙지하고 따라야 합니다.

---

## 1. 핵심 개발 철학

- **탈객체지향 & 데이터 중심 설계**:
  - 복잡한 클래스 상속이나 불필요한 계층 구조를 지양합니다.
  - Go의 구조체(`struct`)를 활용해 데이터 스키마와 타입을 명확히 정의하고, 순수 함수 중심으로 로직을 구성합니다.
- **성능 최우선 (Performance First)**:
  - 불필요한 추상화나 메모리 복사, 임시 파일 생성을 최소화합니다.
  - 대용량 다운로드 및 아카이브 처리는 스트리밍 파이프라인(`io.Reader`, `io.Writer`)을 활용합니다.
- **크로스 플랫폼 지원**:
  - Linux, macOS, Windows 환경을 모두 지원해야 합니다.
  - 경로 구분자(`filepath.Join`), 바이너리 확장자(`platform.ExecutableName`), OS별 표준 설정 경로(`platform.GetPaths`)를 항상 고려합니다.
- **구조화된 에러 및 로깅**:
  - 무분별한 `panic`을 금지하며, `internal/apperr` 패키지를 통해 명확한 에러를 반환합니다.
  - 터미널 출력 및 로깅은 `internal/ui`와 `internal/logger`를 사용합니다.

---

## 2. TDD (테스트 주도 개발) 워크플로우

본 프로젝트는 **TDD(Test-Driven Development)** 방식으로 개발합니다. 기능 구현이나 버그 수정 시 항상 테스트 코드를 먼저 작성하거나 병행하여 검증해야 합니다.

### 개발 3단계 (Red-Green-Refactor)
1. **Red (실패하는 테스트 작성)**:
   - 요구사항과 입력/출력 인터페이스를 정의하고, 실패하는 단위 테스트(`*_test.go`)를 먼저 작성합니다.
2. **Green (최소 구현으로 통과)**:
   - 테스트를 통과하기 위한 가장 단순하고 직관적인 코드를 구현합니다.
3. **Refactor (리팩토링 & 성능 최적화)**:
   - 성능과 가독성을 개선하고 불필요한 추상화를 제거합니다.
   - 테스트가 계속 통과하는지 확인합니다.

### 테스트 작성 수칙
- **빠르고 독립적인 테스트**:
  - 단위 테스트는 외부 네트워크 요청 없이 밀리초 단위로 빠르게 실행되어야 합니다.
  - 외부 의존성(GitHub API, 아카이브 다운로드 등)은 간단한 Mock 인터페이스(`httptest.Server`, `MockFetcher`, `MockArchiver`)로 격리합니다.
- **환경 및 파일시스템 격리**:
  - 테스트 중 로컬 환경을 오염시키지 않도록 `t.TempDir()` 및 `t.Setenv()`를 사용합니다.
  - 사용자 홈 디렉토리 관련 테스트는 `setupTempHome(t)` 패턴을 적용하여 OS별 환경변수를 임시 디렉토리로 격리합니다.
- **Table-Driven Tests 활용**:
  - 다양한 엣지 케이스와 오류 케이스는 Go의 표준 Table-Driven 테스트 패턴(`map[string]struct` 또는 슬라이스)을 사용합니다.
- **경쟁 상태 검증**:
  - 동시성 로직이 포함된 경우 `task test-race`로 데이터 레이스를 사전에 방지합니다.

---

## 3. 프로젝트 구조

```
package_manager/
├── cmd/                # CLI 커맨드 및 플래그 파싱 (Cobra)
│   ├── root.go         # 루트 커맨드
│   ├── install.go      # ppm install
│   ├── update.go       # ppm update
│   ├── uninstall.go    # ppm uninstall
│   └── ...
├── internal/           # 내부 핵심 패키지
│   ├── app/            # CLI 커맨드와 코어 로직을 잇는 어플리케이션 서비스 계층
│   ├── apperr/         # 구조화된 에러 정의 및 에러 래핑
│   ├── archive/        # tar.gz, zip, raw binary 스트리밍 압축 해제 및 팩토리
│   ├── config/         # config.yaml 로딩, 생성, 보안 권한 검증
│   ├── lockfile/       # ppm.lock 파일 관리
│   ├── logger/         # 구조화된 로깅
│   ├── pkg/            # 패키지 모델, 의존성 해결, 설치/제거/검증 핵심 로직
│   ├── platform/       # OS/Arch 감지 및 크로스 플랫폼 표준 경로 계산
│   ├── registry/       # GitHub API 및 외부 레지스트리 메타데이터/다운로드 클라이언트
│   └── ui/             # CLI 터미널 출력 및 프로그레스 바
├── Taskfile.yml        # 빌드, 테스트, 린트 자동화 태스크 정의
├── go.mod / go.sum     # Go 모듈 의존성 정의
└── README.md           # 프로젝트 소개 및 사용 설명서
```

---

## 4. 빌드 및 검증 커맨드

모든 변경 작업 후에는 아래의 `task` 명령어를 실행하여 코드 품질을 검증해야 합니다.

| 명령어 | 설명 |
| :--- | :--- |
| `task test` | 전체 단위 테스트 실행 (`go test -v -count=1 ./...`) |
| `task test-race` | 데이터 경쟁 검출 테스트 (`go test -race -count=1 ./...`) |
| `task lint` | 정적 분석 검사 (`go vet ./...`) |
| `task coverage` | 패키지별 테스트 커버리지 확인 |
| `task fmt` | Go 표준 코드 스타일 포맷팅 (`go fmt ./...`) |
| `task build` | 로컬 최적화 바이너리 빌드 (`-s -w` ldflags 적용) |
| `task build-all` | 멀티 플랫폼 바이너리 빌드 (Linux, macOS, Windows) |

---

## 5. 에이전트 작업 지침

1. **작은 단위로 단계별 진행**:
   - 한 번에 대규모 변경을 하지 않고, 패키지/기능 단위로 나누어 구현 및 검증을 완료합니다.
2. **테스트 검증 필수**:
   - 코드 수정을 마친 후에는 반드시 `task lint`, `task test`, `gofmt -l .`를 실행하여 린트 오류나 깨진 테스트가 없는지 확인합니다.
3. **일관된 문서화 및 커뮤니케이션**:
   - 모든 문서와 설명, 커밋 메시지는 간결하고 명확한 **한국어**로 작성합니다.
   - 불필요한 수식어나 장황한 설명은 지양하고 핵심 사항만 전달합니다.
