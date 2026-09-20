package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/auth"
	"github.com/wkqco33/package_manager/internal/config"
)

var authCmd = newAuthCommand()

func newAuthCommand() *wcli.Command {
	loginCmd := &wcli.Command{
		Use:   "login",
		Short: "GitHub에 로그인",
		Run: func(ctx *wcli.Context) error {
			return auth.Login(context.Background(), os.Stdout)
		},
	}
	statusCmd := &wcli.Command{
		Use:   "status",
		Short: "GitHub 인증 상태 확인",
		Run: func(ctx *wcli.Context) error {
			source, err := auth.Status()
			if err != nil {
				if cfg, configErr := config.LoadConfig(); configErr == nil && cfg.AuthToken != "" {
					source = "config.yaml"
				} else {
					fmt.Println("로그인되지 않았습니다.")
					return nil
				}
			}
			fmt.Printf("GitHub 로그인 상태: %s\n", source)
			return nil
		},
	}
	logoutCmd := &wcli.Command{
		Use:   "logout",
		Short: "ppm에 저장된 GitHub 인증 정보 삭제",
		Run: func(ctx *wcli.Context) error {
			if err := auth.Logout(); err != nil {
				return fmt.Errorf("GitHub 로그아웃 실패: %w", err)
			}
			fmt.Println("ppm에 저장된 GitHub 인증 정보가 삭제되었습니다.")
			return nil
		},
	}
	cmd := &wcli.Command{
		Use:   "auth",
		Short: "GitHub 인증 관리",
	}
	cmd.AddCommand(loginCmd, statusCmd, logoutCmd)
	return cmd
}

func init() {
	rootCmd.AddCommand(authCmd)
}
