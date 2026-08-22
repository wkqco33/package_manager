package pkg

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("signed archive")
	path := t.TempDir() + "/archive.bin"
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	signature := ed25519.Sign(privateKey, content)
	if err := verifySignature(file, base64.StdEncoding.EncodeToString(signature), base64.StdEncoding.EncodeToString(publicKey)); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := verifySignature(file, base64.StdEncoding.EncodeToString([]byte("bad")), base64.StdEncoding.EncodeToString(publicKey)); err == nil {
		t.Fatal("invalid signature accepted")
	}
}
