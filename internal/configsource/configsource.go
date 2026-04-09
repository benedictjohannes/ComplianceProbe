package configsource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/benedictjohannes/crobe/playbook"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads the playbook from either a local file or an HTTPS URL.
func LoadConfig(path string, headers map[string]string) (*playbook.Playbook, []byte, error) {
	var data []byte
	var contentType string
	var err error

	if strings.HasPrefix(path, "http://") {
		return nil, nil, fmt.Errorf("insecure HTTP connections are not allowed: %s", path)
	}
	isHttps := strings.HasPrefix(path, "https://")
	if isHttps {
		data, contentType, err = fetchHttpsPlaybook(path, headers)
	} else {
		data, err = os.ReadFile(path)
	}

	if err != nil {
		return nil, nil, err
	}

	var config playbook.Playbook
	isJson := strings.HasPrefix(strings.ToLower(contentType), "application/json") ||
		strings.HasSuffix(strings.ToLower(path), ".json") && !isHttps

	if isJson {
		err = json.Unmarshal(data, &config)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	} else {
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	}

	return &config, data, nil
}

func fetchHttpsPlaybook(url string, headers map[string]string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch remote playbook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch remote playbook: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return data, resp.Header.Get("Content-Type"), nil
}
