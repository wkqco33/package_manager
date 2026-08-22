package registry

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/pkg"
)

// GitHubRegistry는 GitHub용 pkg.RegistryFetcher 구현체입니다.
type GitHubRegistry struct {
	Token              string
	URL                string // 기본값: https://api.github.com
	Mirrors            []string
	TrustedOwners      []string
	RequireChecksum    bool
	RequireSignature   bool
	SignaturePublicKey string
}

// NewGitHubRegistry creates a registry client with the primary URL and optional mirrors.
func NewGitHubRegistry(token, primary string, mirrors, trustedOwners []string, requireChecksum, requireSignature bool, publicKey string) *GitHubRegistry {
	return &GitHubRegistry{Token: token, URL: primary, Mirrors: append([]string(nil), mirrors...), TrustedOwners: append([]string(nil), trustedOwners...), RequireChecksum: requireChecksum, RequireSignature: requireSignature, SignaturePublicKey: publicKey}
}

// GitHubRegistry가 pkg.RegistryFetcher를 구현하는지 확인
var _ pkg.RegistryFetcher = (*GitHubRegistry)(nil)

var apiHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

var downloadHTTPClient = &http.Client{
	Timeout: 10 * time.Minute,
}

const downloadMaxAttempts = 3

type ghAsset struct {
	Id                 int64  `json:"id"`
	Name               string `json:"name"`
	BrowserDownloadUrl string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	TarballUrl string    `json:"tarball_url"`
	Assets     []ghAsset `json:"assets"`
}

type ghTag struct {
	Name       string `json:"name"`
	TarballUrl string `json:"tarball_url"`
}

type SearchResult struct {
	Name        string `json:"full_name"`
	Description string `json:"description"`
	URL         string `json:"html_url"`
}

type searchResponse struct {
	Items []SearchResult `json:"items"`
}

type ppmMeta struct {
	Description           string            `json:"description"`
	Author                string            `json:"author"`
	Homepage              string            `json:"homepage"`
	BinName               string            `json:"bin_name"`
	Dependencies          []string          `json:"dependencies,omitempty"`
	DependencyConstraints map[string]string `json:"dependency_constraints,omitempty"`
}

// UnmarshalJSON accepts both the legacy string-array and the new map form.
func (m *ppmMeta) UnmarshalJSON(data []byte) error {
	var raw struct {
		Description           string            `json:"description"`
		Author                string            `json:"author"`
		Homepage              string            `json:"homepage"`
		BinName               string            `json:"bin_name"`
		Dependencies          json.RawMessage   `json:"dependencies"`
		DependencyConstraints map[string]string `json:"dependency_constraints"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Description, m.Author, m.Homepage, m.BinName = raw.Description, raw.Author, raw.Homepage, raw.BinName
	m.DependencyConstraints = raw.DependencyConstraints
	if len(raw.Dependencies) == 0 || string(raw.Dependencies) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw.Dependencies, &m.Dependencies); err == nil {
		return nil
	}
	return json.Unmarshal(raw.Dependencies, &m.DependencyConstraints)
}

var publicGitHubAPIURL = "https://api.github.com"

var errLatestReleaseNotFound = errors.New("github latest release not found")
var errRepositoryNotFound = errors.New("github repository not found")

func (g *GitHubRegistry) fetchAssetSignature(baseURL, pkgName string, assets []ghAsset, assetName string) (string, error) {
	var signatureAsset *ghAsset
	for i := range assets {
		if assets[i].Name == assetName+".sig" || assets[i].Name == assetName+".asc" {
			signatureAsset = &assets[i]
			break
		}
	}
	if signatureAsset == nil {
		return "", fmt.Errorf("signature asset for %s not found", assetName)
	}
	endpoint := fmt.Sprintf("%s/repos/%s/releases/assets/%d", strings.TrimRight(baseURL, "/"), pkgName, signatureAsset.Id)
	req, err := g.newRequest("GET", endpoint, "application/octet-stream")
	if err != nil {
		return "", err
	}
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("signature asset returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	encoded := strings.TrimSpace(string(data))
	if decoded, decodeErr := base64.StdEncoding.DecodeString(encoded); decodeErr == nil {
		return base64.StdEncoding.EncodeToString(decoded), nil
	}
	return "", fmt.Errorf("invalid base64 signature")
}

func (g *GitHubRegistry) fetchAssetChecksum(baseURL, pkgName string, assets []ghAsset, assetName string) (string, error) {
	var checksumAsset *ghAsset
	for i := range assets {
		if assets[i].Name == assetName+".sha256" || assets[i].Name == assetName+".sha256sum" {
			checksumAsset = &assets[i]
			break
		}
	}
	if checksumAsset == nil {
		return "", fmt.Errorf("checksum asset for %s not found", assetName)
	}
	endpoint := fmt.Sprintf("%s/repos/%s/releases/assets/%d", strings.TrimRight(baseURL, "/"), pkgName, checksumAsset.Id)
	req, err := g.newRequest("GET", endpoint, "application/octet-stream")
	if err != nil {
		return "", err
	}
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum asset returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", fmt.Errorf("invalid SHA-256 checksum")
	}
	for _, c := range fields[0] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return "", fmt.Errorf("invalid SHA-256 checksum")
		}
	}
	return strings.ToLower(fields[0]), nil
}

// Search searches repositories visible to the configured GitHub account.
func (g *GitHubRegistry) Search(query string) ([]SearchResult, error) {
	base := strings.TrimRight(g.URL, "/")
	if base == "" {
		base = publicGitHubAPIURL
	}
	searchURL := fmt.Sprintf("%s/search/repositories?q=%s&per_page=30", base, url.QueryEscape(query))
	req, err := g.newRequest("GET", searchURL, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeNetwork, err, "repository search failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apperr.New(apperr.CodeRegistry, "repository search failed: HTTP %d", resp.StatusCode)
	}
	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, apperr.Wrap(apperr.CodeRegistry, err, "invalid search response")
	}
	return result.Items, nil
}

// GetMetadata는 GitHub 저장소의 최신 릴리스 메타데이터를 조회합니다.
func (g *GitHubRegistry) GetMetadata(pkgName string) (*pkg.Package, error) {
	if len(g.TrustedOwners) > 0 && !g.isTrustedOwner(pkgName) {
		return nil, apperr.New(apperr.CodeRegistry, "repository owner for %s is not trusted", pkgName)
	}
	baseURL, rel, err := g.resolveReleaseMetadata(pkgName)
	if err != nil {
		return nil, err
	}

	p := &pkg.Package{
		Name:         pkgName,
		Version:      rel.TagName,
		RegistryURL:  baseURL,
		ReleaseNotes: rel.Body,
	}
	if rel.TagName == "" {
		return nil, apperr.New(apperr.CodeRegistry, "failed to determine a version for %s", pkgName)
	}

	// 현재 플랫폼에 맞는 최적 에셋 탐색
	bestAsset := g.findBestAsset(rel.Assets)
	if bestAsset != nil {
		p.Source = bestAsset.BrowserDownloadUrl
		p.AssetID = bestAsset.Id
		if checksum, checksumErr := g.fetchAssetChecksum(baseURL, pkgName, rel.Assets, bestAsset.Name); checksumErr == nil {
			p.Checksum = checksum
		} else if g.RequireChecksum {
			return nil, apperr.Wrap(apperr.CodeRegistry, checksumErr, "required checksum asset is unavailable")
		}
		if signature, signatureErr := g.fetchAssetSignature(baseURL, pkgName, rel.Assets, bestAsset.Name); signatureErr == nil {
			p.Signature = signature
		} else if g.RequireSignature {
			return nil, apperr.Wrap(apperr.CodeRegistry, signatureErr, "required signature asset is unavailable")
		}
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
	p.Dependencies = meta.Dependencies
	p.DependencyConstraints = meta.DependencyConstraints

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

func (g *GitHubRegistry) isTrustedOwner(pkgName string) bool {
	owner := strings.SplitN(pkgName, "/", 2)[0]
	for _, trusted := range g.TrustedOwners {
		if strings.EqualFold(owner, strings.TrimSpace(trusted)) {
			return true
		}
	}
	return false
}

func (g *GitHubRegistry) apiBaseCandidates() []string {
	candidates := make([]string, 0, len(g.Mirrors)+2)
	add := func(value string) {
		value = strings.TrimRight(value, "/")
		if value == "" {
			return
		}
		for _, existing := range candidates {
			if existing == value {
				return
			}
		}
		candidates = append(candidates, value)
	}
	add(g.URL)
	for _, mirror := range g.Mirrors {
		add(mirror)
	}
	add(publicGitHubAPIURL)
	return candidates
}

func (g *GitHubRegistry) newRequest(method, url, acceptHeader string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeNetwork, err, "failed to create http request")
	}
	if acceptHeader != "" {
		req.Header.Set("Accept", acceptHeader)
	}
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	return req, nil
}

func (g *GitHubRegistry) fetchPPMMetadata(baseURL, pkgName string) ppmMeta {
	contentURL := fmt.Sprintf("%s/repos/%s/contents/ppm.json", baseURL, pkgName)
	req, err := g.newRequest("GET", contentURL, "application/vnd.github.v3.raw")
	if err != nil {
		return ppmMeta{}
	}

	resp, err := apiHTTPClient.Do(req)
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
	req, err := g.newRequest("GET", apiURL, "application/vnd.github.v3+json")
	if err != nil {
		return ghRelease{}, err
	}

	resp, err := apiHTTPClient.Do(req)
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
	req, err := g.newRequest("GET", tagsURL, "application/vnd.github.v3+json")
	if err != nil {
		return ghTag{}, err
	}

	resp, err := apiHTTPClient.Do(req)
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

// DownloadSource는 소스 아카이브를 처음부터 반환합니다.
func (g *GitHubRegistry) DownloadSource(p *pkg.Package) (io.ReadCloser, int64, error) {
	body, size, _, err := g.DownloadSourceAt(p, 0)
	return body, size, err
}

// DownloadSourceAt uses HTTP Range when offset is non-zero. The bool result
// tells callers whether the server accepted the range (206 Partial Content).
func (g *GitHubRegistry) DownloadSourceAt(p *pkg.Package, offset int64) (io.ReadCloser, int64, bool, error) {
	downloadURL := p.Source
	if p.AssetID > 0 {
		assetBaseURL := p.RegistryURL
		if assetBaseURL == "" {
			assetBaseURL = g.URL
		}
		downloadURL = fmt.Sprintf("%s/repos/%s/releases/assets/%d", strings.TrimRight(assetBaseURL, "/"), p.Name, p.AssetID)
	}
	acceptHeader := ""
	if p.AssetID > 0 {
		acceptHeader = "application/octet-stream"
	}
	var lastErr error
	for attempt := 1; attempt <= downloadMaxAttempts; attempt++ {
		req, err := g.newRequest("GET", downloadURL, acceptHeader)
		if err != nil {
			return nil, 0, false, err
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
		resp, err := downloadHTTPClient.Do(req)
		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent) {
			return resp.Body, resp.ContentLength, offset > 0 && resp.StatusCode == http.StatusPartialContent, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			resp.Body.Close()
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				break
			}
		}
		if attempt < downloadMaxAttempts {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}
	return nil, 0, false, apperr.Wrap(apperr.CodeNetwork, lastErr, "failed to download source after retries")
}
