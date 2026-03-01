# 패키지 등록 가이드 (Packaging Guide)

`ppm` (Private Package Manager)을 통해 설치 가능한 패키지를 GitHub에 등록하는 방법을 설명합니다.

## 1. 레포지토리 및 바이너리 이름 규칙

`ppm`은 기본적으로 레포지토리의 이름을 바이너리 이름으로 간주합니다.

* **레포지토리 이름**: `my-tool`
* **실행 파일 이름**: `my-tool` (확장자 없음)

레포지토리를 빌드했을 때 생성되는 최종 실행 파일의 이름이 레포지토리 이름과 정확히 일치해야 설치 후 자동으로 링크(`~/.local/bin/my-tool`)가 생성됩니다.

## 2. 권장 레포지토리 구조

가장 간단한 구조는 레포지토리 루트에 바이너리가 포함된 상태로 릴리스하는 것입니다.

```text
my-tool/
├── main.go
├── go.mod
└── my-tool  <-- 실행 파일 (레포지토리 이름과 동일)
```

> **참고**: GitHub에서 제공하는 자동 소스 코드 tarball(`tarball_url`)은 최상위에 `owner-repo-hash` 형태의 디렉토리를 포함합니다. 현재 `ppm`은 이 구조를 지원하기 위해 압축 해제 후 내부 디렉토리를 탐색하거나, 사용자가 실행 파일 경로를 명시할 수 있는 기능을 향후 업데이트할 예정입니다. 지금은 레포지토리 이름과 실행 파일 이름이 같은지 확인해 주세요.

## 3. 플랫폼 및 아키텍처별 바이너리 배포 (권장)

`ppm`은 사용자의 OS와 CPU 아키텍처에 맞는 최적의 바이너리를 자동으로 선택합니다. GitHub 릴리스의 **Assets** 섹션에 플랫폼별로 빌드된 파일을 업로드할 때 아래 명명 규칙을 따르는 것이 좋습니다.

### 권장 파일명 패턴

`{바이너리이름}_{운영체제}_{아키텍처}` (예: `my-tool_darwin_arm64`)

| 대상 플랫폼 | 권장 키워드 (파일명에 포함) | 비고 |
| :--- | :--- | :--- |
| **macOS (Apple Silicon)** | `darwin`, `macos` **+** `arm64`, `aarch64` | 예: `my-tool_darwin_arm64` |
| **macOS (Intel)** | `darwin`, `macos` **+** `amd64`, `x86_64` | 예: `my-tool_darwin_amd64` |
| **Linux (x64)** | `linux` **+** `amd64`, `x86_64` | 예: `my-tool_linux_amd64` |
| **Windows (x64)** | `windows` **+** `amd64`, `x86_64` | `.exe` 확장자 필요 |

### 배포 방식 선택

1. **단일 바이너리 (Bare Binary)**: 압축하지 않고 실행 파일 자체를 업로드합니다. `ppm`은 이를 직접 다운로드하여 설치합니다.
2. **압축 파일 (Archive)**: `.tar.gz`, `.tgz`, `.zip` 형식으로 압축하여 업로드합니다. `ppm`은 압축을 푼 뒤 내부에서 바이너리를 찾아 설치합니다. (권장)

> **주의**: macOS 사용자가 늘어남에 따라 `arm64`와 `amd64` 바이너리를 명확히 구분하여 업로드해야 `exec format error`를 방지할 수 있습니다.

## 4. GitHub 릴리스 등록

`ppm`은 GitHub API의 `/releases/latest` 엔드포인트를 사용하여 패키지를 찾습니다.

1. GitHub 레포지토리의 **Releases** 섹션으로 이동합니다.
2. **Create a new release**를 클릭합니다.
3. **Tag version** (예: `v1.0.0`)을 입력합니다.
4. **Publish release**를 클릭하여 'Latest' 태그가 붙은 릴리스를 생성합니다.
    * `ppm`은 별도의 Asset 업로드 없이도 GitHub이 생성하는 소스 코드 tarball을 기본적으로 다운로드합니다.

## 4. 상세 메타데이터 등록 (`ppm.json`)

`ppm info` 명령어를 통해 패키지의 상세 정보를 표시하려면 레포지토리 루트에 `ppm.json` 파일을 추가하세요.

```json
{
  "description": "이 프로젝트에 대한 짧은 설명",
  "author": "작성자 이름",
  "homepage": "홈페이지 또는 문서 URL",
  "bin_name": "my-tool"
}
```

* **설명 (description)**: `ppm info` 명령어 시 표시될 패키지의 핵심 기능 요약
* **저자 (author)**: 개발자 또는 조직명
* **홈페이지 (homepage)**: 더 자세한 정보를 확인할 수 있는 링크
* **실행 파일 이름 (bin_name)**: (선택 사항) 레포지토리 이름과 실제 실행 파일 이름이 다를 경우 명시합니다. (예: 레포지토리는 `package_manager`지만 실행 파일은 `ppm`인 경우)

예시 파일은 이 레포지토리의 `ppm.json.example`을 참고하세요.

## 5. 프라이빗 레포지토리 설정

패키지가 프라이빗 레포지토리에 있다면, 설치하려는 사용자는 다음 설정을 마쳐야 합니다.

1. **Fine-grained Personal Access Token (PAT)** 생성:
    * `Contents: Read` 권한이 필요합니다.
    * `Metadata: Read` 권한이 필요합니다.
2. `ppm init` 실행 시 해당 토큰을 입력합니다.

## 5. 설치 테스트

패키지 등록이 완료되면 다음 명령어로 설치를 테스트할 수 있습니다.

```bash
ppm install <owner>/<repo>
```

예를 들어, `my-org/my-tool` 레포지토리라면:

```bash
ppm install my-org/my-tool
```

설치가 완료되면 `which my-tool` 명령어로 정상적으로 경로가 잡혔는지 확인하세요.
