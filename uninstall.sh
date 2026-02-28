#!/bin/sh
set -e

# ppm 삭제 스크립트
# 사용법: curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/uninstall.sh | sh

INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="ppm"

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

printf "${BLUE}==>${NC} ppm 삭제를 시작합니다...
"

# OS 감지 및 설정 경로 결정
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    linux*)
        CONFIG_DIR="${HOME}/.config/ppm"
        FINAL_BINARY="ppm"
        ;;
    darwin*)
        CONFIG_DIR="${HOME}/Library/Application Support/ppm"
        FINAL_BINARY="ppm"
        ;;
    msys*|cygwin*|mingw*)
        # Windows 환경 (Git Bash 등에서 실행 시)
        if [ -n "$APPDATA" ]; then
            CONFIG_DIR="$APPDATA/ppm"
        else
            CONFIG_DIR="${HOME}/AppData/Roaming/ppm"
        fi
        FINAL_BINARY="ppm.exe"
        ;;
    *)
        printf "${RED}Error:${NC} 지원하지 않는 OS입니다: ${OS}
"
        exit 1
        ;;
esac

# 1. 바이너리 삭제
if [ -f "${INSTALL_DIR}/${FINAL_BINARY}" ]; then
    printf "${BLUE}==>${NC} 바이너리 삭제 중: ${INSTALL_DIR}/${FINAL_BINARY}
"
    rm -f "${INSTALL_DIR}/${FINAL_BINARY}"
else
    printf "${BLUE}==>${NC} 바이너리를 찾을 수 없습니다. (이미 삭제되었거나 다른 경로에 있음)
"
fi

# 2. 설정 및 데이터 디렉토리 삭제
if [ -d "$CONFIG_DIR" ]; then
    printf "${BLUE}==>${NC} 설정 및 패키지 데이터 삭제 중: ${CONFIG_DIR}
"
    rm -rf "$CONFIG_DIR"
else
    printf "${BLUE}==>${NC} 설정 디렉토리를 찾을 수 없습니다.
"
fi

printf "${GREEN}==>${NC} ppm이 시스템에서 완전히 제거되었습니다.
"
printf "참고: PATH 설정은 자동으로 제거되지 않으니 필요한 경우 쉘 설정 파일(~/.bashrc, ~/.zshrc 등)에서 수동으로 정리하세요.
"
