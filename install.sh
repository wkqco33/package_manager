#!/bin/sh
set -e

# ppm 설치 스크립트
# 사용법: curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/install.sh | sh

GITHUB_REPO="OWNER/REPO" # 실제 배포 시 사용자/레포지토리로 수정 필요
INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="ppm"

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

printf "${BLUE}==>${NC} ppm 설치를 시작합니다...
"

# OS 감지
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    linux*)   OS='linux';;
    darwin*)  OS='darwin';;
    msys*|cygwin*|mingw*) OS='windows';;
    *)        printf "${RED}Error:${NC} 지원하지 않는 OS입니다: ${OS}
"; exit 1;;
esac

# 아키텍처 감지
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) ARCH='amd64';;
    arm64|aarch64) ARCH='arm64';;
    *)            printf "${RED}Error:${NC} 지원하지 않는 아키텍처입니다: ${ARCH}
"; exit 1;;
esac

# 바이너리 이름 결정
if [ "$OS" = "windows" ]; then
    TARGET_BINARY="ppm-${OS}-${ARCH}.exe"
    FINAL_BINARY="ppm.exe"
else
    TARGET_BINARY="ppm-${OS}-${ARCH}"
    FINAL_BINARY="ppm"
fi

# 최신 버전 태그 가져오기 (GitHub API 사용)
printf "${BLUE}==>${NC} 최신 버전을 확인 중...
"
LATEST_TAG=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    printf "${RED}Error:${NC} 최신 버전을 찾을 수 없습니다. 레포지토리 이름을 확인해주세요.
"
    exit 1
fi

DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/${TARGET_BINARY}"

# 설치 경로 생성
mkdir -p "${INSTALL_DIR}"

# 다운로드 및 설치
printf "${BLUE}==>${NC} ${LATEST_TAG} 버전을 다운로드 중: ${TARGET_BINARY}
"
if ! curl -L -f -o "${INSTALL_DIR}/${FINAL_BINARY}" "${DOWNLOAD_URL}"; then
    printf "${RED}Error:${NC} 다운로드에 실패했습니다. URL을 확인해주세요: ${DOWNLOAD_URL}
"
    exit 1
fi

chmod +x "${INSTALL_DIR}/${FINAL_BINARY}"

printf "${GREEN}==>${NC} ppm이 성공적으로 설치되었습니다!
"
printf "
설치 경로: ${INSTALL_DIR}/${FINAL_BINARY}
"

# PATH 확인 및 안내
case :$PATH: in
    *:"${INSTALL_DIR}":*) ;;
    *) 
        printf "${BLUE}주의:${NC} ${INSTALL_DIR}이 PATH에 포함되어 있지 않습니다.
"
        printf "아래 명령어를 쉘 설정 파일(~/.bashrc 또는 ~/.zshrc)에 추가하세요:
"
        printf "  export PATH="\$PATH:${INSTALL_DIR}"
"
        ;;
esac

printf "
설치를 완료하려면 'ppm init'을 실행하여 설정을 초기화하세요.
"
