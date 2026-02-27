# ppm (Private Package Manager)

`ppm`은 GitHub 프라이빗 레포지토리에서 바이너리 패키지를 빠르고 효율적으로 관리하기 위한 성능 중심의 패키지 매니저입니다.

## 주요 특징

- **고성능 스트리밍**: 다운로드와 동시에 압축을 해제하여 임시 파일 생성을 방지하고 속도를 극대화합니다.
- **다중 패키지 병렬 설치**: 여러 개의 패키지를 동시에 설치(기본 최대 5개)하여 전체 설치 시간을 단축합니다.
- **풍부한 메타데이터**: `ppm.json`을 통해 패키지 설명, 저자, 홈페이지 정보를 관리하고 `info` 명령어로 조회할 수 있습니다.
- **최소한의 추상화**: 성능과 직관성을 최우선으로 고려하여 Go로 구현되었습니다.

## 설치 방법

`Makefile`을 사용하여 간편하게 빌드 및 설치할 수 있습니다.

```bash
make build
make install
```

## 사용 방법

### 1. 초기화 (Initialize)

설정 디렉토리(`~/.config/ppm`)를 생성하고 기본 설정 파일을 만듭니다.

```bash
ppm init
```

### 2. 패키지 설치

프라이빗 레포지토리에서 패키지를 설치합니다. 여러 패키지를 한 번에 병렬로 설치할 수 있습니다.

```bash
ppm install owner/repo1 owner/repo2
```

### 3. 패키지 정보 확인

원격 저장소의 메타데이터(`ppm.json`)와 릴리스 정보를 조회합니다.

```bash
ppm info owner/repository
```

### 4. 설치된 패키지 목록 확인

현재 시스템에 설치된 패키지들을 확인합니다.

```bash
ppm list
```

## 설정 상세 (`config.yaml`)

- `registry_url`: 레지스트리 API URL (기본값: `https://api.github.com`)
- `auth_token`: GitHub Personal Access Token (PAT)
- `install_path`: 바이너리 링크가 설치될 경로 (기본값: `~/.local/bin`)
