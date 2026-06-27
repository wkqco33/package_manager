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

## 설치 및 빌드

[Task](https://taskfile.dev)(go-task)를 사용하여 모든 OS(Windows/macOS/Linux)에서 편리하게 빌드 및 설치할 수 있습니다.

먼저 Task를 설치합니다. (자세한 방법: https://taskfile.dev/installation)

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

프라이빗 레포지토리에서 현재 시스템에 맞는 바이너리를 설치합니다. GitHub Release가 없으면 최신 태그로, 바이너리 Asset이 없으면 소스 아카이브로 자동 폴백합니다.

```bash
ppm install owner/repo1 owner/repo2
```

### 3. 패키지 목록 및 정보 확인

설치된 패키지의 버전과 상세 정보를 확인합니다.

```bash
ppm list   # 설치된 패키지 및 버전 확인
ppm info owner/repo  # 원격 레지스트리의 패키지 상세 정보 확인
```

### 4. 패키지 업데이트

설치된 패키지를 최신 버전으로 업데이트합니다. 인자가 없으면 모든 패키지를 업데이트합니다.

```bash
ppm update          # 모든 패키지 업데이트
ppm update owner/repo  # 특정 패키지만 업데이트
```

### 5. 패키지 삭제

설치된 패키지와 바이너리를 제거합니다.

```bash
ppm uninstall owner/repo1 repo2
```

## 설정 상세 (`config.yaml`)

- `registry_url`: 레지스트리 API URL (기본값: `https://api.github.com`). 커스텀 레지스트리에서 패키지를 찾지 못하면 GitHub 공개 API로 한 번 더 조회합니다.
- `auth_token`: GitHub Personal Access Token (PAT)
- `install_path`: 바이너리가 설치될 경로 (기본값: `~/.local/bin` 또는 Windows 사용자 홈 `.local\bin`)
