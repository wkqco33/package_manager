package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
)

var cacheCmd = &wcli.Command{
	Use:   "cache",
	Short: "다운로드 캐시 관리",
}

var cacheListCmd = &wcli.Command{
	Use:   "list",
	Short: "캐시된 아카이브 목록 표시",
	Run: func(ctx *wcli.Context) error {
		dir, err := config.GetCacheDir()
		if err != nil {
			return err
		}
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			logger.Info("캐시된 파일이 없습니다.")
			return nil
		}
		if err != nil {
			return err
		}
		count := 0
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) == ".tmp" {
				continue
			}
			info, statErr := entry.Info()
			if statErr != nil {
				continue
			}
			fmt.Printf("  %s (%d bytes)\n", entry.Name(), info.Size())
			count++
		}
		if count == 0 {
			logger.Info("캐시된 파일이 없습니다.")
		}
		return nil
	},
}

var cacheCleanCmd = &wcli.Command{
	Use:   "clean",
	Short: "다운로드 캐시 삭제",
	Run: func(ctx *wcli.Context) error {
		dir, err := config.GetCacheDir()
		if err != nil {
			return err
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		logger.Success("다운로드 캐시를 삭제했습니다.")
		return nil
	},
}

func init() {
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	rootCmd.AddCommand(cacheCmd)
}
