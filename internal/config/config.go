package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const DefaultPort = 47128

func Dir() (string, error) {
	if override := os.Getenv("CODEX_HANDOFF_HOME"); override != "" {
		return override, nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "codex-lan-handoff"), nil
}

func Ensure() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	for _, p := range []string{d, filepath.Join(d, "inbox"), filepath.Join(d, "replay")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			return "", err
		}
	}
	return d, nil
}

func KeyPath() (string, error) {
	d, err := Ensure()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "shared.key"), nil
}

func GenerateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	return encoded, SaveKey(encoded)
}

func SaveKey(encoded string) error {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(b) != 32 {
		return errors.New("shared key must be a base64url-encoded 32-byte value")
	}
	p, err := KeyPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(strings.TrimSpace(encoded)+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Chmod(p, 0600); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		identityOut, err := exec.Command("whoami").Output()
		if err != nil {
			return fmt.Errorf("resolve Windows identity: %w", err)
		}
		identity := strings.TrimSpace(string(identityOut))
		if identity == "" {
			return errors.New("resolve Windows identity: empty result")
		}
		cmd := exec.Command("icacls", p, "/inheritance:r", "/grant:r", identity+":(R,W)")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("restrict Windows key ACL: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func LoadKey() ([]byte, error) {
	p, err := KeyPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read shared key (run pair init/import first): %w", err)
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(b) != 32 {
		return nil, errors.New("stored shared key is invalid")
	}
	return b, nil
}
