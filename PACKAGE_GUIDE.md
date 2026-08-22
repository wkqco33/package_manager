# 패키지 등록 및 배포 가이드 (PACKAGE_GUIDE.md)

`ppm` (Private Package Manager)을 통해 배포 및 설치 가능한 패키지를 구성하고 GitHub 릴리스에 등록하는 방법을 설명합니다.

---

## 1. 개요 및 배포 원칙

`ppm`은 GitHub 릴리스의 사전 빌드된 바이너리 아카이브를 사용자의 OS와 CPU 아키텍처에 맞춰 자동으로 탐색하고 스트리밍 방식으로 설치합니다.

* **바이너리 우선 배포**: 보안 및 배포 속도를 위해 사전 빌드된 아카이브(`.tar.gz`, `.tgz`, `.zip`) 배포를 권장합니다. 소스 빌드 폴백(`--from-source`)은 사용자가 명시적으로 허용한 경우에만 실행됩니다.
* **무결성 검증 지원**: SHA-256 체크섬(`.sha256`, `.sha256sum`) 및 Ed25519 서명(`.sig`, `.asc`) 파일을 함께 제공하여 패키지의 위변조를 방지할 수 있습니다.
* **메타데이터 및 의존성 관리**: 루트의 `ppm.json` 매니페스트를 통해 실행 파일 이름, 설명, 패키지 간 의존성 관계를 선언합니다.

---

## 2. 레포지토리 및 실행 파일 규칙

### 바이너리 이름 결정 방식
1. **`ppm.json`에 `bin_name`이 명시된 경우**: 해당 이름을 최종 실행 파일명으로 사용합니다.
   * 예: 레포지토리 이름이 `package_manager`이지만 실행 파일명은 `ppm`인 경우
2. **`bin_name`이 없는 경우**: GitHub 레포지토리 이름을 기본 바이너리 이름으로 간주합니다.
   * 예: `my-org/my-tool` -> 실행 파일명 `my-tool`

> **참고**: Windows 환경에서는 자동으로 `.exe` 확장자가 추가됩니다.

### 아카이브 내부 실행 파일 탐색 순서
`ppm`은 다운로드한 아카이브 압축을 해제한 뒤, 최대 3단계 깊이까지 검색하여 아래 우선순위에 따라 실행 파일을 선택하고 `~/.local/bin` (또는 지정된 `install_path`)에 링크합니다.

1. 파일명이 `bin_name`과 **정확히 일치**하는 파일
2. 파일명에 `bin_name`이 **포함**된 파일 (대소문자 무시)
3. 디렉토리 내에 유일한 실행 파일(Unix `x` 권한 보유 파일, Windows `.exe` 파일)이 단 1개 존재하는 경우
4. `bin_name`이 `ppm` 또는 `package_manager`인 경우 `ppm`/`ppm.exe` 파일 우선 선택

---

## 3. 플랫폼 및 아키텍처별 자산(Assets) 배포 규칙

GitHub Release의 Assets 섹션에 업로드할 때 아래 명명 규칙을 준수해야 `ppm`이 자동으로 현재 시스템에 적합한 파일을 감지합니다.

### 권장 파일명 패턴
```text
{bin_name}_{os}_{arch}.{확장자}
```
* 예: `my-tool_linux_amd64.tar.gz`, `my-tool_darwin_arm64.tar.gz`, `my-tool_windows_amd64.zip`

### 플랫폼별 키워드 매칭 기준

| 대상 플랫폼 | OS 키워드 | Arch 키워드 | 비고 |
| :--- | :--- | :--- | :--- |
| **Linux (x86_64)** | `linux` | `amd64`, `x86_64`, `64bit` | `.tar.gz` 또는 `.tgz` 권장 |
| **Linux (ARM64)** | `linux` | `arm64`, `aarch64` | `.tar.gz` 또는 `.tgz` 권장 |
| **macOS (Apple Silicon)** | `darwin`, `macos`, `apple-darwin` | `arm64`, `aarch64` | `.tar.gz` 또는 `.tgz` 권장 |
| **macOS (Intel)** | `darwin`, `macos`, `apple-darwin` | `amd64`, `x86_64`, `64bit` | `.tar.gz` 또는 `.tgz` 권장 |
| **Windows (x86_64)** | `windows` | `amd64`, `x86_64`, `64bit` | `.zip` 권장, 바이너리는 `.exe` |

### 아키텍처 충돌 방지 규칙
`ppm`은 잘못된 아키텍처의 바이너리가 설치되어 발생하는 `exec format error`를 방지하기 위해 상호 배제 필터링을 적용합니다.
* `amd64` 환경 탐색 시: 파일명에 `arm64`, `aarch64`, `armv`가 포함된 파일은 자동 제외
* `arm64` 환경 탐색 시: 파일명에 `amd64`, `x86_64`, `x86`, `i386`이 포함된 파일은 자동 제외

### 배포 포맷 우선순위
1. **아카이브 포맷 (`.tar.gz`, `.tgz`, `.zip`) (권장)**: 스트리밍 압축 해제를 통해 고속으로 설치됩니다.
2. **단일 바이너리 (Bare Binary)**: 압축되지 않은 단일 실행 파일 (`my-tool_linux_amd64`). 아카이브 파일이 없을 경우 폴백으로 선택됩니다.

---

## 4. 패키지 매니페스트 (`ppm.json`)

패키지 메타데이터와 의존성을 선언하려면 레포지토리 루트에 `ppm.json`을 추가합니다.

### `ppm.json` 구조
```json
{
  "description": "패키지에 대한 핵심 기능 요약 설명",
  "author": "작성자 또는 팀/조직명",
  "homepage": "https://github.com/my-org/my-tool",
  "bin_name": "my-tool",
  "dependencies": {
    "my-org/core-lib": ">=1.2.0",
    "my-org/helper-tool": "^0.5.0"
  }
}
```

### 필드 상세 설명
* **`bin_name` (필수)**: 설치 후 생성될 실행 파일의 이름입니다. 경로 구분자(`/`, `\`)나 특수 경로(`.`, `..`)는 사용할 수 없습니다.
* **`description`**: `ppm info` 명령어 실행 시 사용자에게 표시되는 요약 설명입니다.
* **`author`**: 패키지 작성자 또는 유지보수자 정보입니다.
* **`homepage`**: 패키지 관련 문서 또는 웹사이트 URL입니다.
* **`dependencies`**: 패키지 동작에 필요한 다른 `ppm` 패키지들의 의존성 제약 목록입니다 (`owner/repo: constraint`).

### 매니페스트 초기화 (`ppm package init`)
새 프로젝트 디렉토리에서 매니페스트를 대화형/플래그로 손쉽게 생성할 수 있습니다.
```bash
# 현재 디렉토리에 기본 ppm.json 생성 (바이너리명은 디렉토리명으로 자동 설정)
ppm package init

# 특정 프로젝트 디렉토리에 플래그로 매니페스트 생성
ppm package init ./my-project --bin-name my-tool --description "핵심 도구" --author "팀명" --homepage "https://github.com/my-org/my-tool"
```

### 매니페스트 유효성 검증
로컬에서 작성한 `ppm.json`의 유효성을 검사할 수 있습니다.
```bash
ppm package validate [ppm.json 경로]
# 또는
ppm manifest validate [ppm.json 경로]
```

---

## 5. 보안 및 무결성 검증 (체크섬 & 디지털 서명)

배포 파일의 무결성과 출처를 검증할 수 있도록 릴리스 에셋과 동일한 경로에 검증 파일을 함께 업로드하는 것을 권장합니다.

### 1) SHA-256 체크섬 (`.sha256`, `.sha256sum`)
각 아카이브 파일에 대한 64자리 SHA-256 해시값을 담은 텍스트 파일을 업로드합니다.
* **파일 이름**: `{아카이브파일명}.sha256` (예: `my-tool_linux_amd64.tar.gz.sha256`)
* **파일 내용**:
  ```text
  e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  my-tool_linux_amd64.tar.gz
  ```
  *(해시값만 포함되어 있거나 `해시값  파일명` 형식 모두 지원)*

사용자가 `config.yaml`에 `require_checksum: true`를 설정한 경우, 체크섬이 일치하지 않거나 누락되면 설치가 차단됩니다.

### 2) Ed25519 디지털 서명 (`.sig`, `.asc`)
패키지 위변조를 방지하기 위해 Ed25519 개인키로 생성한 Base64 인코딩 서명 파일을 업로드합니다.
* **파일 이름**: `{아카이브파일명}.sig` (예: `my-tool_linux_amd64.tar.gz.sig`)
* 사용자가 `require_signature: true` 및 `signature_public_key`를 설정한 경우 공개키를 통해 서명을 자동 검증합니다.

---

## 6. GitHub 릴리스 등록 워크플로우

### 릴리스 생성 단계
1. 패키지 소스 코드의 버전을 태그로 생성합니다 (예: `v1.0.0`).
2. GitHub 레포지토리의 **Releases** -> **Draft a new release**로 이동합니다.
3. 태그를 선택하고 Release Title 및 Release Notes(변경 사항)를 입력합니다.
4. **Attach binaries by dropping them here or selecting them** 영역에 빌드된 자산들을 업로드합니다.

### 업로드 파일 예시 (v1.0.0 기준)
```text
my-tool_linux_amd64.tar.gz
my-tool_linux_amd64.tar.gz.sha256
my-tool_linux_arm64.tar.gz
my-tool_linux_arm64.tar.gz.sha256
my-tool_darwin_amd64.tar.gz
my-tool_darwin_amd64.tar.gz.sha256
my-tool_darwin_arm64.tar.gz
my-tool_darwin_arm64.tar.gz.sha256
my-tool_windows_amd64.zip
my-tool_windows_amd64.zip.sha256
```

5. **Publish release**를 클릭하여 릴리스를 배포합니다.

---

## 7. 프라이빗 레포지토리 권한 설정

패키지가 프라이빗 레포지토리에 위치한 경우, 사용자는 최소 권한의 GitHub Personal Access Token (Fine-grained PAT)을 설정해야 합니다.

### 필수 토큰 권한 (Fine-grained PAT)
* **Repository access**: 패키지 저장소 선택
* **Permissions**:
  * `Contents: Read` (릴리스 자산 및 소스 아카이브 다운로드)
  * `Metadata: Read` (기본 읽기 권한)

사용자는 아래 방식으로 인증을 구성합니다:
```bash
# 대화형 초기화
ppm init

# 또는 커맨드로 직접 설정
ppm config set auth_token <PAT_TOKEN>

# 또는 환경 변수 주입 (CI/CD)
export PPM_AUTH_TOKEN="<PAT_TOKEN>"
```

---

## 8. 배포 후 테스트 및 검증 명령어

패키지를 릴리스한 후 아래 명령어를 통해 배포 상태가 올바른지 단계별로 검증할 수 있습니다.

```bash
# 1. 원격 레지스트리의 패키지 메타데이터 및 릴리스 확인
ppm info <owner>/<repo>
ppm info <owner>/<repo> --json

# 2. 설치 시뮬레이션 (실제 파일 다운로드/설치 없이 계획만 확인)
ppm install --dry-run <owner>/<repo>

# 3. 실제 설치 수행
ppm install <owner>/<repo>

# 4. 설치된 패키지 및 바이너리 검증
ppm verify
ppm doctor

# 5. 설치된 패키지 실행 확인
which <bin_name>
<bin_name> --version
```
