package executor

import (
	"encoding/json"
	"testing"
)

func TestAssertionStatus_String_Parse(t *testing.T) {
	tests := []struct {
		status AssertionStatus
		str    string
	}{
		{AssertionStatusPending, "pending"},
		{AssertionStatusRunning, "running"},
		{AssertionStatusPassed, "passed"},
		{AssertionStatusFailed, "failed"},
		{AssertionStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			if tt.status.String() != tt.str {
				t.Errorf("expected string %q, got %q", tt.str, tt.status.String())
			}

			parsed, err := ParseAssertionStatus(tt.str)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if parsed != tt.status {
				t.Errorf("expected parsed %v, got %v", tt.status, parsed)
			}

			// Test JSON marshal/unmarshal
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}
			if string(data) != `"`+tt.str+`"` {
				t.Errorf("expected JSON %q, got %q", `"`+tt.str+`"`, string(data))
			}

			var unmarshaled AssertionStatus
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if unmarshaled != tt.status {
				t.Errorf("expected unmarshaled %v, got %v", tt.status, unmarshaled)
			}
		})
	}

	// Unknown string parsing
	if _, err := ParseAssertionStatus("invalid"); err == nil {
		t.Error("expected error parsing invalid AssertionStatus")
	}

	// Invalid JSON unmarshal
	var s AssertionStatus
	if err := json.Unmarshal([]byte(`"invalid"`), &s); err == nil {
		t.Error("expected error unmarshaling invalid AssertionStatus JSON")
	}
	if err := json.Unmarshal([]byte(`123`), &s); err == nil {
		t.Error("expected error unmarshaling non-string JSON")
	}
}

func TestCommandStatus_String_Parse(t *testing.T) {
	tests := []struct {
		status CommandStatus
		str    string
	}{
		{CommandStatusPending, "pending"},
		{CommandStatusRunning, "running"},
		{CommandStatusCompleted, "completed"},
		{CommandStatusFailed, "failed"},
		{CommandStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			if tt.status.String() != tt.str {
				t.Errorf("expected string %q, got %q", tt.str, tt.status.String())
			}

			parsed, err := ParseCommandStatus(tt.str)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if parsed != tt.status {
				t.Errorf("expected parsed %v, got %v", tt.status, parsed)
			}

			// Test JSON marshal/unmarshal
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}
			if string(data) != `"`+tt.str+`"` {
				t.Errorf("expected JSON %q, got %q", `"`+tt.str+`"`, string(data))
			}

			var unmarshaled CommandStatus
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if unmarshaled != tt.status {
				t.Errorf("expected unmarshaled %v, got %v", tt.status, unmarshaled)
			}
		})
	}

	// Unknown string parsing
	if _, err := ParseCommandStatus("invalid"); err == nil {
		t.Error("expected error parsing invalid CommandStatus")
	}

	// Invalid JSON unmarshal
	var s CommandStatus
	if err := json.Unmarshal([]byte(`"invalid"`), &s); err == nil {
		t.Error("expected error unmarshaling invalid CommandStatus JSON")
	}
	if err := json.Unmarshal([]byte(`123`), &s); err == nil {
		t.Error("expected error unmarshaling non-string JSON")
	}
}
