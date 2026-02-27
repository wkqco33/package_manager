package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
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

// GetMetadata fetches the latest release metadata for a given GitHub repository.
// The release API call and the optional ppm.json fetch are performed concurrently.
func (g *GitHubRegistry) GetMetadata(pkgName string) (*pkg.Package, error) {
	// pkgName assumed format: "owner/repo"
	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", g.URL, pkgName)
	contentURL := fmt.Sprintf("%s/repos/%s/contents/ppm.json", g.URL, pkgName)

	type releaseResult struct {
		rel ghRelease
		err error
	}
	type ppmMeta struct {
		Description string `json:"description"`
		Author      string `json:"author"`
		Homepage    string `json:"homepage"`
	}
	type metaResult struct {
		meta ppmMeta
		err  error
	}

	relCh := make(chan releaseResult, 1)
	metaCh := make(chan metaResult, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: fetch release info
	go func() {
		defer wg.Done()
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			relCh <- releaseResult{err: apperr.Wrap(apperr.CodeNetwork, err, "failed to create github api request")}
			return
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if g.Token != "" {
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
		resp, err := defaultHTTPClient.Do(req)
		if err != nil {
			relCh <- releaseResult{err: apperr.Wrap(apperr.CodeNetwork, err, "failed to execute github api request")}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			relCh <- releaseResult{err: apperr.New(apperr.CodeRegistry, "failed to get release info for %s: HTTP %d", pkgName, resp.StatusCode)}
			return
		}
		var rel ghRelease
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			relCh <- releaseResult{err: apperr.Wrap(apperr.CodeRegistry, err, "failed to decode github release metadata")}
			return
		}
		relCh <- releaseResult{rel: rel}
	}()

	// Goroutine 2: fetch optional ppm.json (failure is non-fatal)
	go func() {
		defer wg.Done()
		req, err := http.NewRequest("GET", contentURL, nil)
		if err != nil {
			metaCh <- metaResult{}
			return
		}
		req.Header.Set("Accept", "application/vnd.github.v3.raw")
		if g.Token != "" {
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
		resp, err := defaultHTTPClient.Do(req)
		if err != nil {
			metaCh <- metaResult{}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			metaCh <- metaResult{}
			return
		}
		var meta ppmMeta
		if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
			metaCh <- metaResult{}
			return
		}
		metaCh <- metaResult{meta: meta}
	}()

	wg.Wait()

	relRes := <-relCh
	if relRes.err != nil {
		return nil, relRes.err
	}

	p := &pkg.Package{
		Name:    pkgName,
		Version: relRes.rel.TagName,
		Source:  relRes.rel.TarballUrl,
	}

	metaRes := <-metaCh
	p.Description = metaRes.meta.Description
	p.Author = metaRes.meta.Author
	p.Homepage = metaRes.meta.Homepage

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
