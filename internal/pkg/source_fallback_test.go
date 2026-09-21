package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/wkqco33/package_manager/internal/apperr"
)

// 소스 tarball 폴백이 거부될 때 사용자가 인증 문제와 혼동하지 않도록
// 감지된 플랫폼과 사용 가능한 에셋 목록을 함께 보여주어야 합니다.
func TestInstallWithPackageOptionsRejectsSourceFallbackWithPlatformDiagnostics(t *testing.T) {
	setupTempHome(t)

	p := &Package{
		Name:           "owner/tool",
		Version:        "v0.2.0",
		Source:         "https://example.test/source.tar.gz",
		SourceFallback: true,
		AssetDiagnostics: &AssetDiagnostics{
			Platform:  "darwin/arm64",
			Available: []string{"tool_darwin_amd64.tar.gz", "tool_linux_amd64.tar.gz"},
		},
	}
	archiver := &MockArchiver{}

	err := InstallWithPackageOptions(p, &MockFetcher{pkg: p}, archiver, t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected source fallback to be rejected")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperr.CodeArchive {
		t.Fatalf("error = %v, want ARCHIVE_ERROR", err)
	}
	msg := err.Error()
	for _, want := range []string{"darwin/arm64", "tool_darwin_amd64.tar.gz", "tool_linux_amd64.tar.gz", "--from-source"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q must mention %q", msg, want)
		}
	}
	if archiver.extracted {
		t.Error("no archive should be extracted when the guard rejects the install")
	}
}

// 진단 정보가 없는 경우(예: ppm.lock 기반 설치)에도 실패 사유와 해결 방법은
// 그대로 안내되어야 합니다.
func TestInstallWithPackageOptionsRejectsSourceFallbackWithoutDiagnostics(t *testing.T) {
	setupTempHome(t)

	p := &Package{Name: "owner/tool", Version: "v1.0.0", Source: "https://example.test/source.tar.gz", SourceFallback: true}
	err := InstallWithPackageOptions(p, &MockFetcher{pkg: p}, &MockArchiver{}, t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected source fallback to be rejected")
	}
	for _, want := range []string{"owner/tool", "v1.0.0", "--from-source"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message %q must mention %q", err.Error(), want)
		}
	}
}

func TestSourceFallbackMessageTruncatesLongAssetList(t *testing.T) {
	available := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		available = append(available, fmt.Sprintf("tool_%02d.tar.gz", i))
	}
	p := &Package{
		Name:             "owner/tool",
		Version:          "v1.0.0",
		Source:           "https://example.test/source.tar.gz",
		SourceFallback:   true,
		AssetDiagnostics: &AssetDiagnostics{Platform: "linux/amd64", Available: available},
	}

	msg := sourceFallbackMessage(p)
	if !strings.Contains(msg, "tool_00.tar.gz") {
		t.Errorf("message %q must list the first asset", msg)
	}
	if !strings.Contains(msg, "외 2개") {
		t.Errorf("message %q must summarise the omitted assets", msg)
	}
	if strings.Contains(msg, "tool_11.tar.gz") {
		t.Errorf("message %q must not list omitted assets", msg)
	}
}

func TestSourceFallbackMessageWithoutInstallableAssets(t *testing.T) {
	p := &Package{
		Name:             "owner/tool",
		Version:          "v1.0.0",
		Source:           "https://example.test/source.tar.gz",
		SourceFallback:   true,
		AssetDiagnostics: &AssetDiagnostics{Platform: "windows/amd64", Available: nil},
	}

	msg := sourceFallbackMessage(p)
	if !strings.Contains(msg, "windows/amd64") {
		t.Errorf("message %q must mention the detected platform", msg)
	}
	if !strings.Contains(msg, "--from-source") {
		t.Errorf("message %q must mention --from-source", msg)
	}
}

// 릴리스 에셋 선택 진단은 ppm-meta.json과 ppm.lock에 저장되지 않아야 합니다.
func TestAssetDiagnosticsAreNotSerialized(t *testing.T) {
	p := &Package{
		Name:             "owner/tool",
		Version:          "v1.0.0",
		Source:           "https://example.test/tool.tar.gz",
		SourceFallback:   true,
		AssetDiagnostics: &AssetDiagnostics{Platform: "darwin/arm64", Available: []string{"tool_darwin_amd64.tar.gz"}},
	}

	jsonData, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	yamlData, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	for name, data := range map[string][]byte{"json": jsonData, "yaml": yamlData} {
		if strings.Contains(string(data), "tool_darwin_amd64.tar.gz") {
			t.Errorf("%s output must not persist asset diagnostics: %s", name, data)
		}
	}
}

// 진단 정보가 있더라도 옵션으로 소스 빌드를 허용하면 설치는 계속 진행됩니다.
func TestInstallWithPackageOptionsAllowsSourceFallbackWhenEnabled(t *testing.T) {
	setupTempHome(t)

	p := &Package{
		Name:             "owner/tool",
		Version:          "v1.0.0",
		Source:           "https://example.test/source.tar.gz",
		SourceFallback:   true,
		AssetDiagnostics: &AssetDiagnostics{Platform: "darwin/arm64", Available: []string{"tool_darwin_amd64.tar.gz"}},
	}
	archiver := &MockArchiver{}

	if err := InstallWithPackageOptions(p, &MockFetcher{pkg: p}, archiver, t.TempDir(), InstallOptions{AllowSourceBuild: true}); err != nil {
		t.Fatalf("InstallWithPackageOptions with AllowSourceBuild failed: %v", err)
	}
	if !archiver.extracted || !archiver.linked {
		t.Error("source fallback install must extract and link the archive")
	}
}
