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
	config, _, err := configsource.LoadConfig(rawPlaybookPath, nil)
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

func TestPreprocessAssertionErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Error in PreCmds
	config := &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						PreCmds: []playbook.Exec{{FuncFile: "missing.ts"}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing PreCmds funcFile")
	}

	// 2. Error in PostCmds
	config = &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						PostCmds: []playbook.Exec{{FuncFile: "missing.ts"}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing PostCmds funcFile")
	}

	// 3. Error in Cmds.Exec
	config = &playbook.Playbook{
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
		t.Error("expected error for missing Cmds.Exec funcFile")
	}

	// 4. Error in StdOutRule
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
		t.Error("expected error for missing StdOutRule funcFile")
	}

	// 5. Error in StdErrRule
	config = &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						Cmds: []playbook.Cmd{{StdErrRule: playbook.EvaluationRule{FuncFile: "missing.ts"}}},
					},
				},
			},
		},
	}
	if err := Preprocess(config, tmpDir); err == nil {
		t.Error("expected error for missing StdErrRule funcFile")
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

func TestBakeFileErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Load error
	err := BakeFile("non-existent.yaml", filepath.Join(tmpDir, "out.yaml"))
	if err == nil || !strings.Contains(err.Error(), "failed to load input") {
		t.Errorf("expected load error, got: %v", err)
	}

	// 2. Preprocess error (transpilation failure)
	inputPath := filepath.Join(tmpDir, "invalid_transpile.yaml")
	rawYaml := `
sections:
  - assertions:
      - cmds:
          - exec: { funcFile: "missing.ts" }
`
	if err := os.WriteFile(inputPath, []byte(rawYaml), 0644); err != nil {
		t.Fatal(err)
	}
	err = BakeFile(inputPath, filepath.Join(tmpDir, "out.yaml"))
	if err == nil || !strings.Contains(err.Error(), "transpilation error") {
		t.Errorf("expected preprocessing error, got: %v", err)
	}

	// 3. Validation error
	inputPath = filepath.Join(tmpDir, "invalid_validation.yaml")
	// Minimum valid playbook needs title, but also other fields might fail validation if we use ValidateConfig(config, false)
	// Let's check playbook.ValidateConfig implementation if needed, but usually empty title or missing assertions code works.
	rawYaml = `
title: "Invalid"
sections:
  - title: "S1"
    assertions:
      - title: "Missing Code"
        # code is missing
        cmds:
          - exec: { script: "echo" }
`
	if err := os.WriteFile(inputPath, []byte(rawYaml), 0644); err != nil {
		t.Fatal(err)
	}
	err = BakeFile(inputPath, filepath.Join(tmpDir, "out.yaml"))
	if err == nil || !strings.Contains(err.Error(), "validation error") {
		t.Errorf("expected validation error, got: %v", err)
	}

	// 4. Write error
	inputPath = filepath.Join(tmpDir, "valid.yaml")
	rawYaml = `title: "Valid"`
	if err := os.WriteFile(inputPath, []byte(rawYaml), 0644); err != nil {
		t.Fatal(err)
	}
	// Try writing to a path that is a directory
	err = BakeFile(inputPath, tmpDir)
	if err == nil || !strings.Contains(err.Error(), "failed to write output") {
		t.Errorf("expected write error, got: %v", err)
	}

	// 5. Remote URL error
	err = BakeFile("https://example.com/playbook.yaml", filepath.Join(tmpDir, "out.yaml"))
	if err == nil || !strings.Contains(err.Error(), "baking remote playbooks is not supported") {
		t.Errorf("expected remote URL error, got: %v", err)
	}
}

func TestPreprocessFullAssertion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a dummy TS file
	tsFile := filepath.Join(tmpDir, "script.ts")
	tsCode := "console.log('hello');"
	if err := os.WriteFile(tsFile, []byte(tsCode), 0644); err != nil {
		t.Fatal(err)
	}

	config := &playbook.Playbook{
		Sections: []playbook.Section{
			{
				Assertions: []playbook.Assertion{
					{
						PreCmds: []playbook.Exec{
							{FuncFile: "script.ts"},
						},
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{
									ShellFuncFile: "script.ts",
									Gather: []playbook.GatherSpec{
										{FuncFile: "script.ts"},
									},
								},
								StdOutRule: playbook.EvaluationRule{FuncFile: "script.ts"},
								StdErrRule: playbook.EvaluationRule{FuncFile: "script.ts"},
							},
						},
						PostCmds: []playbook.Exec{
							{FuncFile: "script.ts"},
						},
					},
				},
			},
		},
	}

	if err := Preprocess(config, tmpDir); err != nil {
		t.Fatalf("Preprocess failed: %v", err)
	}

	a := config.Sections[0].Assertions[0]

	// Verify PreCmds
	if a.PreCmds[0].Func == "" || a.PreCmds[0].FuncFile != "" {
		t.Error("PreCmds: Func should be populated and FuncFile cleared")
	}

	// Verify Cmds
	c := a.Cmds[0]
	if c.Exec.ShellFunc == "" || c.Exec.ShellFuncFile != "" {
		t.Error("Cmds.Exec: ShellFunc should be populated and ShellFuncFile cleared")
	}
	if c.Exec.Gather[0].Func == "" || c.Exec.Gather[0].FuncFile != "" {
		t.Error("Cmds.Exec.Gather: Func should be populated and FuncFile cleared")
	}
	if c.StdOutRule.Func == "" || c.StdOutRule.FuncFile != "" {
		t.Error("StdOutRule: Func should be populated and FuncFile cleared")
	}
	if c.StdErrRule.Func == "" || c.StdErrRule.FuncFile != "" {
		t.Error("StdErrRule: Func should be populated and FuncFile cleared")
	}

	// Verify PostCmds
	if a.PostCmds[0].Func == "" || a.PostCmds[0].FuncFile != "" {
		t.Error("PostCmds: Func should be populated and FuncFile cleared")
	}
}
