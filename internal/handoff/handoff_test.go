package handoff

import (
	"encoding/json"
	"testing"
)

func validDocument() Document {
	return Document{SchemaVersion: 1, Goal: "Continue implementation", SuccessCriteria: []string{"Tests pass"}, CurrentState: "Partially implemented", NextSteps: []string{"Run tests"}}
}

func TestDocumentRoundTrip(t *testing.T) {
	b, _ := json.Marshal(validDocument())
	d, err := Parse(b)
	if err != nil || d.Goal != "Continue implementation" {
		t.Fatalf("Parse() = %#v, %v", d, err)
	}
}

func TestSecretDetection(t *testing.T) {
	d := validDocument()
	d.CurrentState = "password=super-secret-value"
	if err := d.Validate(); err == nil {
		t.Fatal("expected secret validation error")
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	b := []byte(`{"schema_version":1,"goal":"g","current_state":"s","success_criteria":["x"],"next_steps":["x"],"raw_chat":"no"}`)
	if _, err := Parse(b); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestTrailingJSONRejected(t *testing.T) {
	b, _ := json.Marshal(validDocument())
	if _, err := Parse(append(b, []byte(` {}`)...)); err == nil {
		t.Fatal("expected trailing JSON error")
	}
}
