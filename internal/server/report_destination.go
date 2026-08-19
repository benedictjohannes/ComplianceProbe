package server

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"
)

// isValidHeaderFieldName checks if a string is a valid HTTP header field name (RFC 7230 token).
func isValidHeaderFieldName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i := 0; i < len(name); i++ {
		b := name[i]
		if isTokenChar(b) {
			continue
		}
		return false
	}
	return true
}

func isTokenChar(b byte) bool {
	// RFC 7230 section 3.2.6 token character set:
	// DIGIT / ALPHA / "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." / "^" / "_" / "`" / "|" / "~"
	if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
		return true
	}
	switch b {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}

// isValidHeaderFieldValue checks if a string is a valid HTTP header field value (no control chars / CRLF).
func isValidHeaderFieldValue(val string) bool {
	for i := 0; i < len(val); {
		r, size := utf8.DecodeRuneInString(val[i:])
		if r == utf8.RuneError && size == 1 {
			return false
		}
		// Disallow control characters (0x00-0x08, 0x0A-0x1F, 0x7F)
		// Space (0x20) and tab (0x09) are permitted
		if (r < 0x20 && r != 0x09) || r == 0x7f {
			return false
		}
		i += size
	}
	return true
}

// Validate checks the validity of an HttpsDestinationConfig.
func (c *HttpsDestinationConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("https configuration is required")
	}

	trimmedURL := strings.TrimSpace(c.URL)
	if trimmedURL == "" {
		return fmt.Errorf("https destination URL is required")
	}

	parsed, err := url.ParseRequestURI(trimmedURL)
	if err != nil {
		return fmt.Errorf("invalid https destination URL: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("HTTPS destination URL must start with https://")
	}

	if parsed.Host == "" {
		return fmt.Errorf("https destination URL must contain a valid host")
	}

	if c.Format != "" && c.Format != "json" && c.Format != "multipart" {
		return fmt.Errorf("https format must be 'json' or 'multipart'")
	}

	for k, v := range c.Headers {
		if !isValidHeaderFieldName(k) {
			return fmt.Errorf("invalid HTTP header name: %q", k)
		}
		if !isValidHeaderFieldValue(v) {
			return fmt.Errorf("invalid HTTP header value for header %q", k)
		}
	}

	return nil
}

// Validate checks the validity of DestinationUpdateRequest fields.
func (req DestinationUpdateRequest) Validate() error {
	if req.FolderSource != nil {
		switch *req.FolderSource {
		case FolderSourceDefault, FolderSourceCLI, FolderSourcePlaybook, FolderSourceCustom, FolderSourceOff:
			// valid
		default:
			return fmt.Errorf("invalid folder_source: %q (must be default, cli, playbook, custom, or off)", *req.FolderSource)
		}

		if *req.FolderSource == FolderSourceCustom {
			if req.Folder == nil || strings.TrimSpace(*req.Folder) == "" {
				return fmt.Errorf("custom folder path cannot be empty")
			}
		}
	}

	if req.Folder != nil && strings.TrimSpace(*req.Folder) != "" {
		// If folder is specified and folder_source is custom (or not specified, but changing folder),
		// verify the directory exists and is a directory
		if req.FolderSource == nil || *req.FolderSource == FolderSourceCustom {
			resolved := resolveFolderPath(*req.Folder)
			info, err := os.Stat(resolved)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("custom folder path does not exist: %s", resolved)
				}
				return fmt.Errorf("cannot access custom folder path: %w", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("custom folder path is not a directory: %s", resolved)
			}
		}
	}

	if req.HttpsSource != nil {
		switch *req.HttpsSource {
		case HttpsSourcePlaybook, HttpsSourceCustom, HttpsSourceOff:
			// valid
		default:
			return fmt.Errorf("invalid https_source: %q (must be playbook, custom, or off)", *req.HttpsSource)
		}

		if *req.HttpsSource == HttpsSourceCustom {
			if req.HTTPS == nil {
				return fmt.Errorf("https configuration is required when https_source is 'custom'")
			}
			if err := req.HTTPS.Validate(); err != nil {
				return err
			}
		}
	} else if req.HTTPS != nil {
		// Partial update with HTTPS config provided
		if err := req.HTTPS.Validate(); err != nil {
			return err
		}
	}

	return nil
}
