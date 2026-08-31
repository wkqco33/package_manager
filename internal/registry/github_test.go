package registry

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/pkg"
)

func TestGitHubRegistry_GetMetadata(t *testing.T) {
	// 모의 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			// 현재 실행 환경에 맞는 에셋 이름을 동적으로 생성
			assetName := fmt.Sprintf("ppm-v1.2.3-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(w, `{
				"tag_name": "v1.2.3",
				"tarball_url": "https://example.com/source.tar.gz",
				"assets": [
					{
						"id": 1,
						"name": "%s",
						"browser_download_url": "https://example.com/%s"
					}
				]
			}`, assetName, assetName)
			return
		}
		if r.URL.Path == "/repos/owner/repo/contents/ppm.json" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"description": "A test package", "author": "Tester", "homepage": "https://test.com"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	g := &GitHubRegistry{
		URL: server.URL,
	}

	p, err := g.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if p.Version != "v1.2.3" {
		t.Errorf("Expected version v1.2.3, got %s", p.Version)
	}
	if p.Source != "https://example.com/"+assetNameForCurrentPlatform() {
		t.Errorf("Expected asset source to be selected, got %s", p.Source)
	}
	if p.Description != "A test package" {
		t.Errorf("Expected description 'A test package', got %s", p.Description)
	}
	if p.Author != "Tester" {
		t.Errorf("Expected author 'Tester', got %s", p.Author)
	}
}

func TestGitHubRegistry_GetMetadataRejectsInvalidPackageName(t *testing.T) {
	g := &GitHubRegistry{}
	for _, name := range []string{"cpp_generator", "/repo", "owner/", "owner/repo/extra"} {
		t.Run(name, func(t *testing.T) {
			if _, err := g.GetMetadata(name); err == nil || !strings.Contains(err.Error(), "owner/repo") {
				t.Fatalf("expected owner/repo validation error, got %v", err)
			}
		})
	}
}

func TestGitHubRegistry_GetMetadataFallsBackToLatestTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			w.WriteHeader(http.StatusNotFound)
		case "/repos/owner/repo/tags":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `[{"name":"v1.2.4","tarball_url":"https://example.com/source-v1.2.4.tar.gz"}]`)
		case "/repos/owner/repo/contents/ppm.json":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	g := &GitHubRegistry{URL: server.URL}
	p, err := g.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if p.Version != "v1.2.4" {
		t.Errorf("Expected version v1.2.4, got %s", p.Version)
	}
	if p.Source != "https://example.com/source-v1.2.4.tar.gz" {
		t.Errorf("Expected source tarball fallback, got %s", p.Source)
	}
	if p.AssetID != 0 {
		t.Errorf("Expected no asset ID for tag fallback, got %d", p.AssetID)
	}
}

func TestGitHubRegistry_GetMetadataRecoversAssetsMissingFromAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			if r.Header.Get("Accept") != "" {
				fmt.Fprint(w, `{"tag_name":"v1.2.5","tarball_url":"https://example.com/source.tar.gz","assets":[]}`)
				return
			}
			http.Redirect(w, r, "/owner/repo/releases/tag/v1.2.5", http.StatusFound)
		case "/owner/repo/releases/latest":
			http.Redirect(w, r, "/owner/repo/releases/tag/v1.2.5", http.StatusFound)
		case "/owner/repo/releases/tag/v1.2.5":
			assetName := fmt.Sprintf("repo_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(w, `<a href="/owner/repo/releases/download/v1.2.5/%s">%s</a>`, assetName, assetName)
		case "/repos/owner/repo/contents/ppm.json":
			fmt.Fprint(w, `{}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	g := &GitHubRegistry{URL: server.URL}
	p, err := g.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	want := fmt.Sprintf("%s/owner/repo/releases/download/v1.2.5/repo_%s_%s.tar.gz", server.URL, runtime.GOOS, runtime.GOARCH)
	if p.Source != want {
		t.Fatalf("expected web release asset %q, got %q", want, p.Source)
	}
}

func TestGitHubRegistry_GetMetadataFallsBackToPublicGitHubAPI(t *testing.T) {
	originalPublicURL := publicGitHubAPIURL
	defer func() { publicGitHubAPIURL = originalPublicURL }()

	privateRegistry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer privateRegistry.Close()

	publicRegistry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"tag_name":"v2.0.0","tarball_url":"https://example.com/v2.tar.gz","assets":[]}`)
		case "/repos/owner/repo/contents/ppm.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"description":"fallback package","author":"Tester"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer publicRegistry.Close()

	publicGitHubAPIURL = publicRegistry.URL

	g := &GitHubRegistry{URL: privateRegistry.URL}
	p, err := g.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if p.Version != "v2.0.0" {
		t.Errorf("Expected version v2.0.0, got %s", p.Version)
	}
	if p.Source != "https://example.com/v2.tar.gz" {
		t.Errorf("Expected fallback tarball source, got %s", p.Source)
	}
	if p.Description != "fallback package" {
		t.Errorf("Expected fallback metadata description, got %s", p.Description)
	}
}

func TestGitHubRegistry_GetMetadataReturnsClearNotFoundError(t *testing.T) {
	originalPublicURL := publicGitHubAPIURL
	defer func() { publicGitHubAPIURL = originalPublicURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	publicGitHubAPIURL = server.URL

	g := &GitHubRegistry{URL: server.URL}
	_, err := g.GetMetadata("owner/typo-repo")
	if err == nil {
		t.Fatal("Expected GetMetadata to fail")
	}

	if got := err.Error(); got == "" || !containsAll(got, "owner/typo-repo", "spelling", "registry_url") {
		t.Fatalf("Expected helpful not found error, got %q", got)
	}
}

func assetNameForCurrentPlatform() string {
	return fmt.Sprintf("ppm-v1.2.3-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}

func TestGitHubRegistry_DownloadSource(t *testing.T) {
	// 모의 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "mock content")
	}))
	defer server.Close()

	g := &GitHubRegistry{}
	p := &pkg.Package{
		Source: server.URL,
	}

	body, _, err := g.DownloadSource(p)
	if err != nil {
		t.Fatalf("DownloadSource failed: %v", err)
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "mock content" {
		t.Errorf("Content mismatch: got %s, want mock content", string(content))
	}
}

func TestGitHubRegistry_DownloadSourcePrefersBrowserDownloadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/download/v1.0.0/tool-linux-amd64" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}
		// 공개 에셋 URL은 토큰 없이도 실제 바이너리 바이트를 반환해야 합니다.
		w.Header().Set("Content-Type", "application/octet-stream")
		fmt.Fprint(w, "ELF binary content")
	}))
	defer server.Close()

	g := &GitHubRegistry{URL: "https://api.github.com"}
	p := &pkg.Package{
		Name:    "owner/repo",
		Source:  server.URL + "/releases/download/v1.0.0/tool-linux-amd64",
		AssetID: 99,
	}

	body, _, err := g.DownloadSource(p)
	if err != nil {
		t.Fatalf("DownloadSource failed: %v", err)
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "ELF binary content" {
		t.Errorf("Content mismatch: got %q", content)
	}
}

func TestGitHubRegistry_DownloadSourceUsesResolvedRegistryURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/assets/99" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, "asset content")
	}))
	defer server.Close()

	g := &GitHubRegistry{URL: "https://custom.example.com"}
	p := &pkg.Package{
		Name:        "owner/repo",
		AssetID:     99,
		RegistryURL: server.URL,
	}

	body, _, err := g.DownloadSource(p)
	if err != nil {
		t.Fatalf("DownloadSource failed: %v", err)
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "asset content" {
		t.Errorf("Content mismatch: got %s, want asset content", string(content))
	}
}

// TestGitHubRegistry_GetMetadataRetriesWithoutTokenOnAuthRejection는 잘못되거나
// 만료된 토큰이 설정되어 있어도 public 저장소는 토큰 없이 재시도하여 다운로드할 수
// 있음을 검증합니다.
func TestGitHubRegistry_GetMetadataRetriesWithoutTokenOnAuthRejection(t *testing.T) {
	var authedRequests, anonymousRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			authedRequests++
			// 잘못된 토큰이면 GitHub처럼 401을 반환합니다.
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		anonymousRequests++
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"tag_name":"v1.0.0","tarball_url":"https://example.com/source.tar.gz","assets":[]}`)
		case "/repos/owner/repo/contents/ppm.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"description":"public package","author":"Tester"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	g := &GitHubRegistry{URL: server.URL, Token: "stale-or-invalid-token"}
	p, err := g.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed with stale token: %v", err)
	}
	if p.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", p.Version)
	}
	if !p.SourceFallback {
		t.Error("Expected source tarball fallback to be marked explicitly")
	}
	if authedRequests == 0 {
		t.Error("Expected at least one authenticated request before the anonymous retry")
	}
	if anonymousRequests == 0 {
		t.Error("Expected an anonymous retry for the public repository")
	}
}

func TestGitHubRegistry_GetMetadataFallsBackToPublicReleasePageOnRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			w.WriteHeader(http.StatusForbidden)
		case "/owner/repo/releases/latest":
			http.Redirect(w, r, "/owner/repo/releases/tag/v2.1.0", http.StatusFound)
		case "/owner/repo/releases/tag/v2.1.0":
			// 최신 releases 페이지처럼 본문에는 에셋 링크가 없습니다.
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<div data-testid="release-body">release</div>`)
		case "/owner/repo/releases/expanded_assets/v2.1.0":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<a href="/owner/repo/releases/download/v2.1.0/ppm-%s-%s.tar.gz">asset</a>`, runtime.GOOS, runtime.GOARCH)
		case "/owner/repo/contents/ppm.json":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	g := &GitHubRegistry{URL: server.URL}
	p, err := g.GetMetadata("owner/repo")
	if err != nil {
		t.Fatalf("GetMetadata failed through release page fallback: %v", err)
	}
	if p.Version != "v2.1.0" {
		t.Errorf("Version = %q, want v2.1.0", p.Version)
	}
	if !strings.Contains(p.Source, "/releases/download/v2.1.0/") {
		t.Errorf("Source = %q, want a release asset URL", p.Source)
	}
	if p.SourceFallback {
		t.Error("Expected expanded release asset to avoid source fallback")
	}
}

// TestGitHubRegistry_GetMetadataKeepsAuthFailureForPrivateRepo는 private 저장소가
// 토큰 없이도 여전히 실패하여 인증 오류가 보존됨을 검증합니다.
func TestGitHubRegistry_GetMetadataKeepsAuthFailureForPrivateRepo(t *testing.T) {
	originalPublicURL := publicGitHubAPIURL
	defer func() { publicGitHubAPIURL = originalPublicURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 토큰 유무와 관계없이 private 저장소는 항상 404를 반환합니다.
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	publicGitHubAPIURL = server.URL

	g := &GitHubRegistry{URL: server.URL, Token: "valid-token"}
	_, err := g.GetMetadata("owner/private-repo")
	if err == nil {
		t.Fatal("Expected GetMetadata to fail for a private repository")
	}
}
