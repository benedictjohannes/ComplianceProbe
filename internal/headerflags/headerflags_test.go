package headerflags

import (
	"reflect"
	"testing"
)

func TestHeaderFlags_SetAndString(t *testing.T) {
	var h HeaderFlags

	if err := h.Set("Authorization: Bearer token"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := h.Set("X-Custom: value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	expectedString := "Authorization: Bearer token, X-Custom: value"
	if h.String() != expectedString {
		t.Errorf("String() = %q, want %q", h.String(), expectedString)
	}

	if len(h) != 2 {
		t.Errorf("len(h) = %d, want 2", len(h))
	}
}

func TestHeaderFlags_ToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:  "Standard headers",
			input: []string{"Content-Type: application/json", "Authorization: Bearer 123"},
			expected: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer 123",
			},
		},
		{
			name:  "Headers with whitespace",
			input: []string{"  X-Header  :  value  ", "Another:thing "},
			expected: map[string]string{
				"X-Header": "value",
				"Another":  "thing",
			},
		},
		{
			name:  "Colon in value",
			input: []string{"URL: https://example.com", "Time: 12:00"},
			expected: map[string]string{
				"URL":  "https://example.com",
				"Time": "12:00",
			},
		},
		{
			name:  "Invalid headers (no colon)",
			input: []string{"InvalidHeader", "Valid: Header"},
			expected: map[string]string{
				"Valid": "Header",
			},
		},
		{
			name:     "Empty",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:  "Duplicate keys (last one wins)",
			input: []string{"X-Same: first", "X-Same: second"},
			expected: map[string]string{
				"X-Same": "second",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HeaderFlags(tt.input)
			got := h.ToMap()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ToMap() = %v, want %v", got, tt.expected)
			}
		})
	}
}
