# ppm (Private Package Manager)

`ppm`은 GitHub 프라이빗 레포지토리에서 바이너리 패키지를 빠르고 효율적으로 관리하기 위한 성능 중심의 패키지 매니저입니다.

## 주요 특징

- **멀티 플랫폼 지원**: Linux, macOS, Windows를 모두 지원하며 각 OS의 표준 경로를 따릅니다.
- **자동 자산 선택**: 현재 OS와 아키텍처(amd64, arm64 등)에 맞는 릴리스 자산을 자동으로 찾아 다운로드합니다.
- **다양한 아카이브 지원**: `.tar.gz`, `.tgz` 및 Windows 자산용 `.zip` 형식을 지원합니다.
- **고성능 스트리밍**: 다운로드와 동시에 압축을 해제하여 속도를 극대화하고 임시 파일 생성을 방지합니다.
- **자동 업데이트 및 버전 관리**: 설치된 패키지들의 메타데이터를 관리하여 최신 버전으로 한 번에 업데이트할 수 있습니다.
- **다중 패키지 병렬 설치**: 여러 개의 패키지를 동시에 설치하여 작업 시간을 단축합니다.
- **최소한의 추상화**: 성능과 직관성을 최우선으로 고려하여 Go로 구현되었습니다.

## 설치

### curl 설치 스크립트 (Linux/macOS)

지원되는 Linux 또는 macOS에서는 저장소에 포함된 설치 스크립트를 사용할 수 있습니다. 스크립트는 운영체제와 아키텍처를 감지하고, Release asset과 SHA-256 checksum을 다운로드한 뒤 검증을 통과한 바이너리만 `~/.local/bin/ppm`에 설치합니다.

```bash
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  https://raw.githubusercontent.com/wkqco33/package_manager/master/install.sh \
  -o /tmp/ppm-install.sh
sh /tmp/ppm-install.sh
rm /tmp/ppm-install.sh
```

스크립트를 실행하기 전에 내용을 검토하는 것을 권장합니다. 특정 버전을 설치하거나 경로를 변경할 수도 있습니다.

```bash
# 위의 다운로드 명령을 다시 실행한 뒤
PPM_VERSION=v1.0.0 PPM_INSTALL_DIR="$HOME/.local/bin" sh /tmp/ppm-install.sh
```

Windows 사용자는 [Releases](https://github.com/wkqco33/package_manager/releases)에서 `.zip` asset을 직접 다운로드하세요.

### GitHub Release에서 직접 설치

[Releases](https://github.com/wkqco33/package_manager/releases)에서 운영체제와 CPU 아키텍처에 맞는 아카이브를 다운로드하세요.

```bash
# Linux amd64 예시
curl -LO https://github.com/wkqco33/package_manager/releases/latest/download/ppm_linux_amd64.tar.gz
curl -LO https://github.com/wkqco33/package_manager/releases/latest/download/ppm_linux_amd64.tar.gz.sha256
sha256sum -c ppm_linux_amd64.tar.gz.sha256
tar -xzf ppm_linux_amd64.tar.gz
install -Dm755 ppm ~/.local/bin/ppm
```

Windows와 macOS 사용자는 Releases 페이지에서 해당 플랫폼의 파일을 선택하고, 함께 제공되는 `.sha256` 파일로 다운로드 무결성을 확인하세요.

### 소스에서 빌드

[Task](https://taskfile.dev)(go-task)를 사용하여 모든 OS(Windows/macOS/Linux)에서 빌드할 수 있습니다.

먼저 Task를 설치합니다. (자세한 방법: <https://taskfile.dev/installation>)

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

### 빌드 및 설치

```bash
# 최적화된 바이너리 빌드 후 로컬 경로(~/.local/bin)에 설치
task install

# 로컬 빌드만 수행
task build

# 모든 플랫폼용 바이너리 빌드 (Linux, macOS, Windows)
task build-all
```

## 삭제 방법

설치된 바이너리를 시스템에서 제거하려면 다음 명령을 실행합니다.

```bash
# 로컬 경로에서 바이너리 삭제
task uninstall
```

## 사용 방법

### 1. 초기화 (Initialize)

플랫폼별 표준 경로에 설정 파일과 디렉토리를 생성합니다.

```bash
ppm init
```

- **Linux**: `~/.config/ppm`
- **macOS**: `~/Library/Application Support/ppm`
- **Windows**: `%AppData%\ppm`

### 2. 패키지 설치

프라이빗 레포지토리에서 현재 시스템에 맞는 바이너리를 설치합니다. GitHub Release가 없으면 최신 태그로 조회하며, 기본적으로는 사전 빌드된 바이너리 asset만 설치합니다.

소스 아카이브를 로컬에서 빌드해야 하는 신뢰할 수 있는 저장소는 명시적으로 다음 옵션을 사용하세요.

```bash
ppm install --from-source owner/repo
```

`--from-source`는 다운로드한 소스에서 로컬 `go build`를 실행하므로 신뢰할 수 있는 저장소에만 사용해야 합니다.

```bash
ppm install owner/repo1 owner/repo2
```

### 3. 패키지 목록 및 정보 확인

설치된 패키지의 버전과 상세 정보를 확인합니다.

```bash
ppm list   # 설치된 패키지 및 버전 확인
ppm list --json   # 자동화용 JSON 출력
ppm info owner/repo  # 원격 레지스트리의 패키지 상세 정보 확인
ppm info owner/repo --json
ppm doctor         # 설정·경로·설치 상태 진단
ppm verify         # 설치된 패키지와 바이너리 검증
ppm completion bash # Bash 자동완성 스크립트 생성
```

### 4. 패키지 업데이트

설치된 패키지를 최신 버전으로 업데이트합니다. 인자가 없으면 모든 패키지를 업데이트합니다.

```bash
ppm update          # 모든 패키지 업데이트
ppm update owner/repo  # 특정 패키지만 업데이트
ppm update --check      # 업데이트 없이 변경 가능 버전 확인
ppm outdated            # 업데이트 가능한 패키지 목록
ppm changelog owner/repo # 최신 릴리스 노트 확인
ppm lock owner/repo    # 버전과 의존성을 ppm.lock에 기록
ppm install --locked owner/repo  # lockfile 기준으로 설치
ppm install --locked --offline owner/repo  # 캐시에서만 설치
ppm install --atomic owner/repo1 owner/repo2  # 전체 성공 시에만 변경
ppm manifest validate   # ppm.json 검증
```

### 5. 패키지 삭제

설치된 패키지와 바이너리를 제거합니다.

```bash
ppm uninstall owner/repo1 repo2

# 설치 계획만 확인 (실제 변경 없음)
ppm install --dry-run owner/repo

# ppm 자신을 최신 GitHub Release로 업데이트
ppm self-update

# 다운로드 캐시 관리
ppm cache list
ppm cache clean
```

## 설정 상세 (`config.yaml`)

설정 파일은 `ppm init`으로 생성할 수 있으며, 커맨드로 조회하거나 변경할 수도 있습니다.

```bash
# 현재 설정 확인 (auth_token은 보안을 위해 마스킹되어 출력됩니다)
ppm config show

# 설정 변경
ppm config set registry_url https://api.github.com
ppm config set auth_token <your-personal-access-token>
ppm config set install_path ~/.local/bin
```

`ppm config set`은 설정 파일이 아직 없으면 기본 설정을 생성한 뒤 지정한 값을 저장합니다.

- `registry_url`: 기본 레지스트리 API URL (기본값: `https://api.github.com`).
- `registries`: 기본 레지스트리 실패 시 순서대로 시도할 mirror API URL 목록입니다. 마지막에는 GitHub 공개 API가 자동으로 시도됩니다.
- `auth_token`: GitHub Personal Access Token (PAT)
- `install_path`: 바이너리가 설치될 경로 (기본값: `~/.local/bin` 또는 Windows 사용자 홈 `.local\bin`)

설정 파일에는 GitHub Personal Access Token이 저장되므로 파일 권한을 다른 사용자에게 공개하지 마세요. `ppm config show`는 토큰을 마스킹해서 출력합니다.

## 개발

```bash
task test       # 전체 테스트
task test-race  # race detector 포함 테스트
task lint       # go vet
task coverage   # 패키지별 커버리지
task build      # 로컬 빌드
```

## 라이선스

이 프로젝트는 [MIT License](LICENSE)로 배포됩니다.
