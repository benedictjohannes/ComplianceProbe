package director

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

type recordingObserver struct {
	events       []string
	runID        string
	results      []AssertionProgressResult
	cancelled    bool
	completed    bool
	partialTrace executor.ExecutionTrace
}

func (r *recordingObserver) OnRunStart(runID string, pb *playbook.Playbook) {
	r.runID = runID
	r.events = append(r.events, "OnRunStart")
}

func (r *recordingObserver) OnSectionStart(section playbook.Section, index int, total int) {
	r.events = append(r.events, fmt.Sprintf("OnSectionStart:%s", section.Title))
}

func (r *recordingObserver) OnAssertionStart(assertion playbook.Assertion, index int, total int) {
	r.events = append(r.events, fmt.Sprintf("OnAssertionStart:%s", assertion.Code))
}

func (r *recordingObserver) OnAssertionComplete(assertion playbook.Assertion, result AssertionProgressResult) {
	r.events = append(r.events, fmt.Sprintf("OnAssertionComplete:%s:%s", assertion.Code, result.Status))
	r.results = append(r.results, result)
}

func (r *recordingObserver) OnLog(message string) {
	r.events = append(r.events, fmt.Sprintf("OnLog:%s", message))
}

func (r *recordingObserver) OnRunComplete(trace executor.ExecutionTrace, rep report.FinalReport) {
	r.events = append(r.events, "OnRunComplete")
	r.completed = true
}

func (r *recordingObserver) OnRunCancelled(runID string, partialTrace executor.ExecutionTrace) {
	r.events = append(r.events, "OnRunCancelled")
	r.cancelled = true
	r.partialTrace = partialTrace
}

func (r *recordingObserver) OnRunError(err error) {
	r.events = append(r.events, fmt.Sprintf("OnRunError:%v", err))
}

func TestRunWithObserver_Callbacks(t *testing.T) {
	config := playbook.Playbook{
		Title: "Observer Test Playbook",
		Sections: []playbook.Section{
			{
				Title: "Section 1",
				Assertions: []playbook.Assertion{
					{
						Code:  "A1",
						Title: "Assertion 1",
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "echo 1"}},
						},
					},
				},
			},
		},
	}

	mockExec := func(ctx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "1"}, nil
	}
	runExec = mockExec

	obs := &recordingObserver{}
	trace := RunWithObserver(context.Background(), config, obs, "run-123")

	if obs.runID != "run-123" {
		t.Errorf("expected runID 'run-123', got %q", obs.runID)
	}
	if !obs.completed {
		t.Error("expected OnRunComplete to be called")
	}
	if obs.cancelled {
		t.Error("did not expect OnRunCancelled to be called")
	}
	if len(obs.results) != 1 || obs.results[0].Code != "A1" || obs.results[0].Status != "passed" {
		t.Errorf("unexpected assertion results: %v", obs.results)
	}
	if trace.TotalPassed != 1 {
		t.Errorf("expected 1 passed, got %d", trace.TotalPassed)
	}
}

func TestRunWithObserver_ContextCancellation(t *testing.T) {
	config := playbook.Playbook{
		Title: "Cancel Test Playbook",
		Sections: []playbook.Section{
			{
				Title: "Section 1",
				Assertions: []playbook.Assertion{
					{
						Code:  "A1",
						Title: "Assertion 1",
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "echo 1"}},
						},
					},
					{
						Code:  "A2",
						Title: "Assertion 2",
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "echo 2"}},
						},
					},
				},
			},
			{
				Title: "Section 2",
				Assertions: []playbook.Assertion{
					{
						Code:  "A3",
						Title: "Assertion 3",
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "echo 3"}},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context when A1 finishes
	mockExec := func(execCtx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "echo 1" {
			cancel()
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "1"}, nil
		}
		return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
	}
	runExec = mockExec

	obs := &recordingObserver{}
	trace := RunWithObserver(ctx, config, obs, "cancel-run-456")

	if !obs.cancelled {
		t.Error("expected OnRunCancelled to be called")
	}
	if obs.completed {
		t.Error("did not expect OnRunComplete when cancelled")
	}

	// Verify A1 passed, A2 and A3 were marked cancelled
	if len(trace.Sections) < 2 {
		t.Fatalf("expected 2 sections in trace, got %d", len(trace.Sections))
	}

	// In Section 1: A1 passed, A2 cancelled
	s1 := trace.Sections[0]
	if len(s1.Assertions) != 2 {
		t.Fatalf("expected 2 assertions in s1, got %d", len(s1.Assertions))
	}
	if s1.Assertions[0].Status != executor.AssertionStatusPassed {
		t.Errorf("A1 status = %v, want Passed", s1.Assertions[0].Status)
	}
	if s1.Assertions[1].Status != executor.AssertionStatusCancelled {
		t.Errorf("A2 status = %v, want Cancelled", s1.Assertions[1].Status)
	}

	// In Section 2: A3 cancelled
	s2 := trace.Sections[1]
	if len(s2.Assertions) != 1 {
		t.Fatalf("expected 1 assertion in s2, got %d", len(s2.Assertions))
	}
	if s2.Assertions[0].Status != executor.AssertionStatusCancelled {
		t.Errorf("A3 status = %v, want Cancelled", s2.Assertions[0].Status)
	}
}

func TestRunWithObserver_InFlightCancellation(t *testing.T) {
	config := playbook.Playbook{
		Title: "In-Flight Cancel Playbook",
		Sections: []playbook.Section{
			{
				Title: "Section With Inflight Cancel",
				Assertions: []playbook.Assertion{
					{
						Code:  "INF1",
						Title: "Cancel in PreCmd",
						PreCmds: []playbook.Exec{
							{Script: "echo pre1"},
							{Script: "echo pre2"},
						},
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "echo cmd1"}},
							{Exec: playbook.Exec{Script: "echo cmd2"}},
						},
						PostCmds: []playbook.Exec{
							{Script: "echo post1"},
						},
					},
					{
						Code:  "INF2",
						Title: "Remaining assertion",
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "echo rem"}},
						},
					},
				},
			},
		},
	}

	// 1. Cancel during PreCmds
	ctx1, cancel1 := context.WithCancel(context.Background())
	runExec = func(execCtx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "echo pre1" {
			cancel1()
		}
		return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
	}
	obs1 := &recordingObserver{}
	trace1 := RunWithObserver(ctx1, config, obs1, "cancel-pre")
	if !obs1.cancelled {
		t.Error("expected cancel observer event during pre-cmd cancel")
	}
	if len(trace1.Sections) == 0 || len(trace1.Sections[0].Assertions) == 0 || trace1.Sections[0].Assertions[0].Status != executor.AssertionStatusCancelled {
		t.Errorf("expected INF1 to be cancelled")
	}

	// 2. Cancel during Cmds
	ctx2, cancel2 := context.WithCancel(context.Background())
	runExec = func(execCtx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "echo cmd1" {
			cancel2()
		}
		return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
	}
	obs2 := &recordingObserver{}
	trace2 := RunWithObserver(ctx2, config, obs2, "cancel-cmd")
	if !obs2.cancelled {
		t.Error("expected cancel observer event during cmd cancel")
	}
	if trace2.Sections[0].Assertions[0].Status != executor.AssertionStatusCancelled {
		t.Errorf("expected INF1 to be cancelled during cmd")
	}

	// 3. Cancel during PostCmds
	configPost := playbook.Playbook{
		Sections: []playbook.Section{
			{
				Title: "Post cancel sec",
				Assertions: []playbook.Assertion{
					{
						Code: "POST1",
						Cmds: []playbook.Cmd{{Exec: playbook.Exec{Script: "echo main"}}},
						PostCmds: []playbook.Exec{
							{Script: "echo post1"},
							{Script: "echo post2"},
						},
					},
				},
			},
		},
	}
	ctx3, cancel3 := context.WithCancel(context.Background())
	runExec = func(execCtx context.Context, e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "echo post1" {
			cancel3()
		}
		return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
	}
	obs3 := &recordingObserver{}
	trace3 := RunWithObserver(ctx3, configPost, obs3, "cancel-post")
	if !obs3.cancelled {
		t.Error("expected cancel observer event during post-cmd cancel")
	}
	if trace3.Sections[0].Assertions[0].Status != executor.AssertionStatusCancelled {
		t.Errorf("expected POST1 to be cancelled during post")
	}

	// 4. Cancel prior to section start
	ctx4, cancel4 := context.WithCancel(context.Background())
	cancel4()
	obs4 := &recordingObserver{}
	trace4 := RunWithObserver(ctx4, config, obs4, "cancel-init")
	if !obs4.cancelled {
		t.Error("expected cancel observer event before section start")
	}
	if len(trace4.Sections[0].Assertions) != 2 || trace4.Sections[0].Assertions[0].Status != executor.AssertionStatusCancelled {
		t.Errorf("expected assertions unwound as cancelled")
	}
}

func TestConsoleObserver(t *testing.T) {
	var buf bytes.Buffer
	obs := NewConsoleObserver(&buf)

	obs.OnRunStart("run-1", &playbook.Playbook{})
	obs.OnSectionStart(playbook.Section{Title: "Security Baseline"}, 1, 3)
	if !strings.Contains(buf.String(), "Processing Section: Security Baseline") {
		t.Errorf("unexpected section start output: %s", buf.String())
	}
	buf.Reset()

	obs.OnAssertionStart(playbook.Assertion{Code: "A1"}, 1, 1)

	// Test Passed
	obs.OnAssertionComplete(
		playbook.Assertion{Title: "Check SSH Root Login"},
		AssertionProgressResult{Status: "passed", Passed: true, Score: 10, MinScore: 10},
	)
	if !strings.Contains(buf.String(), "Check SSH Root Login: ✅ PASS (Score: 10/10)") {
		t.Errorf("unexpected pass output: %s", buf.String())
	}
	buf.Reset()

	// Test Failed
	obs.OnAssertionComplete(
		playbook.Assertion{Title: "Check Firewall Enabled"},
		AssertionProgressResult{Status: "failed", Passed: false, Score: 0, MinScore: 10},
	)
	if !strings.Contains(buf.String(), "Check Firewall Enabled: ❌ FAIL (Score: 0/10)") {
		t.Errorf("unexpected fail output: %s", buf.String())
	}
	buf.Reset()

	// Test Cancelled
	obs.OnAssertionComplete(
		playbook.Assertion{Title: "Check Kernel Params"},
		AssertionProgressResult{Status: "cancelled", Passed: false, Score: 0, MinScore: 5},
	)
	if !strings.Contains(buf.String(), "Check Kernel Params: ⚠️ CANCELLED (Score: 0/5)") {
		t.Errorf("unexpected cancelled output: %s", buf.String())
	}
	buf.Reset()

	// Test Log
	obs.OnLog("streaming log line")
	if !strings.Contains(buf.String(), "streaming log line") {
		t.Errorf("unexpected log output: %s", buf.String())
	}
	buf.Reset()

	obs.OnRunComplete(executor.ExecutionTrace{}, report.FinalReport{})

	// Test Cancelled Run
	obs.OnRunCancelled("run-1", executor.ExecutionTrace{})
	if !strings.Contains(buf.String(), "Execution cancelled") {
		t.Errorf("unexpected cancel output: %s", buf.String())
	}
	buf.Reset()

	// Test Error Run
	obs.OnRunError(fmt.Errorf("connection refused"))
	if !strings.Contains(buf.String(), "Execution error: connection refused") {
		t.Errorf("unexpected error output: %s", buf.String())
	}

	// Test default constructor with stdout
	defaultObs := NewConsoleObserver()
	if defaultObs == nil || defaultObs.out != os.Stdout {
		t.Errorf("expected default ConsoleObserver to write to os.Stdout")
	}
}

func TestNopObserver(t *testing.T) {
	obs := &NopObserver{}
	obs.OnRunStart("run-1", &playbook.Playbook{})
	obs.OnSectionStart(playbook.Section{}, 1, 1)
	obs.OnAssertionStart(playbook.Assertion{}, 1, 1)
	obs.OnAssertionComplete(playbook.Assertion{}, AssertionProgressResult{})
	obs.OnLog("log")
	obs.OnRunComplete(executor.ExecutionTrace{}, report.FinalReport{})
	obs.OnRunCancelled("run-1", executor.ExecutionTrace{})
	obs.OnRunError(fmt.Errorf("err"))
}

func TestRunWithObserver_NilOptions(t *testing.T) {
	config := playbook.Playbook{
		Title: "Nil Options Test",
	}
	// Verify nil ctx and nil observer don't panic
	trace := RunWithObserver(nil, config, nil, "")
	if trace.Playbook.Title != "Nil Options Test" {
		t.Errorf("expected trace playbook title to match")
	}
}
