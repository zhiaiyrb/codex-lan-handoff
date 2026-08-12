package main

import (
	"errors"
	"strings"
	"testing"
)

func TestReadPairKeyStopsAtEnter(t *testing.T) {
	got, err := readPairKey(strings.NewReader("synthetic-key\nsecond-line"), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "synthetic-key" {
		t.Fatalf("got %q", got)
	}
}

func TestReadPairKeyAcceptsEOFWithoutNewline(t *testing.T) {
	got, err := readPairKey(strings.NewReader("synthetic-key"), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "synthetic-key" {
		t.Fatalf("got %q", got)
	}
}

func TestReadPairKeyUsesHiddenTerminalInput(t *testing.T) {
	called := false
	got, err := readPairKey(strings.NewReader("ignored"), true, func() ([]byte, error) {
		called = true
		return []byte(" hidden-key \r\n"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called || got != "hidden-key" {
		t.Fatalf("called=%v got=%q", called, got)
	}
}

func TestReadPairKeyReportsHiddenInputFailure(t *testing.T) {
	want := errors.New("terminal failed")
	_, err := readPairKey(nil, true, func() ([]byte, error) { return nil, want })
	if !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
}
