package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	schemaFlag := flag.Bool("schema", false, "Output the configuration JSON schema and exit")
	preprocessFlag := flag.Bool("preprocess", false, "Preprocess a raw YAML into a baked playbook")
	inputFlag := flag.String("input", "", "Input raw YAML file (for preprocess)")
	outputFlag := flag.String("output", "playbook.yaml", "Output baked YAML file (for preprocess)")
	flag.Parse()

	if *schemaFlag {
		callGenerateSchema()
		return
	}

	if *preprocessFlag {
		if *inputFlag == "" {
			fmt.Println("❌ Error: --input is required for --preprocess")
			os.Exit(1)
		}
		callRunPreprocess(*inputFlag, *outputFlag)
		return
	}

	// Default: Run Agent Report
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

	var config ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// Validate as Agent
	if err := validateConfig(config, true); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	runReport(config)
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
