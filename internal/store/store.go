package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/zhiaiyrb/codex-lan-handoff/internal/config"
)

var validID = regexp.MustCompile(`^[a-f0-9]{32}$`)

func Seen(id string) (bool, error) {
	if !validID.MatchString(id) {
		return false, errors.New("invalid handoff id")
	}
	d, err := config.Ensure()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(d, "replay", id))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func Commit(id string, data []byte) (string, error) {
	if !validID.MatchString(id) {
		return "", errors.New("invalid handoff id")
	}
	d, err := config.Ensure()
	if err != nil {
		return "", err
	}
	if seen, err := Seen(id); err != nil || seen {
		if err != nil {
			return "", err
		}
		return "", errors.New("replayed handoff")
	}
	name := fmt.Sprintf("%s-%s.json", time.Now().UTC().Format("20060102T150405.000000000Z"), id)
	final := filepath.Join(d, "inbox", name)
	tmp, err := os.CreateTemp(filepath.Join(d, "inbox"), ".incoming-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, final); err != nil {
		return "", err
	}
	replay := filepath.Join(d, "replay", id)
	if err := os.WriteFile(replay, []byte(name+"\n"), 0600); err != nil {
		os.Remove(final)
		return "", err
	}
	return final, nil
}

func Latest() (string, []byte, error) {
	d, err := config.Ensure()
	if err != nil {
		return "", nil, err
	}
	entries, err := os.ReadDir(filepath.Join(d, "inbox"))
	if err != nil {
		return "", nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", nil, errors.New("inbox is empty")
	}
	sort.Strings(names)
	p := filepath.Join(d, "inbox", names[len(names)-1])
	b, err := os.ReadFile(p)
	return p, b, err
}
