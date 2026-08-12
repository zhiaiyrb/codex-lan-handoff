package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/zhiaiyrb/codex-lan-handoff/internal/handoff"
)

const (
	Version      = 1
	MaxFrameSize = 1024 * 1024
	MaxAge       = 15 * time.Minute
)

type Envelope struct {
	Version    int    `json:"version"`
	ID         string `json:"id"`
	CreatedAt  string `json:"created_at"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

type Ack struct {
	OK    bool   `json:"ok"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
	MAC   []byte `json:"mac"`
}

func NewAck(key []byte, ok bool, id, message string) Ack {
	a := Ack{OK: ok, ID: id, Error: message}
	a.MAC = ackMAC(key, a)
	return a
}

func VerifyAck(key []byte, a Ack) error {
	if len(a.MAC) != sha256.Size || !hmac.Equal(a.MAC, ackMAC(key, a)) {
		return errors.New("acknowledgement authentication failed")
	}
	return nil
}

func ackMAC(key []byte, a Ack) []byte {
	h := hmac.New(sha256.New, key)
	fmt.Fprintf(h, "codex-lan-handoff-ack|%t|%s|%s", a.OK, a.ID, a.Error)
	return h.Sum(nil)
}

func Seal(key, plaintext []byte, now time.Time) (Envelope, error) {
	if _, err := handoff.Parse(plaintext); err != nil {
		return Envelope{}, err
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return Envelope{}, err
	}
	e := Envelope{Version: Version, ID: hex.EncodeToString(idBytes), CreatedAt: now.UTC().Format(time.RFC3339)}
	aead, err := newAEAD(key)
	if err != nil {
		return Envelope{}, err
	}
	e.Nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(e.Nonce); err != nil {
		return Envelope{}, err
	}
	e.Ciphertext = aead.Seal(nil, e.Nonce, plaintext, aad(e))
	return e, nil
}

func Open(key []byte, e Envelope, now time.Time) ([]byte, error) {
	if e.Version != Version || len(e.ID) != 32 {
		return nil, errors.New("unsupported or malformed envelope")
	}
	created, err := time.Parse(time.RFC3339, e.CreatedAt)
	if err != nil || now.UTC().Sub(created) > MaxAge || created.Sub(now.UTC()) > time.Minute {
		return nil, errors.New("handoff envelope is expired or has an invalid timestamp")
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	if len(e.Nonce) != aead.NonceSize() {
		return nil, errors.New("invalid nonce")
	}
	plain, err := aead.Open(nil, e.Nonce, e.Ciphertext, aad(e))
	if err != nil {
		return nil, errors.New("authentication failed")
	}
	if _, err := handoff.Parse(plain); err != nil {
		return nil, err
	}
	return plain, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256 requires a 32-byte key")
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

func aad(e Envelope) []byte {
	return []byte(fmt.Sprintf("codex-lan-handoff|%d|%s|%s", e.Version, e.ID, e.CreatedAt))
}

func WriteFrame(w io.Writer, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(b) > MaxFrameSize {
		return errors.New("frame exceeds 1 MiB")
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func ReadFrame(r io.Reader, value any) error {
	var n uint32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return err
	}
	if n == 0 || n > MaxFrameSize {
		return errors.New("invalid frame size")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("frame contains trailing JSON data")
		}
		return err
	}
	return nil
}
