# ppm (Private Package Manager) Makefile

# 변수 설정
BINARY_NAME=ppm
INSTALL_DIR=$(HOME)/.local/bin
GO_FILES=$(shell find . -name "*.go" -type f)

# 빌드 플래그 (성능 최적화 및 바이너리 크기 감소)
LDFLAGS=-ldflags="-s -w"

.PHONY: all build install uninstall clean test fmt lint help

all: build

## build: 최적화된 바이너리 생성 (-s -w 플래그 적용)
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) main.go

## lint: go vet를 사용한 정적 분석 실행
lint:
	@echo "Running lint (go vet)..."
	go vet ./...

## install: 빌드 후 시스템 경로($(INSTALL_DIR))에 설치
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installation complete. Please ensure $(INSTALL_DIR) is in your PATH."

## uninstall: 시스템 경로에서 바이너리 삭제
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstallation complete."

## clean: 빌드 결과물 및 임시 파일 삭제
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@go clean
	@echo "Cleanup complete."

## test: 모든 패키지의 테스트 실행
test:
	@echo "Running tests..."
	@go test -v ./...

## fmt: Go 코드 스타일 정렬
fmt:
	@echo "Formatting code..."
	@go fmt ./...

## help: 사용 가능한 명령어 목록 표시
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^##' Makefile | sed -e 's/## //' | column -t -s ':'
