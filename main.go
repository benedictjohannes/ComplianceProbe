package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"gopkg.in/yaml.v3"
)

func main() {
	schemaFlag := flag.Bool("schema", false, "Output the configuration JSON schema and exit")
	flag.Parse()

	if *schemaFlag {
		generateSchema()
		return
	}

	now := time.Now()
	timestamp := now.Format("060102-150405")

	reportsDir := "reports"
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		os.MkdirAll(reportsDir, 0755)
	}

	reportBase := filepath.Join(reportsDir, timestamp+".report")
	logFile := reportBase + ".log"
	mdFile := reportBase + ".md"
	jsonFile := reportBase + ".json"

	config := loadConfig()

	// Validate unique codes
	codes := make(map[string]bool)
	for _, section := range config.Sections {
		for _, assertion := range section.Assertions {
			if assertion.Code == "" {
				fmt.Printf("❌ Assertion '%s' in section '%s' is missing a 'code'\n", assertion.Title, section.Title)
				os.Exit(1)
			}
			if codes[assertion.Code] {
				fmt.Printf("❌ Duplicate code found: %s\n", assertion.Code)
				os.Exit(1)
			}
			codes[assertion.Code] = true
		}
	}

	fmt.Printf("🚀 Starting Security Report Generation: %s\n", timestamp)
	fmt.Printf("📂 Using configuration: %s\n", getConfigPath())

	// Initialize Files
	os.WriteFile(logFile, []byte(fmt.Sprintf("=== SECURITY REPORT LOG: %s ===\n\n", timestamp)), 0644)

	mdHeader := fmt.Sprintf("---\ntitle: %s\ndate: %s\ngeometry: margin=2cm\n---\n\n", config.Title, now.Format("2006-01-02"))
	mdHeader += fmt.Sprintf("# %s\n\nGenerated on: %s\n\n---\n\n", config.Title, now.Format("2006-01-02 15:04:05"))
	os.WriteFile(mdFile, []byte(mdHeader), 0644)

	// Counters
	totalAssertions := 0
	passedAssertions := 0
	failedAssertions := 0
	reportJSON := make(map[string]bool)

	// Iterate Sections
	for _, section := range config.Sections {
		fmt.Printf("  Processing Section: %s\n", section.Title)

		appendToFile(mdFile, fmt.Sprintf("## %s\n\n", section.Title))
		for _, desc := range section.Description {
			appendToFile(mdFile, fmt.Sprintf("%s  \n", desc))
		}
		appendToFile(mdFile, "\n")

		for _, assertion := range section.Assertions {
			// 1. Pre-Commands
			for _, cmd := range assertion.PreCmd {
				runCommand(cmd, logFile)
			}

			// 2. Main Command
			allPassed := true
			var outputs []string
			for _, cmd := range assertion.Cmd {
				out, success := runCommand(cmd, logFile)
				outputs = append(outputs, out)
				if !success {
					allPassed = false
				}
			}

			// 3. Console Output (Status)
			totalAssertions++
			statusEmoji := "✅ PASS"
			if !allPassed {
				statusEmoji = "❌ FAIL"
				failedAssertions++
			} else {
				passedAssertions++
			}
			fmt.Printf("    - %s: %s\n", assertion.Title, statusEmoji)

			appendToFile(mdFile, fmt.Sprintf("### %s\n\n", assertion.Title))
			appendToFile(mdFile, fmt.Sprintf("%s\n\n", assertion.Description))

			fullOutput := strings.Join(outputs, "\n")
			if fullOutput == "" {
				fullOutput = "*(No output)*"
			}

			appendToFile(mdFile, "**Evidence:**\n\n")
			appendToFile(mdFile, "```bash\n")
			displayCmd := strings.Join(assertion.Cmd, " && ")
			appendToFile(mdFile, fmt.Sprintf("> %s\n\n", displayCmd))
			appendToFile(mdFile, fullOutput+"\n")
			appendToFile(mdFile, "```\n\n")

			// Results based on success
			if allPassed {
				if assertion.PassDescription != "" {
					appendToFile(mdFile, fmt.Sprintf("> ✅ **Pass:** %s\n\n", assertion.PassDescription))
				}
			} else {
				if assertion.FailDescription != "" {
					appendToFile(mdFile, fmt.Sprintf("> ❌ **Fail:** %s\n\n", assertion.FailDescription))
				}
			}

			// 4. Update report JSON
			reportJSON[assertion.Code] = allPassed

			// 5. Post-Commands
			for _, cmd := range assertion.PostCmd {
				runCommand(cmd, logFile)
			}
		}
		appendToFile(mdFile, "---\n\n")
	}

	appendToFile(mdFile, "\n\n*End of Report*\n")

	fmt.Println("\n✅ Generation Complete!")
	fmt.Printf("📊 Checks: %d assertions\n", totalAssertions)
	fmt.Printf("   ✅ PASS: %d\n", passedAssertions)
	fmt.Printf("   ❌ FAIL: %d\n", failedAssertions)
	fmt.Printf("\n📄 Log: %s\n", logFile)
	fmt.Printf("📝 Markdown: %s\n", mdFile)

	// Save report.json
	jsonBytes, err := json.MarshalIndent(reportJSON, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to marshal report.json: %v\n", err)
	} else {
		err = os.WriteFile(jsonFile, jsonBytes, 0644)
		if err != nil {
			fmt.Printf("❌ Failed to write %s: %v\n", jsonFile, err)
		} else {
			fmt.Println("📊 JSON Report: " + jsonFile)
		}
	}
}

func loadConfig() ReportConfig {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to read config %s: %v\n", configPath, err)
		os.Exit(1)
	}

	var config ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	return config
}

func generateSchema() {
	reflector := &jsonschema.Reflector{
		DoNotReference: false,
		ExpandedStruct: true,
	}

	s := reflector.Reflect(&ReportConfig{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to generate schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func getConfigPath() string {
	// If a non-flag argument is provided, use it as the config path
	args := flag.Args()
	if len(args) > 0 {
		return args[0]
	}
	return "playbook.yaml"
}

func runCommand(command string, logFile string) (string, bool) {
	var name string
	var args []string

	tmpDir := os.TempDir()
	var tmpFile string

	if runtime.GOOS == "windows" {
		name = "powershell"
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cmd_%d.ps1", time.Now().UnixNano()))
		os.WriteFile(tmpFile, []byte(command), 0644)
		args = []string{"-ExecutionPolicy", "Bypass", "-File", tmpFile}
	} else {
		name = "bash"
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cmd_%d.sh", time.Now().UnixNano()))
		// Add shebang, pipefail, and the command
		script := fmt.Sprintf("#!/bin/bash\nset -o pipefail\n%s\n", command)
		os.WriteFile(tmpFile, []byte(script), 0755)
		args = []string{tmpFile}
	}
	defer os.Remove(tmpFile)

	appendToFile(logFile, fmt.Sprintf("> %s\n", command))

	cmd := exec.Command(name, args...)
	// Set generic env to minimize rubbish
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1", "LANG=en_US.UTF-8")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	rawOutput := stdout.String() + stderr.String()
	if err != nil && rawOutput == "" {
		rawOutput = err.Error()
	}

	output := cleanupOutput(rawOutput)
	appendToFile(logFile, output+"\n\n")

	return output, err == nil
}

func cleanupOutput(input string) string {
	// 1. Strip ANSI escape codes using the robust regex from the Bun original
	ansiRegex := regexp.MustCompile(`[\x1b\x9b][\[\]()#;?]*(?:(?:(?:(?:;[-a-zA-Z\d\\/#&.:=?%@~_]+)*|[a-zA-Z\d]+(?:;[-a-zA-Z\d\\/#&.:=?%@~_]+)*)?\x07)|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PR-TZcf-ntqry=><~]))`)
	output := ansiRegex.ReplaceAllString(input, "")

	// 2. Clear known terminal artifacts (like BEL)
	output = strings.ReplaceAll(output, "\u0007", "")

	// 3. Trim and normalize
	return strings.TrimSpace(output)
}

func appendToFile(filename, content string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("❌ Failed to open file %s: %v\n", filename, err)
		return
	}
	defer f.Close()
	f.WriteString(content)
}
