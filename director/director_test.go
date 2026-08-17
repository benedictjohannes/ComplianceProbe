package director

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

func TestDirectorScoring(t *testing.T) {
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
	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		callIdx++
		if callIdx == 1 {
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
		}
		return executor.ExecutionResult{ExitCode: 1, Success: false, Stdout: "fail"}, nil
	}
	runExec = mockExec
	trace := Run(context.Background(), config)
	
	ass := trace.Sections[0].Assertions[0]
	if ass.Passed {
		t.Errorf("Assertion passed with score %d; expected fail (min 2)", ass.Score)
	}
	if ass.Score != 0 {
		t.Errorf("Assertion score = %d; want 0", ass.Score)
	}

	// Now try a passing case
	mockExecPass := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
	}
	runExec = mockExecPass
	trace2 := Run(context.Background(), config)
	if !trace2.Sections[0].Assertions[0].Passed {
		t.Errorf("Assertion failed with score %d; expected pass (min 2)", trace2.Sections[0].Assertions[0].Score)
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

	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		out := "sensitive_data"
		res := executor.ExecutionResult{Stdout: out, Success: true, ExitCode: 0}
		for _, g := range e.Gather {
			context[g.Key] = out
		}
		return res, nil
	}

	runExec = mockExec
	trace := Run(context.Background(), config)

	ass := trace.Sections[0].Assertions[0]
	if _, exists := ass.Context["sensitive"]; exists {
		t.Errorf("expected 'sensitive' key to be excluded from report context")
	}
	if _, exists := ass.Context["public"]; !exists {
		t.Errorf("expected 'public' key to be included in report context")
	}
}

func TestDirector_AssertionPipeline_GatherAndOutputs(t *testing.T) {
	minScore := 1
	config := playbook.Playbook{
		Title: "Advanced Pipeline Test",
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
							{
								Script: "pre-cmd",
								Gather: []playbook.GatherSpec{
									{Key: "pre", Regex: "(.*)"},
									{Key: "pre_secret", Regex: "(.*)", ExcludeFromReport: true},
								},
							},
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
							{
								Exec:       playbook.Exec{Script: "cmd-mixed"},
								StdErrRule: playbook.EvaluationRule{Regex: "WARN_MATCH"},
							},
							{
								Exec: playbook.Exec{Script: "cmd-stderr-only"},
							},
							{
								Exec: playbook.Exec{Script: "cmd-redacted", ExcludeFromReport: true},
							},
						},
						PostCmds: []playbook.Exec{
							{
								Script: "post-cmd",
								Gather: []playbook.GatherSpec{
									{Key: "post_secret", Regex: "(.*)", ExcludeFromReport: true},
								},
							},
						},
						PassDescription: "Passed!",
						FailDescription: "Failed!",
					},
				},
			},
		},
	}

	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		switch e.Script {
		case "pre-cmd":
			context["pre"] = "pre-val"
			context["pre_secret"] = "pre-secret-val"
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		case "main-cmd":
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "SUCCESS"}, nil
		case "cmd-mixed":
			return executor.ExecutionResult{
				Stdout:   "stdout-data",
				Stderr:   "WARN_MATCH",
				ExitCode: 0,
				Success:  true,
			}, nil
		case "cmd-stderr-only":
			return executor.ExecutionResult{
				Stdout:   "",
				Stderr:   "err-only-data",
				ExitCode: 0,
				Success:  true,
			}, nil
		case "cmd-redacted":
			return executor.ExecutionResult{
				Stdout:   "top-secret",
				ExitCode: 0,
				Success:  true,
			}, nil
		case "post-cmd":
			context["post_secret"] = "post-secret-val"
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		default:
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		}
	}

	runExec = mockExec
	trace := Run(context.Background(), config)
	ass := trace.Sections[0].Assertions[0]

	t.Run("ContextAndRedaction", func(t *testing.T) {
		if ass.Context["pre"] != "pre-val" {
			t.Errorf("Context missing pre-command value: %v", ass.Context)
		}
		if _, exists := ass.Context["pre_secret"]; exists {
			t.Errorf("expected 'pre_secret' key to be excluded from report context")
		}
		if _, exists := ass.Context["post_secret"]; exists {
			t.Errorf("expected 'post_secret' key to be excluded from report context")
		}
	})

	t.Run("OutputsAggregationAndFormatting", func(t *testing.T) {
		expectedOutputs := []string{
			"SUCCESS",
			"# --- STDOUT ---",
			"stdout-data",
			"# --- STDERR ---",
			"WARN_MATCH",
			"# --- STDERR ---",
			"err-only-data",
			"[REDACTED]",
		}
		if len(ass.Outputs) != len(expectedOutputs) {
			t.Fatalf("expected %d output entries, got %d: %v", len(expectedOutputs), len(ass.Outputs), ass.Outputs)
		}
		for i, exp := range expectedOutputs {
			if ass.Outputs[i] != exp {
				t.Errorf("ass.Outputs[%d] = %q; want %q", i, ass.Outputs[i], exp)
			}
		}
	})

	t.Run("PassEvaluation", func(t *testing.T) {
		if !ass.Passed {
			t.Errorf("expected assertion to pass")
		}
	})
}

func TestDirector_ErrorCases(t *testing.T) {
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

	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-fail" || e.Script == "post-fail" {
			return executor.ExecutionResult{}, fmt.Errorf("error in command")
		}
		if e.Script == "main-fail" {
			return executor.ExecutionResult{}, fmt.Errorf("main error")
		}
		return executor.ExecutionResult{Success: true}, nil
	}

	runExec = mockExec
	trace := Run(context.Background(), config)
	ass := trace.Sections[0].Assertions[0]
	if ass.Score != -1 {
		t.Errorf("Score = %d; want -1 (from main error)", ass.Score)
	}
}

func TestDirector_EnvUsage(t *testing.T) {
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
	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{Stdout: "ok", Stderr: "some error"}, nil
	}

	runExec = mockExec
	trace := Run(context.Background(), config)

	if trace.Username != "testuser" {
		t.Errorf("Username = %s; want testuser", trace.Username)
	}
}

func TestDirector_DefaultExitCode(t *testing.T) {
	config := playbook.Playbook{
		Title: "Default Exit Code",
		Sections: []playbook.Section{
			{
				Title: "S1",
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

	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "ok" {
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		}
		return executor.ExecutionResult{ExitCode: 1, Success: false}, nil
	}

	runExec = mockExec
	trace := Run(context.Background(), config)

	if !trace.Sections[0].Assertions[0].Passed {
		t.Errorf("E_PASS should have passed")
	}
	if trace.Sections[0].Assertions[1].Passed {
		t.Errorf("E_FAIL should have failed")
	}
}

func TestDirector_OSNormalization(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{goos: "darwin", want: "mac"},
		{goos: "linux", want: "linux"},
		{goos: "windows", want: "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			oldOS := goos
			goos = tt.goos
			defer func() { goos = oldOS }()

			trace := Run(context.Background(), playbook.Playbook{Title: "OS Test"})
			if trace.OS != tt.want {
				t.Errorf("trace.OS = %q; want %q for GOOS %q", trace.OS, tt.want, tt.goos)
			}
		})
	}
}

func TestRun_Basic(t *testing.T) {
	runExec = func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
	}
	config := playbook.Playbook{
		Title: "Run Basic Playbook",
		Sections: []playbook.Section{
			{
				Title: "Sec1",
				Assertions: []playbook.Assertion{
					{
						Code:  "A1",
						Title: "Ass1",
						Cmds:  []playbook.Cmd{{Exec: playbook.Exec{Script: "echo ok"}}},
					},
				},
			},
		},
	}
	trace := Run(context.Background(), config)
	if trace.TotalPassed != 1 {
		t.Errorf("expected 1 passed in Run, got %d", trace.TotalPassed)
	}
}



