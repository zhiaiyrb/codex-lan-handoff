package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zhiaiyrb/codex-lan-handoff/internal/config"
	"github.com/zhiaiyrb/codex-lan-handoff/internal/handoff"
	"github.com/zhiaiyrb/codex-lan-handoff/internal/store"
	"github.com/zhiaiyrb/codex-lan-handoff/internal/transport"
	"golang.org/x/term"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("command required")
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "setup":
		return setup()
	case "pair":
		return pair(args[1:])
	case "receive":
		return receive(args[1:])
	case "send":
		return send(args[1:])
	case "inbox":
		return inbox(args[1:])
	case "doctor":
		return doctor()
	case "help", "--help", "-h":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Print(`codex-lan-handoff - securely transfer structured task context over a LAN

Commands:
  setup
  pair init
  pair import [--key VALUE]        (paste securely and press Enter)
  receive --from IP [--port 47128] [--timeout 10m]
  send --to IP --file handoff.json [--port 47128]
  inbox latest
  doctor
  version
`)
}

func setup() error {
	d, err := config.Ensure()
	if err == nil {
		fmt.Println("Configuration directory:", d)
	}
	return err
}

func pair(args []string) error {
	if len(args) == 0 {
		return errors.New("pair requires init or import")
	}
	switch args[0] {
	case "init":
		key, err := config.GenerateKey()
		if err != nil {
			return err
		}
		fmt.Println("Shared key (shown once; import it on the peer, then clear your terminal):")
		fmt.Println(key)
		return nil
	case "import":
		fs := flag.NewFlagSet("pair import", flag.ContinueOnError)
		key := fs.String("key", "", "base64url shared key")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		value := strings.TrimSpace(*key)
		if value == "" {
			readValue, err := readPairKey(os.Stdin, term.IsTerminal(int(os.Stdin.Fd())), func() ([]byte, error) {
				fmt.Fprint(os.Stderr, "Paste the shared key and press Enter (input is hidden): ")
				b, readErr := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(os.Stderr)
				return b, readErr
			})
			if err != nil {
				return err
			}
			value = readValue
		}
		if err := config.SaveKey(value); err != nil {
			return err
		}
		fmt.Println("Shared key imported.")
		return nil
	default:
		return errors.New("pair requires init or import")
	}
}

func readPairKey(r io.Reader, interactive bool, readHidden func() ([]byte, error)) (string, error) {
	if interactive {
		b, err := readHidden()
		if err != nil {
			return "", fmt.Errorf("read shared key: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	// Read one line instead of waiting for EOF. This makes both `echo key | ...`
	// and interactive shells that are not detectable as terminals finish on Enter.
	line, err := bufio.NewReader(io.LimitReader(r, 4096)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read shared key: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func receive(args []string) error {
	fs := flag.NewFlagSet("receive", flag.ContinueOnError)
	from := fs.String("from", "", "allowed source IP")
	port := fs.Int("port", config.DefaultPort, "TCP port")
	timeout := fs.Duration("timeout", 10*time.Minute, "listen timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ip := net.ParseIP(*from)
	if ip == nil {
		return errors.New("--from must be a valid IP address")
	}
	key, err := config.LoadKey()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("Waiting on port %d for one handoff from %s (timeout %s)...\n", *port, ip, *timeout)
	path, id, err := transport.Receive(ctx, key, transport.ReceiveOptions{FromIP: ip, Port: *port, Timeout: *timeout})
	if err != nil {
		return err
	}
	fmt.Printf("Received handoff %s\nSaved: %s\n", id, path)
	return nil
}

func send(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	to := fs.String("to", "", "target IP")
	file := fs.String("file", "", "handoff JSON path")
	port := fs.Int("port", config.DefaultPort, "TCP port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("--file is required")
	}
	b, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	if _, err := handoff.Parse(b); err != nil {
		return err
	}
	key, err := config.LoadKey()
	if err != nil {
		return err
	}
	id, err := transport.Send(context.Background(), key, *to, *port, b)
	if err != nil {
		return err
	}
	fmt.Println("Handoff accepted:", id)
	return nil
}

func inbox(args []string) error {
	if len(args) != 1 || args[0] != "latest" {
		return errors.New("inbox requires latest")
	}
	p, b, err := store.Latest()
	if err != nil {
		return err
	}
	fmt.Printf("// Source: %s\n%s\n", p, b)
	return nil
}

func doctor() error {
	d, err := config.Ensure()
	if err != nil {
		return err
	}
	keyPath, _ := config.KeyPath()
	_, keyErr := config.LoadKey()
	fmt.Println("Version:", version)
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("Config:", d)
	if keyErr != nil {
		fmt.Println("Shared key: NOT READY -", keyErr)
	} else {
		fmt.Println("Shared key: ready at", keyPath)
	}
	fmt.Println("Default port:", config.DefaultPort)
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			host := strings.Split(a.String(), "/")[0]
			ip := net.ParseIP(host)
			if ip != nil && !ip.IsLoopback() {
				fmt.Printf("Address: %s (%s)\n", ip, iface.Name)
			}
		}
	}
	if runtime.GOOS == "windows" {
		fmt.Println("Firewall: allow inbound TCP", strconv.Itoa(config.DefaultPort), "on Private networks if receive is blocked")
	}
	if _, _, latestErr := store.Latest(); latestErr == nil {
		fmt.Println("Inbox: contains at least one handoff")
	} else {
		fmt.Println("Inbox: empty")
	}
	return nil
}
