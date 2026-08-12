package handoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const SchemaVersion = 1

type Repository struct {
	Path      string `json:"path"`
	Branch    string `json:"branch,omitempty"`
	Commit    string `json:"commit,omitempty"`
	GitStatus string `json:"git_status,omitempty"`
}

type Document struct {
	SchemaVersion   int          `json:"schema_version"`
	Goal            string       `json:"goal"`
	SuccessCriteria []string     `json:"success_criteria"`
	Decisions       []string     `json:"decisions"`
	Constraints     []string     `json:"constraints"`
	Completed       []string     `json:"completed"`
	CurrentState    string       `json:"current_state"`
	Repositories    []Repository `json:"repositories,omitempty"`
	Validation      []string     `json:"validation"`
	Blockers        []string     `json:"blockers,omitempty"`
	Risks           []string     `json:"risks,omitempty"`
	NextSteps       []string     `json:"next_steps"`
}

func Parse(data []byte) (Document, error) {
	var d Document
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return d, fmt.Errorf("decode handoff: %w", err)
	}
	if err := requireJSONEOF(dec); err != nil {
		return d, err
	}
	if err := d.Validate(); err != nil {
		return d, err
	}
	return d, nil
}

func requireJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("handoff contains trailing JSON data")
		}
		return fmt.Errorf("decode trailing handoff data: %w", err)
	}
	return nil
}

func (d Document) Validate() error {
	if d.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", d.SchemaVersion)
	}
	if strings.TrimSpace(d.Goal) == "" || strings.TrimSpace(d.CurrentState) == "" {
		return errors.New("goal and current_state are required")
	}
	if len(d.SuccessCriteria) == 0 || len(d.NextSteps) == 0 {
		return errors.New("success_criteria and next_steps must not be empty")
	}
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if len(b) > 512*1024 {
		return errors.New("handoff document exceeds 512 KiB")
	}
	if findings := DetectSecrets(string(b)); len(findings) > 0 {
		return fmt.Errorf("possible secret detected: %s", strings.Join(findings, ", "))
	}
	return nil
}

var secretPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"private key", regexp.MustCompile(`(?i)-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
	{"OpenAI-style key", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`)},
	{"GitHub token", regexp.MustCompile(`\b(?:ghp|github_pat)_[A-Za-z0-9_]{20,}\b`)},
	{"authorization token", regexp.MustCompile(`(?i)\b(?:authorization|bearer)\s*[:= ]\s*[A-Za-z0-9._~+/=-]{20,}`)},
	{"password assignment", regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|api[_-]?key|secret|token)\s*[:=]\s*[^\s,;]{8,}`)},
	{"credential URI", regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s/:]+:[^\s/@]+@[^\s]+`)},
}

func DetectSecrets(text string) []string {
	var findings []string
	for _, p := range secretPatterns {
		if p.re.MatchString(text) {
			findings = append(findings, p.name)
		}
	}
	return findings
}
