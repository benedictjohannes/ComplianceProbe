//go:build builder

package main

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestJSExecution(t *testing.T) {
	context := map[string]interface{}{"foo": "bar"}
	code := "({ assertionContext }) => assertionContext.foo + 'baz'"
	got, err := runJS(code, context)
	if err != nil {
		t.Fatalf("runJS() error: %v", err)
	}
	expected := "barbaz"
	if got != expected {
		t.Errorf("runJS() = %q; want %q", got, expected)
	}
}

func TestPerformGatherJS(t *testing.T) {
	g := GatherSpec{
		Key:  "version",
		Func: "(stdout) => stdout.match(/v(\\d+)/)[1]",
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

func TestEvaluateRuleJS(t *testing.T) {
	rule := EvaluationRule{
		Func: "(stdout) => stdout.includes('PASS') ? 1 : -1",
	}
	res := ExecutionResult{Stdout: "Result: PASS"}
	context := make(map[string]interface{})
	got, err := evaluateRule(rule, res, context)
	if err != nil {
		t.Fatalf("evaluateRule() error: %v", err)
	}
	if got != 1 {
		t.Errorf("evaluateRule() = %d; want %d", got, 1)
	}
}

func TestBuilderExportDefault(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(10000)
	fileName := fmt.Sprintf("playbook_js_concept_%d.ts", randomNumber)

	content := `/**
 * Checks if a haystack contains a needle from context
 */
export default (haystack: string, _: string, assertionContext: {[key: string]: string}) => {
    return haystack.includes(assertionContext.needle) ? 1 : -1
}`

	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	defer os.Remove(fileName)

	code, err := transpile(fileName)
	if err != nil {
		t.Fatalf("Transpile failed: %v", err)
	}

	rule := EvaluationRule{Func: code}
	res := ExecutionResult{Stdout: "The quick brown fox"}
	ctx := map[string]interface{}{"needle": "brown"}

	got, err := evaluateRule(rule, res, ctx)
	if err != nil {
		t.Fatalf("evaluateRule failed: %v", err)
	}
	if got != 1 {
		t.Errorf("Expected 1 (pass), got %d", got)
	}

	// Test negative case
	ctx["needle"] = "blue"
	got, err = evaluateRule(rule, res, ctx)
	if got != -1 {
		t.Errorf("Expected -1 (fail), got %d", got)
	}
}

func TestBuilderNakedFunction(t *testing.T) {
	rand.Seed(time.Now().UnixNano() + 1)
	randomNumber := rand.Intn(10000)
	fileName := fmt.Sprintf("playbook_js_naked_%d.ts", randomNumber)

	content := `(haystack: string, _: string, assertionContext: {[key: string]: string}) => {
    return haystack.includes(assertionContext.needle) ? 1 : -1
}`

	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	defer os.Remove(fileName)

	code, err := transpile(fileName)
	if err != nil {
		t.Fatalf("Transpile failed: %v", err)
	}

	rule := EvaluationRule{Func: code}
	res := ExecutionResult{Stdout: "The quick brown fox"}
	ctx := map[string]interface{}{"needle": "fox"}

	got, err := evaluateRule(rule, res, ctx)
	if err != nil {
		t.Fatalf("evaluateRule failed: %v", err)
	}
	if got != 1 {
		t.Errorf("Expected 1 (pass), got %d", got)
	}
}
