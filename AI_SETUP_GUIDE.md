# AI Setup Guide: Making Projects `ppm` Compatible

이 문서는 다른 프로젝트의 개발을 돕는 AI 어시스턴트가 해당 프로젝트를 `ppm` (Private Package Manager) 호환 패키지로 전환하기 위한 가이드라인입니다.

## 🤖 AI 어시스턴트 실행 지침

다른 프로젝트에서 이 가이드를 발견하면, 다음 단계를 순서대로 수행하여 `ppm` 설치가 가능하도록 세팅하십시오.

### 1단계: 프로젝트 분석 (Analysis)

- **바이너리 이름**: 프로젝트의 레포지토리 이름을 확인하십시오. `ppm`은 기본적으로 `~/.local/bin/<repo_name>` 형태로 링크를 생성하므로, 빌드 결과물(바이너리)의 이름이 레포지토리 이름과 일치해야 합니다.
- **빌드 시스템**: Go, Rust, C++ 등 바이너리를 생성할 수 있는 빌드 스크립트(Makefile 등)가 있는지 확인하십시오.

### 2단계: 메타데이터 생성 (`ppm.json`)

프로젝트 루트에 `ppm.json` 파일을 생성하여 패키지 정보를 기술하십시오.

```json
{
  "description": "프로젝트에 대한 짧고 명확한 설명",
  "author": "작성자 또는 조직명",
  "homepage": "프로젝트 홈페이지 URL"
}
```

### 3단계: GitHub Actions를 통한 릴리스 자동화

`ppm`은 GitHub의 `Latest Release`를 기반으로 작동합니다. 태그 푸시 시 자동으로 바이너리를 포함한 릴리스를 생성하는 워크플로우를 `.github/workflows/release.yml`에 추가하십시오.

**추천 템플릿:**

```yaml
name: Release for ppm

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      
      - name: Build Binary
        run: |
          # REPO_NAME을 프로젝트 이름으로 변경하거나 자동 추출
          REPO_NAME=$(basename ${{ github.repository }})
          go build -ldflags="-s -w" -o "$REPO_NAME" main.go
          
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          generate_release_notes: true
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 4단계: 사용자 안내

설정이 완료되면 사용자에게 다음 사항을 알리십시오.

1. `ppm.json`이 생성되었으므로 `ppm info <owner>/<repo>`로 정보를 확인할 수 있습니다.
2. `git tag v1.0.0` 후 푸시하면 `ppm install <owner>/<repo>`로 즉시 설치가 가능합니다.
3. (프라이빗 레포지토리인 경우) 사용자의 PAT에 `Contents: Read` 권한이 있는지 확인을 요청하십시오.

---

## 💡 사용자용 AI 프롬프트 예시 (Copy & Paste)

다른 프로젝트의 AI에게 다음과 같이 명령하면 이 가이드를 기반으로 자동 세팅을 수행합니다.

> "이 프로젝트를 `ppm` 패키지 매니저와 호환되도록 세팅해줘. `ppm.json` 메타데이터를 작성하고, 태그 푸시 시 자동으로 빌드해서 릴리스를 생성하는 GitHub Actions 워크플로우를 추가해줘. 바이너리 이름은 레포지토리 이름과 일치시켜야 해."
