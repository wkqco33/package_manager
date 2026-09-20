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
