package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveDirFromLinkTarget(t *testing.T) {
	packagesDir := filepath.Join("home", ".config", "ppm", "packages")

	// packagesDir 내부를 가리키는 링크 → 직계 하위 디렉터리명 반환
	inside := filepath.Join(packagesDir, "repo-v1.0.0", "repo")
	if got := activeDirFromLinkTarget(packagesDir, inside); got != "repo-v1.0.0" {
		t.Errorf("내부 링크 대상에서 repo-v1.0.0을 기대했으나 %q", got)
	}

	// packagesDir 외부를 가리키는 링크 → 빈 문자열
	outside := filepath.Join("home", "other", "bin", "repo")
	if got := activeDirFromLinkTarget(packagesDir, outside); got != "" {
		t.Errorf("외부 링크 대상에서 빈 문자열을 기대했으나 %q", got)
	}
}

func TestHashFileAndDirContainsMatchingFile(t *testing.T) {
	// 설치 바이너리(원본) 생성
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "installed")
	content := []byte("identical-binary-content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(srcFile)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hashFile(srcFile)
	if err != nil {
		t.Fatalf("hashFile failed: %v", err)
	}

	// 동일 내용 파일을 (중첩 디렉터리에) 가진 패키지 디렉터리 → 매칭
	matchDir := t.TempDir()
	nested := filepath.Join(matchDir, "sub")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "renamed"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if !dirContainsMatchingFile(matchDir, info.Size(), hash) {
		t.Error("동일 내용 파일을 가진 디렉터리는 매칭되어야 합니다")
	}

	// 다른 내용 파일만 가진 디렉터리 → 미매칭
	noMatchDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(noMatchDir, "other"), []byte("different content!!"), 0644); err != nil {
		t.Fatal(err)
	}
	if dirContainsMatchingFile(noMatchDir, info.Size(), hash) {
		t.Error("다른 내용 디렉터리는 매칭되지 않아야 합니다")
	}
}

// writeVersionDir는 packagesDir 하위에 버전 디렉터리와 바이너리(지정 내용)를 생성합니다.
func writeVersionDir(t *testing.T, packagesDir, dirName, content string) {
	t.Helper()
	dir := filepath.Join(packagesDir, dirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "repo"), []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

// TestCollectActiveDirs_Copy는 Windows 복사본 시나리오(심볼릭 링크 없음)에서
// 설치된 바이너리와 내용이 같은 버전만 활성으로 감지되는지 검증합니다.
// (이전 버그: readlink 전용 로직이라 Windows에서 활성 버전을 못 찾아 전부 삭제됨)
func TestCollectActiveDirs_Copy(t *testing.T) {
	packagesDir := t.TempDir()
	writeVersionDir(t, packagesDir, "repo-v1.0.0", "current-binary")
	writeVersionDir(t, packagesDir, "repo-v0.9.0", "old-binary")

	// installPath에 현재 버전 바이너리의 복사본 배치
	installPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(installPath, "repo"), []byte("current-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	pkgEntries, err := os.ReadDir(packagesDir)
	if err != nil {
		t.Fatal(err)
	}

	active := collectActiveDirs(packagesDir, installPath, pkgEntries)
	if !active["repo-v1.0.0"] {
		t.Error("현재 설치된 버전(repo-v1.0.0)이 활성으로 감지되어야 합니다")
	}
	if active["repo-v0.9.0"] {
		t.Error("구버전(repo-v0.9.0)은 활성이 아니어야 합니다")
	}
}

// TestPerformCleanUnused_PreservesActiveVersion은 clean이 활성 버전을 보존하고
// 구버전만 삭제하는지 종단 검증합니다.
func TestPerformCleanUnused_PreservesActiveVersion(t *testing.T) {
	packagesDir := t.TempDir()
	writeVersionDir(t, packagesDir, "repo-v1.0.0", "current-binary")
	writeVersionDir(t, packagesDir, "repo-v0.9.0", "old-binary")

	installPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(installPath, "repo"), []byte("current-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	performCleanUnused(packagesDir, installPath)

	if _, err := os.Stat(filepath.Join(packagesDir, "repo-v1.0.0")); err != nil {
		t.Errorf("활성 버전이 보존되어야 합니다: %v", err)
	}
	if _, err := os.Stat(filepath.Join(packagesDir, "repo-v0.9.0")); !os.IsNotExist(err) {
		t.Error("구버전이 삭제되어야 합니다")
	}
}

// TestCollectActiveDirs_Symlink는 Unix 심볼릭 링크 시나리오를 검증합니다.
func TestCollectActiveDirs_Symlink(t *testing.T) {
	packagesDir := t.TempDir()
	writeVersionDir(t, packagesDir, "repo-v1.0.0", "current-binary")
	writeVersionDir(t, packagesDir, "repo-v0.9.0", "old-binary")

	installPath := t.TempDir()
	linkPath := filepath.Join(installPath, "repo")
	target := filepath.Join(packagesDir, "repo-v1.0.0", "repo")
	if err := os.Symlink(target, linkPath); err != nil {
		// Windows 등 심볼릭 링크 생성 권한이 없으면 건너뜀
		t.Skipf("심볼릭 링크를 생성할 수 없어 건너뜁니다: %v", err)
	}

	pkgEntries, err := os.ReadDir(packagesDir)
	if err != nil {
		t.Fatal(err)
	}

	active := collectActiveDirs(packagesDir, installPath, pkgEntries)
	if !active["repo-v1.0.0"] {
		t.Error("심볼릭 링크가 가리키는 버전이 활성으로 감지되어야 합니다")
	}
	if active["repo-v0.9.0"] {
		t.Error("구버전은 활성이 아니어야 합니다")
	}
}
