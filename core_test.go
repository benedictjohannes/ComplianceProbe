package main

import (
	"strings"
	"testing"
)

func TestCleanupOutput(t *testing.T) {
	input := "\x1b[31mError:\x1b[0m \u0007Something went wrong\n"
	expected := "Error: Something went wrong"
	got := cleanupOutput(input)
	if got != expected {
		t.Errorf("cleanupOutput() = %q; want %q", got, expected)
	}
}

func TestPerformGatherRegex(t *testing.T) {
	g := GatherSpec{
		Key:   "version",
		Regex: "v(\\d+)",
	}
	res := ExecutionResult{Stdout: "Product v123"}
	context := make(map[string]interface{})
	got, err := performGather(g, res, context)
	if err != nil {
		t.Fatalf("performGather() error: %v", err)
	}
	if got != "123" {
		t.Errorf("performGather() = %q; want %q", got, "123")
	}
}

func TestRunShellSuccess(t *testing.T) {
	res := runShell("ls -lh", "")
	if !res.Success {
		t.Errorf("runShell() success = false; want true. Stderr: %q", res.Stderr)
	}
	if res.ExitCode != 0 {
		t.Errorf("runShell() exitCode = %d; want 0", res.ExitCode)
	}
}

func TestRunShellFailure(t *testing.T) {
	// Use a highly unlikely command to ensure failure
	res := runShell("non_existent_command_12345", "bash")
	if res.Success {
		t.Errorf("runShell() success = true; want false")
	}
	if res.ExitCode == 0 {
		t.Errorf("runShell() exitCode = 0; want non-zero")
	}
}

func TestReporterScoring(t *testing.T) {
	config := ReportConfig{
		Title: "Test Report",
		Sections: []Section{
			{
				Title:       "Test Section: Should Fail",
				Description: []string{"Desc"},
				Assertions: []Assertion{
					{
						Code:            "TEST_01",
						Title:           "Test Assertion",
						Description:     "Test Description",
						MinPassingScore: &[]int{2}[0], // Pointer to 2
						Cmds: []Cmd{
							{
								Exec:      Exec{Script: "echo 1"},
								PassScore: &[]int{1}[0],
							},
							{
								Exec:      Exec{Script: "echo 2"},
								PassScore: &[]int{1}[0],
							},
						},
						PassDescription: "Passed",
						FailDescription: "Failed",
					},
				},
			},
		},
	}

	// Mock execution: first succeeds, second fails
	callIdx := 0
	mockExec := func(e *Exec, context map[string]interface{}) (ExecutionResult, error) {
		callIdx++
		if callIdx == 1 {
			return ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
		}
		return ExecutionResult{ExitCode: 1, Success: false, Stdout: "fail"}, nil
	}

	report, _, _ := generateReport(config, mockExec)

	// Score: 1 (cmd1 pass) + -1 (cmd2 fail default) = 0
	// MinScore: 2
	// Expect: Passed = false
	config.Sections[0].Title = "Test Section: Should Pass"
	ass := report.Assertions["TEST_01"]
	if ass.Passed {
		t.Errorf("Assertion passed with score %d; expected fail (min 2)", ass.Score)
	}
	if ass.Score != 0 {
		t.Errorf("Assertion score = %d; want 0", ass.Score)
	}

	// Now try a passing case
	mockExecPass := func(e *Exec, context map[string]interface{}) (ExecutionResult, error) {
		return ExecutionResult{ExitCode: 0, Success: true}, nil
	}
	report2, _, _ := generateReport(config, mockExecPass)
	if !report2.Assertions["TEST_01"].Passed {
		t.Errorf("Assertion failed with score %d; expected pass (min 2)", report2.Assertions["TEST_01"].Score)
	}
}

func TestValidator(t *testing.T) {
	config := ReportConfig{
		Title: "Test",
		Sections: []Section{
			{
				Title: "S1",
				Assertions: []Assertion{
					{Code: "DUP_01", Title: "A1"},
					{Code: "DUP_01", Title: "A2"},
				},
			},
		},
	}

	err := validateConfig(config, false)
	if err == nil {
		t.Errorf("validateConfig() failed to catch duplicate codes")
	}
	if !strings.Contains(err.Error(), "duplicate code found") {
		t.Errorf("validateConfig() error message = %v; want duplicate code error", err)
	}
}
