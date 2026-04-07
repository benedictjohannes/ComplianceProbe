package playbook

import (
	"encoding/json"
	"testing"
)

func TestGenerateSchema(t *testing.T) {
	schema, err := GenerateSchema()
	if err != nil {
		t.Fatalf("GenerateSchema() error = %v", err)
	}

	if schema == "" {
		t.Error("GenerateSchema() returned empty string")
	}

	// Validate it is valid JSON
	var js map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &js); err != nil {
		t.Errorf("GenerateSchema() returned invalid JSON: %v", err)
	}

	// Basic check for expected fields
	if _, ok := js["$schema"]; !ok {
		t.Error("GenerateSchema() output missing $schema field")
	}
	if js["type"] != "object" {
		t.Errorf("GenerateSchema() output type = %v, want object", js["type"])
	}
}
