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

## 3. GitHub 릴리스 등록

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
  "homepage": "홈페이지 또는 문서 URL"
}
```

*   **설명 (description)**: `ppm info` 명령어 시 표시될 패키지의 핵심 기능 요약
*   **저자 (author)**: 개발자 또는 조직명
*   **홈페이지 (homepage)**: 더 자세한 정보를 확인할 수 있는 링크

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
