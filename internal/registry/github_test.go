package registry

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ppm/internal/pkg"
)

func TestGitHubRegistry_GetMetadata(t *testing.T) {
	// 모의 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"tag_name": "v1.2.3", "tarball_url": "https://example.com/source.tar.gz"}`)
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
	if p.Description != "A test package" {
		t.Errorf("Expected description 'A test package', got %s", p.Description)
	}
	if p.Author != "Tester" {
		t.Errorf("Expected author 'Tester', got %s", p.Author)
	}
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
