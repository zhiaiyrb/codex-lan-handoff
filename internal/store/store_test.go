package store

import (
	"os"
	"testing"
)

func TestCommitAndReplay(t *testing.T) {
	t.Setenv("CODEX_HANDOFF_HOME", t.TempDir())
	id := "0123456789abcdef0123456789abcdef"
	p, err := Commit(id, []byte(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit(id, []byte("again")); err == nil {
		t.Fatal("expected replay rejection")
	}
	_, b, err := Latest()
	if err != nil || string(b) != `{"ok":true}` {
		t.Fatalf("Latest = %q, %v", b, err)
	}
}
