package headerflags

import (
	"strings"
)

// HeaderFlags implements flag.Value to parse multiple HTTP headers from CLI.
type HeaderFlags []string

func (h *HeaderFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *HeaderFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

// ToMap converts the slice of "Key: Value" strings into a map.
func (h *HeaderFlags) ToMap() map[string]string {
	headers := make(map[string]string)
	for _, val := range *h {
		parts := strings.SplitN(val, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}
