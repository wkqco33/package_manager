package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/auth"
	"github.com/wkqco33/package_manager/internal/config"
)

// authDependencies는 auth 커맨드가 사용하는 의존성입니다. 테스트에서
// 인증 소스와 출력 대상을 주입할 수 있도록 분리했습니다.
type authDependencies struct {
	// ConfigToken은 config.yaml에 명시된 원시 auth_token을 반환합니다.
	ConfigToken func() string
	Login       func(context.Context, io.Writer) error
	Logout      func() error
	Sources     func(configToken string) []auth.Source
	Out         io.Writer
}

func defaultAuthDependencies() authDependencies {
	return authDependencies{
		ConfigToken: func() string {
			token, err := config.ReadConfigToken()
			if err != nil {
				return ""
			}
			return token
		},
		Login:   auth.Login,
		Logout:  auth.Logout,
		Sources: auth.Sources,
		Out:     os.Stdout,
	}
}

var authCmd = newAuthCommand(defaultAuthDependencies())

func newAuthCommand(deps authDependencies) *wcli.Command {
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}

	loginCmd := &wcli.Command{
		Use:   "login",
		Short: "GitHub에 로그인",
		Run: func(ctx *wcli.Context) error {
			return deps.Login(context.Background(), out)
		},
	}
	statusCmd := &wcli.Command{
		Use:   "status",
		Short: "GitHub 인증 상태 확인",
		Run: func(ctx *wcli.Context) error {
			sources := deps.Sources(deps.ConfigToken())
			active := activeSource(sources)
			if active == "" {
				fmt.Fprintln(out, "로그인되지 않았습니다.")
				return nil
			}
			fmt.Fprintf(out, "GitHub 로그인 상태: %s\n", active)
			fmt.Fprintln(out, "인증 소스 확인 (우선순위 순, 첫 번째 감지된 소스를 사용):")
			for _, source := range sources {
				state := "미설정"
				if source.Detected {
					state = "감지됨"
				}
				fmt.Fprintf(out, "  - %s: %s\n", source.Name, state)
			}
			return nil
		},
	}
	logoutCmd := &wcli.Command{
		Use:   "logout",
		Short: "ppm에 저장된 GitHub 인증 정보 삭제",
		Run: func(ctx *wcli.Context) error {
			if err := deps.Logout(); err != nil {
				return fmt.Errorf("GitHub 로그아웃 실패: %w", err)
			}
			fmt.Fprintln(out, "ppm에 저장된 GitHub 인증 정보가 삭제되었습니다.")

			// logout은 ppm이 저장한 토큰만 삭제합니다. gh CLI와 환경 변수 토큰은
			// 그대로 남아 계속 사용되므로 남은 소스와 해제 방법을 안내합니다.
			remaining := detectedSources(deps.Sources(deps.ConfigToken()))
			if len(remaining) == 0 {
				fmt.Fprintln(out, "사용 가능한 다른 인증 소스가 없어 GitHub 인증이 완전히 해제되었습니다.")
				return nil
			}
			fmt.Fprintf(out, "주의: 다음 인증 소스는 그대로 남아 있어 ppm은 계속 해당 토큰을 사용합니다: %s\n", strings.Join(remaining, ", "))
			if containsSource(remaining, "GitHub CLI (gh)") {
				fmt.Fprintln(out, "gh 인증까지 해제하려면 'gh auth logout --hostname github.com'을 실행하세요.")
			}
			if containsSource(remaining, "GITHUB_TOKEN 환경 변수") || containsSource(remaining, "PPM_AUTH_TOKEN 환경 변수") {
				fmt.Fprintln(out, "환경 변수 토큰까지 해제하려면 해당 환경 변수를 unset 하세요.")
			}
			return nil
		},
	}
	cmd := &wcli.Command{
		Use:       "auth",
		Short:     "GitHub 인증 관리",
		OutWriter: out,
	}
	cmd.AddCommand(loginCmd, statusCmd, logoutCmd)
	return cmd
}

func activeSource(sources []auth.Source) string {
	for _, source := range sources {
		if source.Detected {
			return source.Name
		}
	}
	return ""
}

func detectedSources(sources []auth.Source) []string {
	names := make([]string, 0, len(sources))
	for _, source := range sources {
		if source.Detected {
			names = append(names, source.Name)
		}
	}
	return names
}

func containsSource(sources []string, name string) bool {
	for _, source := range sources {
		if source == name {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(authCmd)
}
