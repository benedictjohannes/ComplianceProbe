//go:build builder

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"gopkg.in/yaml.v3"
)

func runPreprocess(inputPath string, outputPath string) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("❌ Failed to read input: %v\n", err)
		os.Exit(1)
	}

	var config ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// Walk and transpile
	for i := range config.Sections {
		for j := range config.Sections[i].Assertions {
			a := &config.Sections[i].Assertions[j]
			processAssertion(a, filepath.Dir(inputPath))
		}
	}

	// Validate (including builder-specific properties before they are ignored)
	if err := validateConfig(config, false); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	// Save "baked" playbook
	outData, err := yaml.Marshal(config)
	if err != nil {
		fmt.Printf("❌ Failed to marshal YAML: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputPath, outData, 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Preprocessing Complete! Baked playbook saved to: %s\n", outputPath)
}

func processAssertion(a *Assertion, baseDir string) {
	for i := range a.PreCmds {
		processExec(&a.PreCmds[i], baseDir)
	}
	for i := range a.Cmds {
		processExec(&a.Cmds[i].Exec, baseDir)
		processEvalRule(&a.Cmds[i].StdOutRule, baseDir)
		processEvalRule(&a.Cmds[i].StdErrRule, baseDir)
	}
	for i := range a.PostCmds {
		processExec(&a.PostCmds[i], baseDir)
	}
}

func processExec(e *Exec, baseDir string) {
	if e.FuncFile != "" {
		code, err := transpile(filepath.Join(baseDir, e.FuncFile))
		if err != nil {
			fmt.Printf("❌ Transpilation Error (%s): %v\n", e.FuncFile, err)
			os.Exit(1)
		}
		e.Func = code
		e.FuncFile = ""
	}
	for i := range e.Gather {
		if e.Gather[i].FuncFile != "" {
			code, err := transpile(filepath.Join(baseDir, e.Gather[i].FuncFile))
			if err != nil {
				fmt.Printf("❌ Transpilation Error (%s): %v\n", e.Gather[i].FuncFile, err)
				os.Exit(1)
			}
			e.Gather[i].Func = code
			e.Gather[i].FuncFile = ""
		}
	}
}

func processEvalRule(r *EvaluationRule, baseDir string) {
	if r.FuncFile != "" {
		code, err := transpile(filepath.Join(baseDir, r.FuncFile))
		if err != nil {
			fmt.Printf("❌ Transpilation Error (%s): %v\n", r.FuncFile, err)
			os.Exit(1)
		}
		r.Func = code
		r.FuncFile = ""
	}
}

func transpile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	codeStr := string(data)
	// Naive check for module keywords.
	// Esbuild will handle the actual transpilation.
	isModule := false
	if (len(codeStr) > 7 && codeStr[:7] == "export ") ||
		(len(codeStr) > 7 && codeStr[:7] == "import ") ||
		strings.Contains(codeStr, "\nexport ") ||
		strings.Contains(codeStr, "\nimport ") {
		isModule = true
	}

	opts := api.TransformOptions{
		Loader: api.LoaderTS,
		Target: api.ES5,
	}

	if isModule {
		opts.Format = api.FormatCommonJS
		opts.MinifyWhitespace = true
		opts.MinifyIdentifiers = true
		opts.MinifySyntax = true
	} else {
		opts.Format = api.FormatDefault
		// For naked expressions, we avoid aggressive minification that might
		// eliminate the expression as dead code.
		opts.MinifyWhitespace = true
	}

	result := api.Transform(codeStr, opts)
	if len(result.Errors) > 0 {
		return "", fmt.Errorf("esbuild error: %v", result.Errors[0].Text)
	}

	if isModule {
		// Wrap in an IIFE to capture CommonJS exports and return the default one (or the whole object)
		return fmt.Sprintf("(function(){var exports={};var module={exports:exports};%s;return module.exports.default||module.exports})()", result.Code), nil
	}

	return strings.TrimSpace(string(result.Code)), nil
}
