package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/manifest"
)

func TestPackageCommandShowsGuide(t *testing.T) {
	// PackageGuide가 설정되어 있을 때 출력 확인
	origGuide := PackageGuide
	defer func() { PackageGuide = origGuide }()

	testGuideContent := "# 테스트 패키지 가이드 내용입니다."
	PackageGuide = testGuideContent

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe error = %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	execErr := ExecuteArgs([]string{"package"})

	w.Close()
	os.Stdout = origStdout

	if execErr != nil {
		t.Fatalf("ExecuteArgs(package) error = %v", execErr)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, testGuideContent) {
		t.Errorf("output = %q, want contains %q", output, testGuideContent)
	}
}

func TestPackageInitCommandCreatesPpmJson(t *testing.T) {
	tempDir := t.TempDir()
	targetProject := filepath.Join(tempDir, "sample-project")
	if err := os.MkdirAll(targetProject, 0755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}

	err := ExecuteArgs([]string{
		"package", "init", targetProject,
		"--bin-name", "sample-bin",
		"--description", "샘플 프로젝트 설명",
		"--author", "개발자",
		"--homepage", "https://github.com/org/sample",
	})
	if err != nil {
		t.Fatalf("package init error = %v", err)
	}

	manifestPath := filepath.Join(targetProject, "ppm.json")
	m, err := manifest.Load(manifestPath)
	if err != nil {
		t.Fatalf("failed to load created ppm.json: %v", err)
	}

	if m.BinName != "sample-bin" {
		t.Errorf("BinName = %q, want 'sample-bin'", m.BinName)
	}
	if m.Description != "샘플 프로젝트 설명" {
		t.Errorf("Description = %q, want '샘플 프로젝트 설명'", m.Description)
	}
	if m.Author != "개발자" {
		t.Errorf("Author = %q, want '개발자'", m.Author)
	}
	if m.Homepage != "https://github.com/org/sample" {
		t.Errorf("Homepage = %q, want 'https://github.com/org/sample'", m.Homepage)
	}
}

func TestPackageInitCommandRejectsExistingWithoutForce(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "ppm.json")
	if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	err := ExecuteArgs([]string{"package", "init", tempDir})
	if err == nil {
		t.Fatal("expected error when ppm.json already exists without --force")
	}

	// With --force
	err = ExecuteArgs([]string{"package", "init", tempDir, "--force", "--bin-name", "overwritten-bin"})
	if err != nil {
		t.Fatalf("expected success with --force, got: %v", err)
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		t.Fatalf("failed to load overwritten ppm.json: %v", err)
	}
	if m.BinName != "overwritten-bin" {
		t.Errorf("BinName = %q, want 'overwritten-bin'", m.BinName)
	}
}

func TestPackageValidateCommand(t *testing.T) {
	tempDir := t.TempDir()
	validPath := filepath.Join(tempDir, "valid.json")
	invalidPath := filepath.Join(tempDir, "invalid.json")

	validM := manifest.Manifest{
		BinName:     "tool",
		Description: "valid",
	}
	if err := validM.Save(validPath); err != nil {
		t.Fatalf("Save valid manifest error = %v", err)
	}

	if err := os.WriteFile(invalidPath, []byte(`{"description": "no bin name"}`), 0644); err != nil {
		t.Fatalf("WriteFile invalid manifest error = %v", err)
	}

	if err := ExecuteArgs([]string{"package", "validate", validPath}); err != nil {
		t.Errorf("expected package validate on valid manifest to succeed, got: %v", err)
	}

	if err := ExecuteArgs([]string{"package", "validate", invalidPath}); err == nil {
		t.Error("expected package validate on invalid manifest to fail, got nil")
	}
}
