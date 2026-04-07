package main

import (
	"compliance-probe/executor"
	"compliance-probe/playbook"
	"compliance-probe/report"
	"compliance-probe/internal/reportwriter"
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	flag.Parse()

	configPath := getConfigPath()
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'compliance-probe [path/to/playbook.yaml]'")
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to read playbook %s: %v\n", configPath, err)
		os.Exit(1)
	}

	var config playbook.ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// Validate as Agent
	if err := playbook.ValidateConfig(config, true); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	result := report.GenerateReport(config, executor.RunExec)
	reportwriter.WriteToFolder(result)
}

func getConfigPath() string {
	args := flag.Args()
	if len(args) > 0 {
		return args[0]
	}
	if fileExists("playbook.yaml") {
		return "playbook.yaml"
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
