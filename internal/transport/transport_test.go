package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/zhiaiyrb/codex-lan-handoff/internal/handoff"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

func TestLoopbackTransfer(t *testing.T) {
	t.Setenv("CODEX_HANDOFF_HOME", t.TempDir())
	key := make([]byte, 32)
	for i := range key {
		key[i] = 9
	}
	d := handoff.Document{SchemaVersion: 1, Goal: "g", SuccessCriteria: []string{"done"}, CurrentState: "s", NextSteps: []string{"next"}}
	b, _ := json.Marshal(d)
	port := freePort(t)
	type result struct {
		path, id string
		err      error
	}
	ch := make(chan result, 1)
	go func() {
		p, id, err := Receive(context.Background(), key, ReceiveOptions{FromIP: net.ParseIP("127.0.0.1"), Port: port, Timeout: 5 * time.Second})
		ch <- result{p, id, err}
	}()
	var id string
	var err error
	for i := 0; i < 20; i++ {
		id, err = Send(context.Background(), key, "127.0.0.1", port, b)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	r := <-ch
	if r.err != nil || r.id != id {
		t.Fatalf("receive: %#v", r)
	}
	got, err := os.ReadFile(r.path)
	if err != nil || string(got) != string(b) {
		t.Fatalf("file=%q err=%v", got, err)
	}
}

func TestWrongKeyRejectedUntilTimeout(t *testing.T) {
	t.Setenv("CODEX_HANDOFF_HOME", t.TempDir())
	port := freePort(t)
	key := make([]byte, 32)
	wrong := make([]byte, 32)
	wrong[0] = 1
	d := handoff.Document{SchemaVersion: 1, Goal: "g", SuccessCriteria: []string{"done"}, CurrentState: "s", NextSteps: []string{"next"}}
	b, _ := json.Marshal(d)
	ch := make(chan error, 1)
	go func() {
		_, _, err := Receive(context.Background(), key, ReceiveOptions{FromIP: net.ParseIP("127.0.0.1"), Port: port, Timeout: 500 * time.Millisecond})
		ch <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if _, err := Send(context.Background(), wrong, "127.0.0.1", port, b); err == nil {
		t.Fatal("expected rejection")
	}
	if err := <-ch; err == nil {
		t.Fatal("expected receive timeout")
	}
}

func TestReceiveCancellation(t *testing.T) {
	t.Setenv("CODEX_HANDOFF_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	port := freePort(t)
	ch := make(chan error, 1)
	go func() {
		_, _, err := Receive(ctx, make([]byte, 32), ReceiveOptions{FromIP: net.ParseIP("127.0.0.1"), Port: port, Timeout: time.Minute})
		ch <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-ch:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("receive did not stop after cancellation")
	}
}
