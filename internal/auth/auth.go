// Package auth는 GitHub 인증 정보의 조회와 OAuth Device Flow를 담당합니다.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/wkqco33/package_manager/internal/platform"
)

const (
	GitHubHost = "github.com"
	// GitHub OAuth App의 Client ID는 비밀값이 아니므로 실행 파일에 포함할 수 있습니다.
	GitHubClientID = "Ov23liXrOIgHjyvOIMIV"
	keyringService = "ppm/github"
	keyringUser    = "access-token"
)

var (
	deviceCodeURL = "https://github.com/login/device/code"
	tokenURL      = "https://github.com/login/oauth/access_token"
	httpClient    = &http.Client{Timeout: 15 * time.Second}
	commandOutput = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).Output()
	}
	openBrowser = func(rawURL string) error {
		var name string
		var args []string
		switch runtime.GOOS {
		case "darwin":
			name, args = "open", []string{rawURL}
		case "windows":
			name, args = "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}
		default:
			name, args = "xdg-open", []string{rawURL}
		}
		_, err := commandOutput(name, args...)
		return err
	}
)

type CredentialStore interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type keyringStore struct{}

func (keyringStore) Get(s, u string) (string, error) { return keyring.Get(s, u) }
func (keyringStore) Set(s, u, p string) error        { return keyring.Set(s, u, p) }
func (keyringStore) Delete(s, u string) error        { return keyring.Delete(s, u) }

type persistentStore struct{}

func (persistentStore) Get(service, user string) (string, error) {
	if token, err := (keyringStore{}).Get(service, user); err == nil && token != "" {
		return token, nil
	}
	path, err := credentialPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (persistentStore) Set(service, user, password string) error {
	if err := (keyringStore{}).Set(service, user, password); err == nil {
		return nil
	}
	path, err := credentialPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(password+"\n"), 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func (persistentStore) Delete(service, user string) error {
	keyringErr := (keyringStore{}).Delete(service, user)
	path, pathErr := credentialPath()
	if pathErr != nil {
		return pathErr
	}
	fileErr := os.Remove(path)
	if errors.Is(fileErr, os.ErrNotExist) {
		fileErr = nil
	}
	if fileErr == nil {
		return nil
	}
	if keyringErr != nil && !errors.Is(keyringErr, keyring.ErrNotFound) {
		return keyringErr
	}
	return fileErr
}

func credentialPath() (string, error) {
	paths, err := platform.GetPaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.ConfigDir, "credentials"), nil
}

var store CredentialStore = persistentStore{}

var ErrNotAuthenticated = errors.New("github authentication is not configured")

// ResolveToken은 명시적 토큰, OS credential store, gh 순서로 인증 정보를 찾습니다.
func ResolveToken(configToken string) string {
	if token := os.Getenv("PPM_AUTH_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	if configToken != "" {
		return configToken
	}
	if token, err := store.Get(keyringService, keyringUser); err == nil && token != "" {
		return token
	}
	if token, err := tokenFromGitHubCLI(); err == nil {
		return token
	}
	return ""
}

func tokenFromGitHubCLI() (string, error) {
	data, err := commandOutput("gh", "auth", "token", "--hostname", GitHubHost)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", ErrNotAuthenticated
	}
	return token, nil
}

// Login은 gh 인증을 먼저 재사용하고, 실패하면 OAuth Device Flow를 실행합니다.
func Login(ctx context.Context, out io.Writer) error {
	if _, err := tokenFromGitHubCLI(); err == nil {
		fmt.Fprintln(out, "기존 GitHub CLI 인증을 사용합니다.")
		return nil
	}
	return deviceLogin(ctx, out)
}

type deviceResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int    `json:"interval"`
}

func deviceLogin(ctx context.Context, out io.Writer) error {
	form := url.Values{"client_id": {GitHubClientID}, "scope": {"repo"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub device code 요청 실패: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub device code 요청 실패: HTTP %d", resp.StatusCode)
	}
	var device deviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
		return fmt.Errorf("device code 응답 해석 실패: %w", err)
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI == "" {
		return errors.New("GitHub device code 응답이 올바르지 않습니다")
	}
	fmt.Fprintf(out, "브라우저에서 %s 를 열고 다음 코드를 입력하세요: %s\n", device.VerificationURI, device.UserCode)
	if device.VerificationURIComplete != "" {
		_ = openBrowser(device.VerificationURIComplete)
	} else {
		_ = openBrowser(device.VerificationURI)
	}

	interval := time.Duration(device.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.NewTimer(time.Duration(device.ExpiresIn) * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("GitHub device 인증 시간이 만료되었습니다")
		case <-time.After(interval):
		}
		form := url.Values{"client_id": {GitHubClientID}, "device_code": {device.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("GitHub token 요청 실패: %w", err)
		}
		var token tokenResponse
		err = json.NewDecoder(resp.Body).Decode(&token)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("GitHub token 응답 해석 실패: %w", err)
		}
		if token.AccessToken != "" {
			if err := store.Set(keyringService, keyringUser, token.AccessToken); err != nil {
				return fmt.Errorf("인증 토큰을 OS credential store에 저장할 수 없습니다: %w", err)
			}
			fmt.Fprintln(out, "GitHub 로그인이 완료되었습니다.")
			return nil
		}
		switch token.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			return errors.New("GitHub 인증이 거부되었습니다")
		case "expired_token":
			return errors.New("GitHub device 인증 코드가 만료되었습니다")
		default:
			return fmt.Errorf("GitHub 인증 실패: %s", token.ErrorDescription)
		}
	}
}

func Logout() error {
	err := store.Delete(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func Status() (string, error) {
	if os.Getenv("PPM_AUTH_TOKEN") != "" {
		return "PPM_AUTH_TOKEN 환경 변수", nil
	}
	if os.Getenv("GITHUB_TOKEN") != "" {
		return "GITHUB_TOKEN 환경 변수", nil
	}
	if token, err := store.Get(keyringService, keyringUser); err == nil && token != "" {
		return "ppm OS credential store", nil
	}
	if _, err := tokenFromGitHubCLI(); err == nil {
		return "GitHub CLI (gh)", nil
	}
	return "", ErrNotAuthenticated
}
