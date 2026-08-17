package director

import (
	"fmt"
	"io"
	"os"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

// AssertionProgressResult represents the outcome of an executed assertion dispatched to observers.
type AssertionProgressResult struct {
	Code       string `json:"code"`
	Status     string `json:"status"` // "passed" | "failed" | "cancelled"
	Passed     bool   `json:"passed"`
	Score      int    `json:"score"`
	MinScore   int    `json:"min_score"`
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"` // Strictly pre-redacted output
}

// ExecutionObserver receives lifecycle events during playbook execution.
type ExecutionObserver interface {
	OnRunStart(runID string, pb *playbook.Playbook)
	OnSectionStart(section playbook.Section, index int, total int)
	OnAssertionStart(assertion playbook.Assertion, index int, total int)
	OnAssertionComplete(assertion playbook.Assertion, result AssertionProgressResult)
	OnLog(message string)
	OnRunComplete(trace executor.ExecutionTrace, rep report.FinalReport)
	OnRunCancelled(runID string, partialTrace executor.ExecutionTrace)
	OnRunError(err error)
}

// ConsoleObserver implements ExecutionObserver for CLI terminal output.
type ConsoleObserver struct {
	out io.Writer
}

// NewConsoleObserver creates a new ConsoleObserver writing to out, or os.Stdout by default.
func NewConsoleObserver(out ...io.Writer) *ConsoleObserver {
	var w io.Writer = os.Stdout
	if len(out) > 0 && out[0] != nil {
		w = out[0]
	}
	return &ConsoleObserver{out: w}
}

func (c *ConsoleObserver) OnRunStart(runID string, pb *playbook.Playbook) {}

func (c *ConsoleObserver) OnSectionStart(section playbook.Section, index int, total int) {
	fmt.Fprintf(c.out, "  Processing Section: %s\n", section.Title)
}

func (c *ConsoleObserver) OnAssertionStart(assertion playbook.Assertion, index int, total int) {}

func (c *ConsoleObserver) OnAssertionComplete(assertion playbook.Assertion, result AssertionProgressResult) {
	status := "✅ PASS"
	if result.Status == "cancelled" {
		status = "⚠️ CANCELLED"
	} else if !result.Passed {
		status = "❌ FAIL"
	}
	fmt.Fprintf(c.out, "    - %s: %s (Score: %d/%d)\n", assertion.Title, status, result.Score, result.MinScore)
}

func (c *ConsoleObserver) OnLog(message string) {
	fmt.Fprintln(c.out, message)
}

func (c *ConsoleObserver) OnRunComplete(trace executor.ExecutionTrace, rep report.FinalReport) {}

func (c *ConsoleObserver) OnRunCancelled(runID string, partialTrace executor.ExecutionTrace) {
	fmt.Fprintln(c.out, "  ⚠️ Execution cancelled")
}

func (c *ConsoleObserver) OnRunError(err error) {
	fmt.Fprintf(c.out, "  ❌ Execution error: %v\n", err)
}

// NopObserver is a no-op implementation of ExecutionObserver for tests.
type NopObserver struct{}

func (n *NopObserver) OnRunStart(runID string, pb *playbook.Playbook)                               {}
func (n *NopObserver) OnSectionStart(section playbook.Section, index int, total int)               {}
func (n *NopObserver) OnAssertionStart(assertion playbook.Assertion, index int, total int)        {}
func (n *NopObserver) OnAssertionComplete(assertion playbook.Assertion, result AssertionProgressResult) {}
func (n *NopObserver) OnLog(message string)                                                        {}
func (n *NopObserver) OnRunComplete(trace executor.ExecutionTrace, rep report.FinalReport)        {}
func (n *NopObserver) OnRunCancelled(runID string, partialTrace executor.ExecutionTrace)          {}
func (n *NopObserver) OnRunError(err error)                                                         {}
