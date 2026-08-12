package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/zhiaiyrb/codex-lan-handoff/internal/handoff"
)

func documentBytes(t *testing.T) []byte {
	t.Helper()
	d := handoff.Document{SchemaVersion: 1, Goal: "g", SuccessCriteria: []string{"done"}, CurrentState: "s", NextSteps: []string{"next"}}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSealOpen(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	now := time.Now().UTC()
	e, err := Seal(key, documentBytes(t), now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(key, e, now)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, documentBytes(t)) {
		t.Fatal("plaintext mismatch")
	}
}

func TestTamperAndExpiry(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	now := time.Now().UTC()
	e, _ := Seal(key, documentBytes(t), now)
	e.Ciphertext[0] ^= 1
	if _, err := Open(key, e, now); err == nil {
		t.Fatal("expected authentication failure")
	}
	e, _ = Seal(key, documentBytes(t), now.Add(-MaxAge-time.Second))
	if _, err := Open(key, e, now); err == nil {
		t.Fatal("expected expiry failure")
	}
}

func TestFrames(t *testing.T) {
	var b bytes.Buffer
	want := NewAck(bytes.Repeat([]byte{1}, 32), true, "abc", "")
	if err := WriteFrame(&b, want); err != nil {
		t.Fatal(err)
	}
	var got Ack
	if err := ReadFrame(&b, &got); err != nil {
		t.Fatal(err)
	}
	if got.OK != want.OK || got.ID != want.ID || !bytes.Equal(got.MAC, want.MAC) {
		t.Fatalf("got %#v", got)
	}
}

func TestAckAuthentication(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	a := NewAck(key, true, "abc", "")
	if err := VerifyAck(key, a); err != nil {
		t.Fatal(err)
	}
	a.OK = false
	if err := VerifyAck(key, a); err == nil {
		t.Fatal("expected tampered acknowledgement rejection")
	}
}

func TestFrameBounds(t *testing.T) {
	var b bytes.Buffer
	b.Write([]byte{0, 16, 0, 1})
	var got Ack
	if err := ReadFrame(&b, &got); err == nil {
		t.Fatal("expected oversized frame rejection")
	}
}
