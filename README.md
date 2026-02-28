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

### 자동 설치 (추천)

터미널에서 아래 명령어를 실행하여 현재 환경에 맞는 최신 버전을 자동으로 설치할 수 있습니다.

```bash
curl -fsSL https://raw.githubusercontent.com/wkqco33/package_manager/main/install.sh | sh
```

*(주의: `wkqco33/package_manager` 부분을 실제 GitHub 레포지토리 주소로 변경하여 사용하세요.)*

### 로컬 빌드

`Makefile`을 사용하여 직접 빌드할 수 있습니다.

```bash
# 로컬 빌드
make build

# 모든 플랫폼용 바이너리 빌드 (Linux, macOS, Windows)
make build-all
```

## 삭제 방법

터미널에서 아래 명령어를 실행하여 설치된 바이너리와 모든 설정, 패키지 데이터를 시스템에서 완전히 제거할 수 있습니다.

```bash
curl -fsSL https://raw.githubusercontent.com/wkqco33/package_manager/main/uninstall.sh | sh
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

프라이빗 레포지토리에서 현재 시스템에 맞는 바이너리를 설치합니다.

```bash
ppm install wkqco33/package_manager1 wkqco33/package_manager2
```

### 3. 패키지 목록 및 정보 확인

설치된 패키지의 버전과 상세 정보를 확인합니다.

```bash
ppm list   # 설치된 패키지 및 버전 확인
ppm info wkqco33/package_managersitory  # 원격 레지스트리의 패키지 상세 정보 확인
```

### 4. 패키지 업데이트

설치된 패키지를 최신 버전으로 업데이트합니다. 인자가 없으면 모든 패키지를 업데이트합니다.

```bash
ppm update          # 모든 패키지 업데이트
ppm update wkqco33/package_manager  # 특정 패키지만 업데이트
```

### 5. 패키지 삭제

설치된 패키지와 바이너리를 제거합니다.

```bash
ppm uninstall wkqco33/package_manager1 repo2
```

## 설정 상세 (`config.yaml`)

- `registry_url`: 레지스트리 API URL (기본값: `https://api.github.com`)
- `auth_token`: GitHub Personal Access Token (PAT)
- `install_path`: 바이너리가 설치될 경로 (기본값: `~/.local/bin` 또는 Windows 사용자 홈 `.local\bin`)
