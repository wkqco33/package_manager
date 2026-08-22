package pkg

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "checksum-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("archive contents"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256([]byte("archive contents"))
	checksum := fmt.Sprintf("sha256:%x", sum)
	if err := verifyChecksum(file, checksum); err != nil {
		t.Fatalf("verifyChecksum failed for valid checksum: %v", err)
	}

	if got, err := file.Stat(); err != nil || got.Size() != int64(len("archive contents")) {
		t.Fatalf("checksum verification changed archive: stat=%v err=%v", got, err)
	}
}

func TestVerifyChecksumRejectsMismatchAndMalformedValues(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "checksum-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("archive contents"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	tests := []string{
		strings.Repeat("0", sha256.Size*2),
		"not-a-checksum",
	}
	for _, checksum := range tests {
		t.Run(checksum, func(t *testing.T) {
			if err := verifyChecksum(file, checksum); err == nil {
				t.Fatal("expected checksum verification to fail")
			}
			if _, err := file.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
		})
	}
}
