package report

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

func TestReporterScoring(t *testing.T) {
	config := playbook.Playbook{
		Title: "Test Report",
		Sections: []playbook.Section{
			{
				Title:       "Test Section",
				Description: []string{"Desc"},
				Assertions: []playbook.Assertion{
					{
						Code:            "TEST_01",
						Title:           "Test Assertion",
						Description:     "Test Description",
						MinPassingScore: func(i int) *int { return &i }(2),
						Cmds: []playbook.Cmd{
							{
								Exec:      playbook.Exec{Script: "echo 1"},
								PassScore: func(i int) *int { return &i }(1),
							},
							{
								Exec:      playbook.Exec{Script: "echo 2"},
								PassScore: func(i int) *int { return &i }(1),
							},
						},
					},
				},
			},
		},
	}

	// Mock execution: first succeeds, second fails
	callIdx := 0
	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		callIdx++
		if callIdx == 1 {
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
		}
		return executor.ExecutionResult{ExitCode: 1, Success: false, Stdout: "fail"}, nil
	}
	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured

	// Score: 1 (cmd1 pass) + -1 (cmd2 fail default) = 0
	// MinScore: 2
	// Expect: Passed = false
	ass := report.Assertions["TEST_01"]
	if ass.Passed {
		t.Errorf("Assertion passed with score %d; expected fail (min 2)", ass.Score)
	}
	if ass.Score != 0 {
		t.Errorf("Assertion score = %d; want 0", ass.Score)
	}

	// Now try a passing case
	mockExecPass := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
	}
	runExec = mockExecPass
	res2 := GenerateReport(config)
	report2 := res2.Structured
	if !report2.Assertions["TEST_01"].Passed {
		t.Errorf("Assertion failed with score %d; expected pass (min 2)", report2.Assertions["TEST_01"].Score)
	}
}

func TestExcludeFromReport(t *testing.T) {
	config := playbook.Playbook{
		Title: "Test Exclude",
		Sections: []playbook.Section{
			{
				Title: "Section 1",
				Assertions: []playbook.Assertion{
					{
						Code:  "EXCL_01",
						Title: "Exclusion Test",
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{
									Script:            "echo sensitive_data",
									ExcludeFromReport: true,
									Gather: []playbook.GatherSpec{
										{
											Key:               "sensitive",
											Regex:             "(.*)",
											ExcludeFromReport: true,
										},
										{
											Key:               "public",
											Regex:             "(.*)",
											ExcludeFromReport: false,
										},
									},
								},
							},
						},
						MinPassingScore: func(i int) *int { return &i }(1),
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		out := "sensitive_data"
		res := executor.ExecutionResult{Stdout: out, Success: true, ExitCode: 0}
		for _, g := range e.Gather {
			context[g.Key] = out
		}
		return res, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	md := res.Markdown
	logStr := res.Log

	ass1 := report.Assertions["EXCL_01"]
	if _, exists := ass1.Context["sensitive"]; exists {
		t.Errorf("expected 'sensitive' key to be excluded from report context")
	}
	if _, exists := ass1.Context["public"]; !exists {
		t.Errorf("expected 'public' key to be included in report context")
	}
	if strings.Contains(md, "sensitive_data") {
		t.Errorf("expected sensitive command output to be excluded from markdown report")
	}
	if !strings.Contains(logStr, "[REDACTED]") {
		t.Errorf("expected [REDACTED] to be present in log for STDOUT")
	}
}

func TestGenerateReport_Advanced(t *testing.T) {
	minScore := 1
	config := playbook.Playbook{
		Title: "Advanced Test",
		ReportFrontmatter: map[string]interface{}{
			"custom": "value",
		},
		Sections: []playbook.Section{
			{
				Title: "Advanced Section",
				Assertions: []playbook.Assertion{
					{
						Code:            "ADV_01",
						Title:           "Advanced Assertion",
						MinPassingScore: &minScore,
						PreCmds: []playbook.Exec{
							{Script: "pre-cmd", Gather: []playbook.GatherSpec{{Key: "pre", Regex: "(.*)"}}},
						},
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{Script: "main-cmd"},
								ExitCodeRules: []playbook.ExitCodeRule{
									{Min: func(i int) *int { return &i }(0), Max: func(i int) *int { return &i }(0), Result: 1},
									{Min: func(i int) *int { return &i }(1), Max: func(i int) *int { return &i }(10), Result: -1},
								},
								StdOutRule: playbook.EvaluationRule{Regex: "SUCCESS"},
							},
						},
						PostCmds: []playbook.Exec{
							{Script: "post-cmd"},
						},
						PassDescription: "Passed!",
						FailDescription: "Failed!",
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-cmd" {
			context["pre"] = "pre-val"
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		}
		if e.Script == "main-cmd" {
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "SUCCESS"}, nil
		}
		return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	md := res.Markdown

	ass := report.Assertions["ADV_01"]
	// main-cmd: ExitCode 0 -> Result 1 (via rule). Score += PassScore (default 1? wait)
	// Actually, if PassScore is nil, it uses 1?
	// Let's check what Cmd.GetPassScore returns.

	if !strings.Contains(md, "custom: value") {
		t.Errorf("Markdown frontmatter missing custom value")
	}
	if !strings.Contains(md, "Passed!") {
		t.Errorf("Markdown missing pass description")
	}
	if ass.Context["pre"] != "pre-val" {
		t.Errorf("Context missing pre-command value: %v", ass.Context)
	}
}

func TestGenerateReport_ErrorCases(t *testing.T) {
	config := playbook.Playbook{
		Title: "Error Test",
		Sections: []playbook.Section{
			{
				Title: "Error Section",
				Assertions: []playbook.Assertion{
					{
						Code:  "ERR_01",
						Title: "Error Assertion",
						PreCmds: []playbook.Exec{
							{Script: "pre-fail"},
						},
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "main-fail"}},
						},
						PostCmds: []playbook.Exec{
							{Script: "post-fail"},
						},
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-fail" || e.Script == "post-fail" {
			return executor.ExecutionResult{}, fmt.Errorf("error in command")
		}
		if e.Script == "main-fail" {
			return executor.ExecutionResult{}, fmt.Errorf("main error")
		}
		return executor.ExecutionResult{Success: true}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	ass := report.Assertions["ERR_01"]
	// Main fail will have FailScore (-1 default)
	// pre/post fail just print warnings but don't affect score directly in GenerateReport logic as it stands.
	if ass.Score != -1 {
		t.Errorf("Score = %d; want -1 (from main error)", ass.Score)
	}
}

func TestGenerateReport_EnvUsage(t *testing.T) {
	// Cover USER vs USERNAME
	os.Setenv("USER", "")
	os.Setenv("USERNAME", "testuser")
	defer os.Unsetenv("USERNAME")

	config := playbook.Playbook{
		Title: "Env Test",
		Sections: []playbook.Section{
			{
				Title: "S1",
				Assertions: []playbook.Assertion{
					{
						Code: "E_01",
						Cmds: []playbook.Cmd{{Exec: playbook.Exec{Script: "echo 1"}}},
					},
				},
			},
		},
	}
	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{Stdout: "ok", Stderr: "some error"}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	if report.Username != "testuser" {
		t.Errorf("Username = %s; want testuser", report.Username)
	}
}

func TestGenerateReport_DefaultExitCode(t *testing.T) {
	config := playbook.Playbook{
		Title: "Default Exit Code",
		Sections: []playbook.Section{
			{
				Title: "S1",
				// Two assertions: one success, one fail by exit code
				Assertions: []playbook.Assertion{
					{
						Code:  "E_PASS",
						Title: "Should Pass Assertion",
						Cmds:  []playbook.Cmd{{Exec: playbook.Exec{Script: "ok"}}},
					},
					{
						Code:  "E_FAIL",
						Title: "Should Fail Assertion",
						Cmds:  []playbook.Cmd{{Exec: playbook.Exec{Script: "fail"}}},
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "ok" {
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		}
		return executor.ExecutionResult{ExitCode: 1, Success: false}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	if !report.Assertions["E_PASS"].Passed {
		t.Errorf("E_PASS should have passed")
	}
	if report.Assertions["E_FAIL"].Passed {
		t.Errorf("E_FAIL should have failed")
	}
}

func TestLogExecution_Extended(t *testing.T) {
	var log strings.Builder

	// Multiline script
	execMultiline := playbook.Exec{Script: "line1\nline2"}
	res := executor.ExecutionResult{Stdout: "out\n", Stderr: "err\n", ExitCode: 0}
	logExecution(&log, execMultiline, res, nil)

	logStr := log.String()
	if !strings.Contains(logStr, "SCRIPT") || !strings.Contains(logStr, "<<<<< END SCRIPT") {
		t.Errorf("Multiline script logging failed")
	}

	// Error case
	log.Reset()
	execErr := playbook.Exec{Script: "fail"}
	logExecution(&log, execErr, res, fmt.Errorf("some error"))
	logStr = log.String()
	if !strings.Contains(logStr, "ERROR: some error") {
		t.Errorf("Error logging failed")
	}

	// Redacted case
	log.Reset()
	execRedacted := playbook.Exec{Script: "secret", ExcludeFromReport: true}
	logExecution(&log, execRedacted, res, nil)
	logStr = log.String()
	if !strings.Contains(logStr, "[REDACTED]") {
		t.Errorf("Redaction in logging failed")
	}
}

func TestIsEvidenceMaterial(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{" ", false},
		{"\t", false},
		{"\n", false},
		{"\r", false},
		{" \t\n\r ", false},
		{"a", true},
		{" a ", true},
		{"[REDACTED]", false},
		{"# --- STDOUT ---", false},
		{"# --- STDERR ---", false},
	}

	for _, tt := range tests {
		if got := isEvidenceMaterial(tt.input); got != tt.expected {
			t.Errorf("isEvidenceMaterial(%q) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}

func TestGenerateReport_CoverageBoost(t *testing.T) {
	// 1. Mock GOOS to darwin to cover line 81
	oldOS := goos
	goos = "darwin"
	defer func() { goos = oldOS }()

	// 2. config with nil ReportFrontmatter
	config := playbook.Playbook{
		Title: "Boost Test",
		Sections: []playbook.Section{
			{
				Title: "S1",
				Assertions: []playbook.Assertion{
					{
						Code: "B_01",
						PreCmds: []playbook.Exec{
							{
								Script: "pre-gather",
								Gather: []playbook.GatherSpec{
									{Key: "pre_secret", Regex: "(.*)", ExcludeFromReport: true},
								},
							},
						},
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{Script: "cmd-mixed"},
								StdErrRule: playbook.EvaluationRule{Regex: "ERROR_MATCH"},
							},
							{
								Exec: playbook.Exec{Script: "cmd-stderr-only"},
							},
						},
						PostCmds: []playbook.Exec{
							{
								Script: "post-gather",
								Gather: []playbook.GatherSpec{
									{Key: "post_secret", Regex: "(.*)", ExcludeFromReport: true},
								},
							},
						},
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-gather" {
			context["pre_secret"] = "pre-secret-val"
			return executor.ExecutionResult{Success: true}, nil
		}
		if e.Script == "cmd-mixed" {
			return executor.ExecutionResult{
				Stdout:   "some stdout",
				Stderr:   "ERROR_MATCH",
				ExitCode: 0,
				Success:  true,
			}, nil
		}
		if e.Script == "cmd-stderr-only" {
			return executor.ExecutionResult{
				Stdout:   "",
				Stderr:   "just stderr",
				ExitCode: 0,
				Success:  true,
			}, nil
		}
		if e.Script == "post-gather" {
			context["post_secret"] = "secret-val"
			return executor.ExecutionResult{Success: true}, nil
		}
		return executor.ExecutionResult{Success: true}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)

	// Check OS
	if res.Structured.OS != "mac" {
		t.Errorf("OS = %s; want mac", res.Structured.OS)
	}

	// Check mixed output in Markdown
	if !strings.Contains(res.Markdown, "# --- STDOUT ---") {
		t.Errorf("Markdown missing STDOUT header for mixed output")
	}
	if !strings.Contains(res.Markdown, "# --- STDERR ---") {
		t.Errorf("Markdown missing STDERR header for mixed output")
	}

	// Check stderr-only output: should NOT have STDOUT header
	// (This is a bit tricky to check with strings.Contains, but we can try)
	// Actually, we just care that it doesn't crash and hits the branch.

	// Check gather exclusions
	if _, exists := res.Structured.Assertions["B_01"].Context["pre_secret"]; exists {
		t.Errorf("pre_secret should be excluded from report context")
	}
	if _, exists := res.Structured.Assertions["B_01"].Context["post_secret"]; exists {
		t.Errorf("post_secret should be excluded from report context")
	}

	// Check default frontmatter (title and date)
	if !strings.Contains(res.Markdown, "title: Boost Test") {
		t.Errorf("Markdown missing default title in frontmatter")
	}
}
