package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"ppm/internal/apperr"
	"ppm/internal/pkg"
)

// GitHubRegistry는 GitHub용 pkg.RegistryFetcher 구현체입니다.
type GitHubRegistry struct {
	Token string
	URL   string // 기본값: https://api.github.com
}

// GitHubRegistry가 pkg.RegistryFetcher를 구현하는지 확인
var _ pkg.RegistryFetcher = (*GitHubRegistry)(nil)

var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

type ghAsset struct {
	Id                 int64  `json:"id"`
	Name               string `json:"name"`
	BrowserDownloadUrl string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	TarballUrl string    `json:"tarball_url"`
	Assets     []ghAsset `json:"assets"`
}

type ghTag struct {
	Name       string `json:"name"`
	TarballUrl string `json:"tarball_url"`
}

type ppmMeta struct {
	Description string `json:"description"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
	BinName     string `json:"bin_name"`
}

var publicGitHubAPIURL = "https://api.github.com"

var errLatestReleaseNotFound = errors.New("github latest release not found")
var errRepositoryNotFound = errors.New("github repository not found")

// GetMetadata는 GitHub 저장소의 최신 릴리스 메타데이터를 조회합니다.
func (g *GitHubRegistry) GetMetadata(pkgName string) (*pkg.Package, error) {
	baseURL, rel, err := g.resolveReleaseMetadata(pkgName)
	if err != nil {
		return nil, err
	}

	p := &pkg.Package{
		Name:        pkgName,
		Version:     rel.TagName,
		RegistryURL: baseURL,
	}
	if rel.TagName == "" {
		return nil, apperr.New(apperr.CodeRegistry, "failed to determine a version for %s", pkgName)
	}

	// 현재 플랫폼에 맞는 최적 에셋 탐색
	bestAsset := g.findBestAsset(rel.Assets)
	if bestAsset != nil {
		p.Source = bestAsset.BrowserDownloadUrl
		p.AssetID = bestAsset.Id
	} else if rel.TarballUrl != "" {
		p.Source = rel.TarballUrl
	} else {
		return nil, apperr.New(apperr.CodeRegistry, "현재 플랫폼(%s/%s)에 맞는 바이너리 에셋이나 소스 아카이브를 찾을 수 없습니다.", runtime.GOOS, runtime.GOARCH)
	}

	meta := g.fetchPPMMetadata(baseURL, pkgName)
	p.Description = meta.Description
	p.Author = meta.Author
	p.Homepage = meta.Homepage
	p.BinName = meta.BinName

	return p, nil
}

func (g *GitHubRegistry) resolveReleaseMetadata(pkgName string) (string, ghRelease, error) {
	for _, baseURL := range g.apiBaseCandidates() {
		apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", baseURL, pkgName)
		rel, err := g.fetchLatestRelease(pkgName, apiURL)
		if err == nil {
			return baseURL, rel, nil
		}
		if !errors.Is(err, errLatestReleaseNotFound) {
			return "", ghRelease{}, err
		}

		tag, tagErr := g.fetchLatestTag(pkgName, baseURL)
		if tagErr == nil {
			return baseURL, ghRelease{
				TagName:    tag.Name,
				TarballUrl: tag.TarballUrl,
			}, nil
		}
		if errors.Is(tagErr, errRepositoryNotFound) {
			continue
		}
		return "", ghRelease{}, tagErr
	}

	return "", ghRelease{}, apperr.New(apperr.CodeRegistry, "repository %s was not found. Check the owner/repo spelling, auth_token, and registry_url.", pkgName)
}

func (g *GitHubRegistry) apiBaseCandidates() []string {
	baseURL := strings.TrimRight(g.URL, "/")
	if baseURL == "" {
		baseURL = publicGitHubAPIURL
	}
	if baseURL == publicGitHubAPIURL {
		return []string{publicGitHubAPIURL}
	}
	return []string{baseURL, publicGitHubAPIURL}
}

func (g *GitHubRegistry) fetchPPMMetadata(baseURL, pkgName string) ppmMeta {
	contentURL := fmt.Sprintf("%s/repos/%s/contents/ppm.json", baseURL, pkgName)
	req, err := http.NewRequest("GET", contentURL, nil)
	if err != nil {
		return ppmMeta{}
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return ppmMeta{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ppmMeta{}
	}

	var meta ppmMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return ppmMeta{}
	}
	return meta
}

func (g *GitHubRegistry) fetchLatestRelease(pkgName, apiURL string) (ghRelease, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return ghRelease{}, apperr.Wrap(apperr.CodeNetwork, err, "failed to create github api request")
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return ghRelease{}, apperr.Wrap(apperr.CodeNetwork, err, "failed to execute github api request")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ghRelease{}, errLatestReleaseNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return ghRelease{}, apperr.New(apperr.CodeRegistry, "failed to get release info for %s: HTTP %d", pkgName, resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ghRelease{}, apperr.Wrap(apperr.CodeRegistry, err, "failed to decode github release metadata")
	}
	return rel, nil
}

func (g *GitHubRegistry) fetchLatestTag(pkgName, baseURL string) (ghTag, error) {
	tagsURL := fmt.Sprintf("%s/repos/%s/tags", baseURL, pkgName)
	req, err := http.NewRequest("GET", tagsURL, nil)
	if err != nil {
		return ghTag{}, apperr.Wrap(apperr.CodeNetwork, err, "failed to create github tags request")
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return ghTag{}, apperr.Wrap(apperr.CodeNetwork, err, "failed to execute github tags request")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ghTag{}, errRepositoryNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return ghTag{}, apperr.New(apperr.CodeRegistry, "failed to get tag info for %s: HTTP %d", pkgName, resp.StatusCode)
	}

	var tags []ghTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return ghTag{}, apperr.Wrap(apperr.CodeRegistry, err, "failed to decode github tag metadata")
	}
	if len(tags) == 0 {
		return ghTag{}, apperr.New(apperr.CodeRegistry, "no releases or tags found for %s", pkgName)
	}
	return tags[0], nil
}

func (g *GitHubRegistry) findBestAsset(assets []ghAsset) *ghAsset {
	osNames := []string{runtime.GOOS}
	if runtime.GOOS == "darwin" {
		osNames = append(osNames, "macos", "apple-darwin")
	}

	archNames := []string{runtime.GOARCH}
	var forbiddenArch []string

	if runtime.GOARCH == "amd64" {
		archNames = append(archNames, "x86_64", "64bit")
		forbiddenArch = []string{"arm64", "aarch64", "armv"}
	} else if runtime.GOARCH == "arm64" {
		archNames = append(archNames, "aarch64")
		forbiddenArch = []string{"amd64", "x86_64", "x86", "i386"}
	}

	var bestAsset *ghAsset
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)

		// OS 매칭
		osMatch := false
		for _, osName := range osNames {
			if strings.Contains(name, osName) {
				osMatch = true
				break
			}
		}
		if !osMatch {
			continue
		}

		// 아키텍처 매칭
		archMatch := false
		for _, archName := range archNames {
			if strings.Contains(name, archName) {
				archMatch = true
				break
			}
		}

		// 금지된 아키텍처가 포함되어 있다면 매칭 실패로 간주 (예: arm64 환경에서 amd64가 포함된 경우)
		for _, forbidden := range forbiddenArch {
			if strings.Contains(name, forbidden) {
				archMatch = false
				break
			}
		}

		if !archMatch {
			continue
		}

		// 압축 파일(.tar.gz, .zip, .tgz)을 발견하면 즉시 반환 (우선순위 높음)
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tgz") {
			return &asset
		}

		// 단일 바이너리인 경우 일단 보관하고 더 좋은(압축된) 에셋이 있는지 계속 탐색
		bestAsset = &asset
	}

	return bestAsset
}

// DownloadSource는 소스 아카이브 리더를 반환합니다.
func (g *GitHubRegistry) DownloadSource(p *pkg.Package) (io.ReadCloser, int64, error) {
	downloadURL := p.Source

	// 프라이빗 저장소의 릴리스 에셋인 경우 전용 API URL 사용
	if p.AssetID > 0 {
		// p.Source가 "https://github.com/owner/repo/releases/download/v1.0.0/asset.zip" 형태라면
		// 이를 "https://api.github.com/repos/owner/repo/releases/assets/asset_id" 형태로 변환하거나
		// p.Name(owner/repo)을 활용하여 직접 생성합니다.
		assetBaseURL := p.RegistryURL
		if assetBaseURL == "" {
			assetBaseURL = g.URL
		}
		downloadURL = fmt.Sprintf("%s/repos/%s/releases/assets/%d", strings.TrimRight(assetBaseURL, "/"), p.Name, p.AssetID)
	}

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.CodeNetwork, err, "failed to create download request")
	}

	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
		// 릴리스 에셋 API를 호출할 때는 이 헤더가 필수입니다.
		if p.AssetID > 0 {
			req.Header.Set("Accept", "application/octet-stream")
		}
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
