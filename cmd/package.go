package cmd

import (
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/manifest"
)

// PackageGuide는 PACKAGE_GUIDE.md의 내용을 담는 변수입니다 (main에서 주입 가능).
var PackageGuide string

var packageCmd = newPackageCommand()

func newPackageCommand() *wcli.Command {
	cmd := &wcli.Command{
		Use:   "package",
		Short: "패키지 가이드 조회 및 패키지 개발 관리",
		Long:  `패키지 등록 및 배포 가이드(PACKAGE_GUIDE.md)를 확인하거나, 서브커맨드를 통해 프로젝트 매니페스트(ppm.json)를 생성 및 검증합니다.`,
		Run: func(ctx *wcli.Context) error {
			guide := PackageGuide
			if guide == "" {
				if data, err := os.ReadFile("PACKAGE_GUIDE.md"); err == nil {
					guide = string(data)
				}
			}
			if guide == "" {
				return fmt.Errorf("패키지 가이드(PACKAGE_GUIDE.md)를 찾을 수 없습니다")
			}
			fmt.Println(guide)
			return nil
		},
	}

	cmd.AddCommand(newPackageInitCommand())
	cmd.AddCommand(newPackageValidateCommand())
	return cmd
}

func newPackageInitCommand() *wcli.Command {
	var binName string
	var description string
	var author string
	var homepage string
	var force bool

	initCmd := &wcli.Command{
		Use:   "init [dir]",
		Short: "현재 프로젝트에 ppm.json 매니페스트 생성",
		Long:  `지정된 디렉토리(기본값: 현재 디렉토리)에 패키지 매니페스트(ppm.json)를 생성하고 초기 설정을 구성합니다.`,
		Run: func(ctx *wcli.Context) error {
			targetDir := "."
			if len(ctx.Args) > 0 {
				targetDir = ctx.Args[0]
			}

			opts := manifest.InitOptions{
				Dir:         targetDir,
				BinName:     binName,
				Description: description,
				Author:      author,
				Homepage:    homepage,
				Force:       force,
			}

			_, targetPath, err := manifest.Init(opts)
			if err != nil {
				return err
			}

			logger.Success("ppm.json 매니페스트가 성공적으로 생성되었습니다: %s", targetPath)
			logger.Info("패키지 유효성 검증: ppm package validate %s", targetPath)
			return nil
		},
	}

	initCmd.Flags().StringVar(&binName, "bin-name", "b", "", "생성될 바이너리/실행 파일 이름 (기본값: 디렉토리명)")
	initCmd.Flags().StringVar(&description, "description", "d", "", "패키지 요약 설명")
	initCmd.Flags().StringVar(&author, "author", "a", "", "작성자 또는 팀/조직명")
	initCmd.Flags().StringVar(&homepage, "homepage", "", "", "패키지 홈페이지 또는 레포지토리 URL")
	initCmd.Flags().BoolVar(&force, "force", "f", false, "이미 존재하는 ppm.json 덮어쓰기")

	return initCmd
}

func newPackageValidateCommand() *wcli.Command {
	return &wcli.Command{
		Use:   "validate [path]",
		Short: "ppm.json 유효성 검증",
		Long:  `ppm.json 매니페스트 파일의 구문 및 필수 필드 유효성을 검증합니다.`,
		Run: func(ctx *wcli.Context) error {
			path := "ppm.json"
			if len(ctx.Args) > 0 {
				path = ctx.Args[0]
			}

			m, err := manifest.Load(path)
			if err != nil {
				return err
			}

			if err := m.Validate(); err != nil {
				return err
			}

			logger.Success("manifest가 유효합니다: %s", path)
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(packageCmd)
}
