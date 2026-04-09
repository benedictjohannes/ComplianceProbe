package transpile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/playbook"
)

func TestPreprocessWithE2EAssets(t *testing.T) {
	// Root of the project (assuming we are in internal/transpile)
	// We'll use a relative path to find the test-e2e folder
	baseDir, err := filepath.Abs("../../test-e2e")
	if err != nil {
		t.Fatalf("failed to get absolute path to test-e2e: %v", err)
	}

	rawPlaybookPath := filepath.Join(baseDir, "test.playbook.raw.yaml")
	
	// 1. Load the raw playbook
	config, _, err := configsource.LoadConfig(rawPlaybookPath)
	if err != nil {
		t.Fatalf("failed to load raw playbook: %v", err)
	}

	// 2. Run Preprocess
	if err := Preprocess(config, baseDir); err != nil {
		t.Fatalf("Preprocess failed: %v", err)
	}

	// 3. Verify specifically the 'PlainTest' assertion which uses both shellFuncFile and funcFile
	foundPlainTest := false
	for _, section := range config.Sections {
		for _, assertion := range section.Assertions {
			if assertion.Code == "PlainTest" {
				foundPlainTest = true
				exec := assertion.Cmds[0].Exec
				
				// Verify shellFunc was transpiled from plainVariable.ts
				if exec.ShellFunc == "" {
					t.Error("PlainTest: ShellFunc should be populated")
				}
				if !strings.Contains(exec.ShellFunc, "node") {
					t.Errorf("PlainTest: ShellFunc content mismatch, got: %s", exec.ShellFunc)
				}
				if exec.ShellFuncFile != "" {
					t.Error("PlainTest: ShellFuncFile should be cleared")
				}

				// Verify Func was transpiled from plainFunction.ts
				if exec.Func == "" {
					t.Error("PlainTest: Func should be populated")
				}
				if !strings.Contains(exec.Func, "hello from plain") {
					t.Errorf("PlainTest: Func content mismatch, got: %s", exec.Func)
				}
				if exec.FuncFile != "" {
					t.Error("PlainTest: FuncFile should be cleared")
				}
			}
		}
	}

	if !foundPlainTest {
		t.Error("Could not find 'PlainTest' assertion in the E2E playbook")
	}
}

func TestPreprocessErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Missing funcFile
	config := &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						Cmds: []playbook.Cmd{{Exec: playbook.Exec{FuncFile: "missing.ts"}}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing funcFile")
	}

	// 2. Missing shellFuncFile
	config = &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						Cmds: []playbook.Cmd{{Exec: playbook.Exec{ShellFuncFile: "missing.ts"}}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing shellFuncFile")
	}

	// 3. Missing gather funcFile
	config = &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						Cmds: []playbook.Cmd{{Exec: playbook.Exec{Gather: []playbook.GatherSpec{{FuncFile: "missing.ts"}}}}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing gather funcFile")
	}

	// 4. Missing evalRule funcFile
	config = &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						Cmds: []playbook.Cmd{{StdOutRule: playbook.EvaluationRule{FuncFile: "missing.ts"}}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing evalRule funcFile")
	}
}

func TestBakeFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create raw playbook
	rawYaml := `
title: "Bake Test"
sections:
  - title: "S1"
    assertions:
      - code: A1
        title: "T1"
        cmds:
          - exec:
              script: "echo test"
`
	inputPath := filepath.Join(tmpDir, "raw.yaml")
	outputPath := filepath.Join(tmpDir, "baked.yaml")
	
	if err := os.WriteFile(inputPath, []byte(rawYaml), 0644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}
	
	// Run BakeFile
	if err := BakeFile(inputPath, outputPath); err != nil {
		t.Fatalf("BakeFile failed: %v", err)
	}
	
	// Verify output exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("baked file was not created")
	}
}
