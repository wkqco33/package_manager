package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ppm/internal/apperr"
	"ppm/internal/pkg"
)

// GitHubRegistry implements the pkg.RegistryFetcher interface for GitHub
type GitHubRegistry struct {
	Token string
	URL   string // default: https://api.github.com
}

// Ensure GitHubRegistry implements pkg.RegistryFetcher
var _ pkg.RegistryFetcher = (*GitHubRegistry)(nil)

var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

type ghRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	TarballUrl string `json:"tarball_url"`
}

// GetMetadata fetches the latest release metadata for a given GitHub repository
func (g *GitHubRegistry) GetMetadata(pkgName string) (*pkg.Package, error) {
	// pkgName assumed format: "owner/repo"
	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", g.URL, pkgName)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeNetwork, err, "failed to create github api request")
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeNetwork, err, "failed to execute github api request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Could read body to provide better error message
		return nil, apperr.New(apperr.CodeRegistry, "failed to get release info for %s: HTTP %d", pkgName, resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, apperr.Wrap(apperr.CodeRegistry, err, "failed to decode github release metadata")
	}

	p := &pkg.Package{
		Name:    pkgName,
		Version: rel.TagName,
		Source:  rel.TarballUrl, // Prefer tarball
	}

	// Fetch optional ppm.json for rich metadata
	contentURL := fmt.Sprintf("%s/repos/%s/contents/ppm.json", g.URL, pkgName)
	contentReq, _ := http.NewRequest("GET", contentURL, nil)
	contentReq.Header.Set("Accept", "application/vnd.github.v3.raw")
	if g.Token != "" {
		contentReq.Header.Set("Authorization", "Bearer "+g.Token)
	}

	contentResp, err := defaultHTTPClient.Do(contentReq)
	if err == nil {
		defer contentResp.Body.Close()
		if contentResp.StatusCode == http.StatusOK {
			var meta struct {
				Description string `json:"description"`
				Author      string `json:"author"`
				Homepage    string `json:"homepage"`
			}
			if err := json.NewDecoder(contentResp.Body).Decode(&meta); err == nil {
				p.Description = meta.Description
				p.Author = meta.Author
				p.Homepage = meta.Homepage
			}
		}
	}

	return p, nil
}

// DownloadSource returns a reader for the source tarball
func (g *GitHubRegistry) DownloadSource(p *pkg.Package) (io.ReadCloser, int64, error) {
	req, err := http.NewRequest("GET", p.Source, nil)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.CodeNetwork, err, "failed to create download request")
	}

	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.CodeNetwork, err, "failed to execute download request")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, apperr.New(apperr.CodeNetwork, "failed to download source: HTTP %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}
