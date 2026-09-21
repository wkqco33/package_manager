package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/auth"
)

func newTestAuthDependencies(out io.Writer) authDependencies {
	return authDependencies{
		ConfigToken: func() string { return "" },
		Login:       func(context.Context, io.Writer) error { return nil },
		Logout:      func() error { return nil },
		Sources: func(string) []auth.Source {
			return []auth.Source{
				{Name: "PPM_AUTH_TOKEN 환경 변수"},
				{Name: "GITHUB_TOKEN 환경 변수"},
				{Name: "config.yaml auth_token"},
				{Name: "ppm credential store"},
				{Name: "GitHub CLI (gh)", Detected: true},
			}
		},
		Out: out,
	}
}

// auth status는 실제로 사용되는 소스와 함께 각 후보 소스의 감지 여부를 보여주어야
// 합니다. 그래야 auth logout 후에도 gh 토큰이 계속 사용되는 이유를 알 수 있습니다.
func TestAuthStatusShowsActiveSourceAndAllCandidates(t *testing.T) {
	var out bytes.Buffer
	command := newAuthCommand(newTestAuthDependencies(&out))

	if err := command.Execute([]string{"status"}); err != nil {
		t.Fatalf("auth status error = %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"GitHub 로그인 상태: GitHub CLI (gh)",
		"ppm credential store: 미설정",
		"GitHub CLI (gh): 감지됨",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("auth status output %q must contain %q", got, want)
		}
	}
}

func TestAuthStatusReportsNotLoggedIn(t *testing.T) {
	var out bytes.Buffer
	deps := newTestAuthDependencies(&out)
	deps.Sources = func(string) []auth.Source {
		return []auth.Source{{Name: "GitHub CLI (gh)"}}
	}
	command := newAuthCommand(deps)

	if err := command.Execute([]string{"status"}); err != nil {
		t.Fatalf("auth status error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "로그인되지 않았습니다.") {
		t.Errorf("auth status output = %q, want a not-logged-in notice", got)
	}
}

// ppm auth logout은 ppm이 저장한 토큰만 지웁니다. gh CLI나 환경 변수 토큰이
// 남아 있으면 계속 사용되므로, 그 사실과 해제 방법을 안내해야 합니다.
func TestAuthLogoutWarnsAboutRemainingSources(t *testing.T) {
	var out bytes.Buffer
	loggedOut := false
	deps := newTestAuthDependencies(&out)
	deps.Logout = func() error { loggedOut = true; return nil }
	deps.Sources = func(string) []auth.Source {
		return []auth.Source{
			{Name: "GITHUB_TOKEN 환경 변수", Detected: true},
			{Name: "ppm credential store"},
			{Name: "GitHub CLI (gh)", Detected: true},
		}
	}
	command := newAuthCommand(deps)

	if err := command.Execute([]string{"logout"}); err != nil {
		t.Fatalf("auth logout error = %v", err)
	}
	if !loggedOut {
		t.Fatal("auth logout must delete the stored credential")
	}

	got := out.String()
	for _, want := range []string{
		"ppm에 저장된 GitHub 인증 정보가 삭제되었습니다.",
		"GITHUB_TOKEN 환경 변수",
		"GitHub CLI (gh)",
		"gh auth logout --hostname github.com",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("auth logout output %q must contain %q", got, want)
		}
	}
	if strings.Contains(got, "ppm credential store") {
		t.Errorf("auth logout output %q must not warn about the deleted ppm credential store", got)
	}
}

func TestAuthLogoutReportsCompleteLogout(t *testing.T) {
	var out bytes.Buffer
	deps := newTestAuthDependencies(&out)
	deps.Sources = func(string) []auth.Source {
		return []auth.Source{{Name: "ppm credential store"}, {Name: "GitHub CLI (gh)"}}
	}
	command := newAuthCommand(deps)

	if err := command.Execute([]string{"logout"}); err != nil {
		t.Fatalf("auth logout error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "완전히 해제되었습니다") {
		t.Errorf("auth logout output = %q, want a complete-logout notice", got)
	}
}

func TestAuthLogoutPropagatesFailure(t *testing.T) {
	var out bytes.Buffer
	wantErr := errors.New("keyring unavailable")
	deps := newTestAuthDependencies(&out)
	deps.Logout = func() error { return wantErr }
	command := newAuthCommand(deps)

	err := command.Execute([]string{"logout"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("auth logout error = %v, want %v", err, wantErr)
	}
	if out.Len() != 0 {
		t.Errorf("auth logout must not report success on failure: %q", out.String())
	}
}

// config.yaml에 원시 auth_token이 있을 때만 해당 소스가 감지되어야 합니다.
// LoadConfig는 토큰을 해석하므로 status는 해석 전 값을 사용해야 합니다.
func TestAuthStatusUsesRawConfigToken(t *testing.T) {
	var out bytes.Buffer
	deps := newTestAuthDependencies(&out)
	deps.ConfigToken = func() string { return "config-token" }
	var gotToken string
	deps.Sources = func(configToken string) []auth.Source {
		gotToken = configToken
		return []auth.Source{{Name: "config.yaml auth_token", Detected: configToken != ""}}
	}
	command := newAuthCommand(deps)

	if err := command.Execute([]string{"status"}); err != nil {
		t.Fatalf("auth status error = %v", err)
	}
	if gotToken != "config-token" {
		t.Errorf("Sources() received %q, want the raw config token", gotToken)
	}
	if got := out.String(); !strings.Contains(got, "GitHub 로그인 상태: config.yaml auth_token") {
		t.Errorf("auth status output = %q, want config.yaml as the active source", got)
	}
}
