package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"ppm/internal/apperr"
	"ppm/internal/pkg"
)

// TarArchiver는 .tar.gz용 pkg.Archiver 구현체입니다.
type TarArchiver struct{}

// TarArchiver가 pkg.Archiver를 구현하는지 확인
var _ pkg.Archiver = (*TarArchiver)(nil)

// Extract는 리더의 .tar.gz 아카이브를 destDir로 풉니다.
func (a *TarArchiver) Extract(r io.Reader, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create dest directory")
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "failed to create gzip reader")
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	dirsCreated := make(map[string]bool)
	dirsCreated[destDir] = true

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apperr.Wrap(apperr.CodeArchive, err, "failed to read tar header")
		}

		target, err := archiveTargetPath(destDir, header.Name)
		if err != nil {
			return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive path %q", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := ensureNoSymlinkPath(destDir, target); err != nil {
				return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive directory %q", header.Name)
			}
			if !dirsCreated[target] {
				if err := os.MkdirAll(target, 0755); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create dir in archive")
				}
				dirsCreated[target] = true
			}
		case tar.TypeReg:
			if err := ensureNoSymlinkPath(destDir, target); err != nil {
				return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive file %q", header.Name)
			}
			// 상위 디렉터리 보장
			parent := filepath.Dir(target)
			if !dirsCreated[parent] {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create parent dir")
				}
				dirsCreated[parent] = true
			}

			// tar 헤더의 모드값 사용
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create file")
			}

			bw := bufio.NewWriterSize(outFile, 64*1024)
			n, copyErr := io.Copy(bw, io.LimitReader(tr, maxExtractedFileSize+1))
			if copyErr != nil {
				outFile.Close()
				return apperr.Wrap(apperr.CodeFileSystem, copyErr, "failed to write file")
			}
			if n > maxExtractedFileSize {
				outFile.Close()
				return apperr.New(apperr.CodeArchive, "archive member %q exceeds the maximum extracted file size", header.Name)
			}
			if err := bw.Flush(); err != nil {
				outFile.Close()
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to flush file")
			}
			outFile.Close()
		case tar.TypeSymlink:
			// 링크 자체와 링크가 가리키는 경로 모두 추출 디렉터리 안에 있어야 합니다.
			if runtime.GOOS != "windows" {
				if err := ensureNoSymlinkPath(destDir, target); err != nil {
					return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive symlink %q", header.Name)
				}
				linkTarget := filepath.Join(filepath.Dir(target), header.Linkname)
				if err := ensureWithinRoot(destDir, linkTarget); err != nil {
					return apperr.Wrap(apperr.CodeArchive, err, "unsafe archive symlink target %q", header.Linkname)
				}
				if err := os.Symlink(header.Linkname, target); err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create symlink %s", header.Name)
				}
			}
		}
	}
	return nil
}

func archiveTargetPath(root, name string) (string, error) {
	targetAbs, err := filepath.Abs(filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	if err := ensureWithinRoot(root, targetAbs); err != nil {
		return "", err
	}
	return targetAbs, nil
}

func ensureWithinRoot(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes destination directory")
	}
	return nil
}

func ensureNoSymlinkPath(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return err
	}
	current := rootAbs
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path contains symbolic link: %s", current)
		}
	}
	return nil
}

// Link는 심볼릭 링크를 만들거나 바이너리를 복사합니다.
func (a *TarArchiver) Link(extractedDir, binName, targetLinkPath string) error {
	return findAndLinkExecutable(extractedDir, binName, targetLinkPath)
}
