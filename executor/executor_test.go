package executor

import (
	"context"
	"testing"

	"github.com/benedictjohannes/crobe/playbook"
)

func TestCleanupOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ANSI and BEL",
			input:    "\x1b[31mError:\x1b[0m \u0007Something went wrong\n",
			expected: "Error: Something went wrong",
		},
		{
			name:     "Mixed Newlines and Tabs",
			input:    "Line 1\r\nLine 2\tTabbed",
			expected: "Line 1\r\nLine 2\tTabbed",
		},
		{
			name:     "Control Characters",
			input:    "Hello\u0001World",
			expected: "HelloWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanupOutput(tt.input)
			if got != tt.expected {
				t.Errorf("CleanupOutput() = %q; want %q", got, tt.expected)
			}
		})
	}
}

func TestPerformGather(t *testing.T) {
	tests := []struct {
		name     string
		spec     playbook.GatherSpec
		res      ExecutionResult
		expected string
	}{
		{
			name: "Regex Capture",
			spec: playbook.GatherSpec{
				Key:   "v",
				Regex: "v(\\d+)",
			},
			res:      ExecutionResult{Stdout: "Product v123"},
			expected: "123",
		},
		{
			name: "JS Function",
			spec: playbook.GatherSpec{
				Key:  "v",
				Func: "(stdout) => stdout.split(' ')[1].substring(1)",
			},
			res:      ExecutionResult{Stdout: "Product v123"},
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PerformGather(tt.spec, tt.res, make(map[string]interface{}))
			if err != nil {
				t.Fatalf("PerformGather() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("PerformGather() = %q; want %q", got, tt.expected)
			}
		})
	}

	// Test IncludeStdErr in Gather
	isTrue := true
	spec := playbook.GatherSpec{Key: "err", Regex: "FATAL: (.*)", IncludeStdErr: &isTrue}
	res := ExecutionResult{Stdout: "", Stderr: "FATAL: system crash"}
	got, _ := PerformGather(spec, res, nil)
	if got != "system crash" {
		t.Errorf("PerformGather(stderr) = %q; want \"system crash\"", got)
	}

	// Test Regex with no capture group
	spec2 := playbook.GatherSpec{Key: "all", Regex: "Product v123"}
	res2 := ExecutionResult{Stdout: "Product v123"}
	got, _ = PerformGather(spec2, res2, nil)
	if got != "Product v123" {
		t.Errorf("PerformGather(no-group) = %q; want \"Product v123\"", got)
	}

	// Test No match
	spec3 := playbook.GatherSpec{Key: "none", Regex: "MISSING"}
	res3 := ExecutionResult{Stdout: "nothing here"}
	got, _ = PerformGather(spec3, res3, nil)
	if got != "" {
		t.Errorf("PerformGather(no-match) = %q; want \"\"", got)
	}

	// Test invalid regex
	spec4 := playbook.GatherSpec{Key: "err", Regex: "[["}
	_, err := PerformGather(spec4, res3, nil)
	if err == nil {
		t.Error("PerformGather should return error on invalid regex")
	}

	// Test JS error
	spec5 := playbook.GatherSpec{Key: "jserr", Func: "() => { throw new Error('ops') }"}
	_, err = PerformGather(spec5, res3, nil)
	if err == nil {
		t.Error("PerformGather should return error on JS error")
	}

	// Test JS returning undefined
	spec6 := playbook.GatherSpec{Key: "v", Func: "() => undefined"}
	got, err = PerformGather(spec6, res, nil)
	if err != nil || got != "undefined" { // goja.Value.String() for undefined is "undefined"
		t.Errorf("PerformGather(undefined) = %q, %v", got, err)
	}

	// Test JS returning number
	spec7 := playbook.GatherSpec{Key: "v", Func: "() => 123"}
	got, err = PerformGather(spec7, res, nil)
	if err != nil || got != "123" {
		t.Errorf("PerformGather(123) = %q, %v", got, err)
	}

	// Test JS returning direct value
	spec8 := playbook.GatherSpec{Key: "v", Func: "'direct value'"}
	got, err = PerformGather(spec8, res, nil)
	if err != nil || got != "direct value" {
		t.Errorf("PerformGather(direct) = %q, %v", got, err)
	}
}

func TestRunJS(t *testing.T) {
	context := map[string]interface{}{"foo": "bar"}
	code := "({ assertionContext }) => assertionContext.foo + 'baz'"
	got, err := RunJS(code, context)
	if err != nil {
		t.Fatalf("RunJS() error: %v", err)
	}
	if got != "barbaz" {
		t.Errorf("RunJS() = %q; want %q", got, "barbaz")
	}

	// Test direct code returning value
	got, err = RunJS("'hello ' + os", context)
	if err != nil {
		t.Fatalf("RunJS(direct) error: %v", err)
	}
	if got == "" {
		t.Error("RunJS(direct) should not be empty")
	}

	// Test returning empty/null
	got, err = RunJS("null", context)
	if err != nil || got != "" {
		t.Errorf("RunJS(null) = %q, %v; want \"\", nil", got, err)
	}

	// Test syntax error
	_, err = RunJS("this is not valid js", context)
	if err == nil {
		t.Error("RunJS should return error on syntax error")
	}

	// Test execution error within function
	_, err = RunJS("() => { nonExistent() }", context)
	if err == nil {
		t.Error("RunJS should return error on JS execution error")
	}
}

func TestRunExec(t *testing.T) {
	contextMap := make(map[string]interface{})

	// 1. Simple Script
	e := &playbook.Exec{Script: "echo hello world"}
	res, err := RunExec(context.Background(), e, contextMap)
	if err != nil {
		t.Fatalf("RunExec(simple) error: %v", err)
	}
	if res.Stdout != "hello world" {
		t.Errorf("RunExec(simple) stdout = %q; want \"hello world\"", res.Stdout)
	}

	// 2. JS Func generates script
	e2 := &playbook.Exec{Func: "() => 'echo js script'"}
	res, err = RunExec(context.Background(), e2, contextMap)
	if err != nil {
		t.Fatalf("RunExec(js-func) error: %v", err)
	}
	if res.Stdout != "js script" {
		t.Errorf("RunExec(js-func) stdout = %q; want \"js script\"", res.Stdout)
	}

	// 3. Gathering in RunExec
	e3 := &playbook.Exec{
		Script: "echo 12345",
		Gather: []playbook.GatherSpec{
			{Key: "test_key", Regex: "(\\d+)"},
		},
	}
	res, err = RunExec(context.Background(), e3, contextMap)
	if err != nil {
		t.Fatalf("RunExec(gather) error: %v", err)
	}
	if contextMap["test_key"] != "12345" {
		t.Errorf("RunExec(gather) contextMap[\"test_key\"] = %v; want \"12345\"", contextMap["test_key"])
	}

	// 4. Empty script
	e4 := &playbook.Exec{Script: ""}
	res, err = RunExec(context.Background(), e4, contextMap)
	if err != nil || !res.Success {
		t.Errorf("RunExec(empty) = %+v, %v; want Success: true, nil err", res, err)
	}

	// 5. JS error
	e5 := &playbook.Exec{Func: "() => { throw new Error('ops') }"}
	_, err = RunExec(context.Background(), e5, contextMap)
	if err == nil {
		t.Error("RunExec should fail on JS error")
	}

	// 6. JS returns empty string
	e6 := &playbook.Exec{Func: "() => ''"}
	res, err = RunExec(context.Background(), e6, contextMap)
	if err != nil || !res.Success {
		t.Errorf("RunExec(js-empty) = %+v, %v; want Success: true, nil err", res, err)
	}

	// 7. ShellFunc decides shell
	e7 := &playbook.Exec{
		ShellFunc: "() => '!'",
		Script:    "echo shell_func_test",
	}
	res, err = RunExec(context.Background(), e7, contextMap)
	if err != nil {
		t.Fatalf("RunExec(shell-func) error: %v", err)
	}
	if res.Stdout != "shell_func_test" {
		t.Errorf("RunExec(shell-func) stdout = %q; want \"shell_func_test\"", res.Stdout)
	}

	// 8. ShellFunc error
	e8 := &playbook.Exec{
		ShellFunc: "() => { throw new Error('shell error') }",
		Script:    "echo fail",
	}
	_, err = RunExec(context.Background(), e8, contextMap)
	if err == nil {
		t.Error("RunExec should fail on ShellFunc JS error")
	}

	// 9. ShellFunc returns empty string (should fallback)
	e9 := &playbook.Exec{
		ShellFunc: "() => ''",
		Shell:     "sh",
		Script:    "echo hello",
	}
	res, err = RunExec(context.Background(), e9, contextMap)
	if err != nil || res.Stdout != "hello" {
		t.Errorf("RunExec(ShellFunc empty) = %+v, %v; want \"hello\"", res, err)
	}

	// 10. PerformGather error in RunExec
	e10 := &playbook.Exec{
		Script: "echo hello",
		Gather: []playbook.GatherSpec{
			{Key: "fail", Regex: "[["},
		},
	}
	_, err = RunExec(context.Background(), e10, contextMap)
	if err == nil {
		t.Error("RunExec should fail on gather error")
	}
}

func TestEvaluateRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     playbook.EvaluationRule
		res      ExecutionResult
		expected int
	}{
		{
			name:     "Regex Match",
			rule:     playbook.EvaluationRule{Regex: "PASS"},
			res:      ExecutionResult{Stdout: "Result: PASS"},
			expected: 1,
		},
		{
			name:     "Regex No Match",
			rule:     playbook.EvaluationRule{Regex: "PASS"},
			res:      ExecutionResult{Stdout: "Result: FAIL"},
			expected: -1,
		},
		{
			name:     "JS Function Pass",
			rule:     playbook.EvaluationRule{Func: "(stdout) => stdout.includes('OK') ? 1 : -1"},
			res:      ExecutionResult{Stdout: "Status is OK"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateRule(tt.rule, tt.res, make(map[string]interface{}))
			if err != nil {
				t.Fatalf("EvaluateRule() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("EvaluateRule() = %d; want %d", got, tt.expected)
			}
		})
	}

	// Test IncludeStdErr
	rule := playbook.EvaluationRule{Regex: "ERROR"}
	res := ExecutionResult{Stdout: "", Stderr: "ERROR: something happened"}
	// Without IncludeStdErr, it shouldn't find it if Stdout is empty
	got, _ := EvaluateRule(rule, res, nil)
	if got == 1 {
		t.Error("EvaluateRule should not match stderr by default if stdout is empty")
	}

	isTrue := true
	rule.IncludeStdErr = &isTrue
	got, _ = EvaluateRule(rule, res, nil)
	if got != 1 {
		t.Error("EvaluateRule should match stderr when IncludeStdErr is true")
	}

	// Test invalid regex
	rule2 := playbook.EvaluationRule{Regex: "[["}
	_, err := EvaluateRule(rule2, res, nil)
	if err == nil {
		t.Error("EvaluateRule should return error on invalid regex")
	}

	// Test no rule
	rule3 := playbook.EvaluationRule{}
	got, _ = EvaluateRule(rule3, res, nil)
	if got != 0 {
		t.Errorf("EvaluateRule(no-rule) = %d; want 0", got)
	}

	// Test JS error
	rule4 := playbook.EvaluationRule{Func: "() => { throw new Error('ops') }"}
	_, err = EvaluateRule(rule4, res, nil)
	if err == nil {
		t.Error("EvaluateRule should return error on JS error")
	}

	// Test JS returning number directly
	rule5 := playbook.EvaluationRule{Func: "1"}
	got, _ = EvaluateRule(rule5, res, nil)
	if got != 1 {
		t.Errorf("EvaluateRule(direct-1) = %d; want 1", got)
	}

	// Test JS returning -1
	rule6 := playbook.EvaluationRule{Func: "-1"}
	got, _ = EvaluateRule(rule6, res, nil)
	if got != -1 {
		t.Errorf("EvaluateRule(direct--1) = %d; want -1", got)
	}
}
