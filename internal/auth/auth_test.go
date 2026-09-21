package auth

import (
	"errors"
	"testing"
)

type memoryStore struct{ token string }

func (s *memoryStore) Get(string, string) (string, error) {
	if s.token == "" {
		return "", errors.New("not found")
	}
	return s.token, nil
}
func (s *memoryStore) Set(_, _, token string) error { s.token = token; return nil }
func (s *memoryStore) Delete(string, string) error  { s.token = ""; return nil }

func TestResolveTokenPrefersExplicitEnvironment(t *testing.T) {
	oldStore := store
	store = &memoryStore{token: "stored-token"}
	t.Cleanup(func() { store = oldStore })
	t.Setenv("PPM_AUTH_TOKEN", "environment-token")
	t.Setenv("GITHUB_TOKEN", "")

	if got := ResolveToken("config-token"); got != "environment-token" {
		t.Fatalf("ResolveToken() = %q, want environment-token", got)
	}
}

func TestResolveTokenUsesCredentialStoreBeforeGitHubCLI(t *testing.T) {
	oldStore, oldCommand := store, commandOutput
	store = &memoryStore{token: "stored-token"}
	commandOutput = func(string, ...string) ([]byte, error) {
		return []byte("gh-token"), nil
	}
	t.Cleanup(func() { store, commandOutput = oldStore, oldCommand })
	t.Setenv("PPM_AUTH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	if got := ResolveToken(""); got != "stored-token" {
		t.Fatalf("ResolveToken() = %q, want stored-token", got)
	}
}

func TestResolveTokenFallsBackToGitHubCLI(t *testing.T) {
	oldStore, oldCommand := store, commandOutput
	store = &memoryStore{}
	commandOutput = func(name string, args ...string) ([]byte, error) {
		if name != "gh" || len(args) != 4 || args[0] != "auth" || args[1] != "token" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return []byte("gh-token\n"), nil
	}
	t.Cleanup(func() { store, commandOutput = oldStore, oldCommand })
	t.Setenv("PPM_AUTH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	if got := ResolveToken(""); got != "gh-token" {
		t.Fatalf("ResolveToken() = %q, want gh-token", got)
	}
}

// 인증 소스 조회는 ResolveToken과 동일한 우선순위를 그대로 노출해야 합니다.
// 그래야 auth logout 후에도 GitHub CLI 토큰이 계속 사용되는 이유를
// 사용자가 확인할 수 있습니다.
func TestSourcesReportsCandidatesInResolutionOrder(t *testing.T) {
	oldStore, oldCommand := store, commandOutput
	store = &memoryStore{}
	commandOutput = func(string, ...string) ([]byte, error) {
		return nil, errors.New("gh not installed")
	}
	t.Cleanup(func() { store, commandOutput = oldStore, oldCommand })
	t.Setenv("PPM_AUTH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	got := Sources("")
	want := []string{
		"PPM_AUTH_TOKEN 환경 변수",
		"GITHUB_TOKEN 환경 변수",
		"config.yaml auth_token",
		"ppm credential store",
		"GitHub CLI (gh)",
	}
	if len(got) != len(want) {
		t.Fatalf("Sources() = %+v, want %d candidates", got, len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("Sources()[%d].Name = %q, want %q", i, got[i].Name, name)
		}
		if got[i].Detected {
			t.Errorf("Sources()[%d] (%s) must not be detected", i, got[i].Name)
		}
	}

	// 소스가 하나도 감지되지 않으면 인증되지 않은 상태입니다.
	if _, err := Status(""); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("Status() error = %v, want ErrNotAuthenticated", err)
	}
}

func TestSourcesDetectsEnvironmentConfigAndGitHubCLI(t *testing.T) {
	oldStore, oldCommand := store, commandOutput
	store = &memoryStore{}
	commandOutput = func(string, ...string) ([]byte, error) {
		return []byte("gh-token\n"), nil
	}
	t.Cleanup(func() { store, commandOutput = oldStore, oldCommand })
	t.Setenv("PPM_AUTH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "env-token")

	got := Sources("config-token")
	detected := make(map[string]bool, len(got))
	for _, source := range got {
		detected[source.Name] = source.Detected
	}
	for _, name := range []string{"GITHUB_TOKEN 환경 변수", "config.yaml auth_token", "GitHub CLI (gh)"} {
		if !detected[name] {
			t.Errorf("Sources() must detect %s: %+v", name, got)
		}
	}
	if detected["PPM_AUTH_TOKEN 환경 변수"] {
		t.Errorf("Sources() must not detect an unset environment variable: %+v", got)
	}
	if detected["ppm credential store"] {
		t.Errorf("Sources() must not detect an empty credential store: %+v", got)
	}

	// 실제로 사용되는 소스는 우선순위가 가장 높은 GITHUB_TOKEN 입니다.
	if token := ResolveToken("config-token"); token != "env-token" {
		t.Fatalf("ResolveToken() = %q, want env-token", token)
	}
	if source, err := Status("config-token"); err != nil || source != "GITHUB_TOKEN 환경 변수" {
		t.Fatalf("Status() = (%q, %v), want GITHUB_TOKEN 환경 변수", source, err)
	}
}

func TestSourcesDetectsCredentialStoreWithoutGitHubCLI(t *testing.T) {
	oldStore, oldCommand := store, commandOutput
	store = &memoryStore{token: "stored-token"}
	commandOutput = func(string, ...string) ([]byte, error) {
		return nil, errors.New("gh not installed")
	}
	t.Cleanup(func() { store, commandOutput = oldStore, oldCommand })
	t.Setenv("PPM_AUTH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	got := Sources("")
	detected := make(map[string]bool, len(got))
	for _, source := range got {
		detected[source.Name] = source.Detected
	}
	if !detected["ppm credential store"] {
		t.Errorf("Sources() must detect the ppm credential store: %+v", got)
	}
	if detected["GitHub CLI (gh)"] {
		t.Errorf("Sources() must not detect a missing gh CLI: %+v", got)
	}
	if source, err := Status(""); err != nil || source != "ppm credential store" {
		t.Fatalf("Status() = (%q, %v), want ppm credential store", source, err)
	}
}
