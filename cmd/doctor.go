package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/platform"
)

var doctorCmd = &wcli.Command{
	Use:   "doctor",
	Short: "ppm 설치 상태 진단",
	Run: func(ctx *wcli.Context) error {
		failures := 0
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("FAIL config: %v\n", err)
			return err
		}
		fmt.Println("ppm 진단 결과")
		fmt.Printf("  OK   registry: %s\n", cfg.RegistryURL)
		if cfg.AuthToken == "" {
			fmt.Println("  WARN auth_token이 설정되지 않았습니다")
		} else {
			fmt.Println("  OK   auth_token이 설정되어 있습니다")
		}
		if err := checkDirectory("install path", cfg.InstallPath); err != nil {
			fmt.Printf("  FAIL %v\n", err)
			failures++
		} else {
			fmt.Printf("  OK   install path: %s\n", cfg.InstallPath)
		}
		if !pathContains(cfg.InstallPath) {
			fmt.Printf("  WARN install path가 PATH에 없습니다: %s\n", cfg.InstallPath)
		}
		packagesDir, err := config.GetPackagesDir()
		if err != nil {
			fmt.Printf("  FAIL packages path: %v\n", err)
			failures++
		} else if err := checkDirectory("packages path", packagesDir); err != nil {
			fmt.Printf("  WARN %v\n", err)
		} else {
			fmt.Printf("  OK   packages path: %s\n", packagesDir)
			if err := verifyPackages(packagesDir, cfg.InstallPath, nil); err != nil {
				fmt.Printf("  FAIL packages: %v\n", err)
				failures++
			}
		}
		cacheDir, err := config.GetCacheDir()
		if err == nil {
			if _, statErr := os.Stat(cacheDir); statErr == nil {
				fmt.Printf("  OK   cache path: %s\n", cacheDir)
			} else if os.IsNotExist(statErr) {
				fmt.Println("  OK   cache path: 아직 생성되지 않음")
			}
		}
		if failures > 0 {
			return fmt.Errorf("진단에서 %d개의 문제를 발견했습니다", failures)
		}
		return nil
	},
}

var verifyCmd = &wcli.Command{
	Use:   "verify [package...]",
	Short: "설치된 패키지의 메타데이터와 바이너리 검증",
	Run: func(ctx *wcli.Context) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		dir, err := config.GetPackagesDir()
		if err != nil {
			return err
		}
		if err := verifyPackages(dir, cfg.InstallPath, ctx.Args); err != nil {
			return err
		}
		return nil
	},
}

func checkDirectory(label, dir string) error {
	if dir == "" {
		return fmt.Errorf("%s가 비어 있습니다", label)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("%s를 확인할 수 없습니다: %w", label, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s가 디렉터리가 아닙니다: %s", label, dir)
	}
	return nil
}

func pathContains(dir string) bool {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if samePath(entry, dir) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
	}
	return filepath.Clean(aa) == filepath.Clean(bb)
}

func verifyPackages(packagesDir, installPath string, names []string) error {
	installed, err := pkg.ListInstalled(packagesDir)
	if err != nil {
		return err
	}
	wanted := make(map[string]bool)
	for _, name := range names {
		wanted[name] = true
	}
	failures := 0
	matched := make(map[string]bool)
	for _, p := range installed {
		if len(wanted) > 0 && !wanted[p.Name] && !wanted[filepath.Base(p.Name)] {
			continue
		}
		for name := range wanted {
			if name == p.Name || name == filepath.Base(p.Name) {
				matched[name] = true
			}
		}
		bin := p.BinName
		if bin == "" {
			bin = filepath.Base(p.Name)
		}
		binary := filepath.Join(installPath, platform.ExecutableName(bin))
		packageDir := filepath.Join(packagesDir, filepath.Base(p.Name)+"-"+p.Version)
		if _, err := os.Stat(packageDir); err != nil {
			fmt.Printf("FAIL %s: 패키지 디렉터리 없음\n", p.Name)
			failures++
			continue
		}
		if _, err := os.Stat(binary); err != nil {
			fmt.Printf("FAIL %s: 바이너리 없음 (%s)\n", p.Name, binary)
			failures++
			continue
		}
		if _, err := exec.LookPath(bin); err != nil && !pathContains(installPath) {
			fmt.Printf("WARN %s: PATH에서 바이너리를 찾을 수 없음\n", p.Name)
		}
		fmt.Printf("OK   %s %s\n", p.Name, p.Version)
	}
	for name := range wanted {
		if !matched[name] {
			fmt.Printf("FAIL %s: 설치된 패키지를 찾을 수 없음\n", name)
			failures++
		}
	}
	if len(installed) == 0 && len(names) == 0 {
		fmt.Println("OK   설치된 패키지가 없습니다")
	}
	if failures > 0 {
		return fmt.Errorf("%d개 패키지 검증 실패", failures)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(verifyCmd)
}
