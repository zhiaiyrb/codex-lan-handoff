package config

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"
)

func TestSaveAndLoadKey(t *testing.T) {
	t.Setenv("CODEX_HANDOFF_HOME", t.TempDir())
	want := bytes.Repeat([]byte{42}, 32)
	encoded := base64.RawURLEncoding.EncodeToString(want)
	if err := SaveKey(encoded); err != nil {
		t.Fatal(err)
	}
	got, err := LoadKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("key mismatch")
	}
	p, _ := KeyPath()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 && os.PathSeparator != '\\' {
		t.Fatalf("key permissions too broad: %o", info.Mode().Perm())
	}
}

func TestInvalidKeyRejected(t *testing.T) {
	t.Setenv("CODEX_HANDOFF_HOME", t.TempDir())
	if err := SaveKey("not-a-key"); err == nil {
		t.Fatal("expected invalid key error")
	}
}
