package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/zhiaiyrb/codex-lan-handoff/internal/config"
	"github.com/zhiaiyrb/codex-lan-handoff/internal/protocol"
	"github.com/zhiaiyrb/codex-lan-handoff/internal/store"
)

type ReceiveOptions struct {
	FromIP  net.IP
	Port    int
	Timeout time.Duration
}

func Receive(ctx context.Context, key []byte, opts ReceiveOptions) (string, string, error) {
	if opts.FromIP == nil {
		return "", "", errors.New("a valid --from IP is required")
	}
	if opts.Port == 0 {
		opts.Port = config.DefaultPort
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort("", strconv.Itoa(opts.Port)))
	if err != nil {
		return "", "", err
	}
	defer ln.Close()
	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = ln.Close()
		case <-stopClose:
		}
	}()
	defer close(stopClose)
	deadline := time.Now().Add(opts.Timeout)
	if dl, ok := ln.(*net.TCPListener); ok {
		_ = dl.SetDeadline(deadline)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return "", "", ctx.Err()
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return "", "", errors.New("receive timed out")
			}
			return "", "", err
		}
		path, id, accepted := handleConnection(conn, key, opts.FromIP, deadline)
		if accepted {
			return path, id, nil
		}
		if time.Now().After(deadline) {
			return "", "", errors.New("receive timed out")
		}
	}
}

func handleConnection(conn net.Conn, key []byte, allowed net.IP, deadline time.Time) (string, string, bool) {
	defer conn.Close()
	_ = conn.SetDeadline(minTime(deadline, time.Now().Add(15*time.Second)))
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	remote := net.ParseIP(strings.Trim(host, "[]"))
	if err != nil || remote == nil || !remote.Equal(allowed) {
		_ = protocol.WriteFrame(conn, protocol.NewAck(key, false, "", "source IP not allowed"))
		return "", "", false
	}
	var e protocol.Envelope
	if err := protocol.ReadFrame(conn, &e); err != nil {
		_ = protocol.WriteFrame(conn, protocol.NewAck(key, false, "", "invalid frame"))
		return "", "", false
	}
	if seen, err := store.Seen(e.ID); err != nil || seen {
		_ = protocol.WriteFrame(conn, protocol.NewAck(key, false, e.ID, "replayed or invalid handoff"))
		return "", "", false
	}
	plain, err := protocol.Open(key, e, time.Now())
	if err != nil {
		_ = protocol.WriteFrame(conn, protocol.NewAck(key, false, e.ID, err.Error()))
		return "", "", false
	}
	path, err := store.Commit(e.ID, plain)
	if err != nil {
		_ = protocol.WriteFrame(conn, protocol.NewAck(key, false, e.ID, "could not commit handoff"))
		return "", "", false
	}
	if err := protocol.WriteFrame(conn, protocol.NewAck(key, true, e.ID, "")); err != nil {
		// The authenticated document is durable even when the sender misses the ACK.
	}
	return path, e.ID, true
}

func Send(ctx context.Context, key []byte, target string, port int, plaintext []byte) (string, error) {
	if net.ParseIP(target) == nil {
		return "", errors.New("--to must be an IP address")
	}
	if port == 0 {
		port = config.DefaultPort
	}
	e, err := protocol.Seal(key, plaintext, time.Now())
	if err != nil {
		return "", err
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(target, strconv.Itoa(port)))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	if err := protocol.WriteFrame(conn, e); err != nil {
		return "", err
	}
	var ack protocol.Ack
	if err := protocol.ReadFrame(conn, &ack); err != nil {
		return "", err
	}
	if err := protocol.VerifyAck(key, ack); err != nil {
		return "", err
	}
	if !ack.OK {
		return "", fmt.Errorf("receiver rejected handoff: %s", ack.Error)
	}
	if ack.ID != e.ID {
		return "", errors.New("receiver acknowledgement ID mismatch")
	}
	return e.ID, nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
