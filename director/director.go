package director

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

var (
	runExec = executor.RunExec
	goos    = runtime.GOOS
)

// Run executes a playbook using the default ConsoleObserver.
func Run(ctx context.Context, config playbook.Playbook) executor.ExecutionTrace {
	return RunWithObserver(ctx, config, NewConsoleObserver(), "")
}

// RunWithObserver executes a playbook, streaming events to the provided ExecutionObserver and respecting context cancellation.
func RunWithObserver(ctx context.Context, config playbook.Playbook, observer ExecutionObserver, runID string) executor.ExecutionTrace {
	if ctx == nil {
		ctx = context.Background()
	}
	if observer == nil {
		observer = &NopObserver{}
	}

	now := time.Now()
	osName := goos
	if osName == "darwin" {
		osName = "mac"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	trace := executor.ExecutionTrace{
		Playbook: config,
		Username: username,
		OS:       osName,
		Arch:     runtime.GOARCH,
	}
	trace.Timestamps.Start = now

	observer.OnRunStart(runID, &config)

	totalPassed := 0
	totalFailed := 0
	totalSections := len(config.Sections)

	for sectionIdx, section := range config.Sections {
		// Check for cancellation before starting a section
		if ctx.Err() != nil {
			unwindRemaining(sectionIdx, 0, config, &trace, observer, runID)
			return trace
		}

		observer.OnSectionStart(section, sectionIdx+1, totalSections)
		trace.Sections = append(trace.Sections, executor.SectionContext{
			PlaybookSection: section,
		})
		sCtx := &trace.Sections[len(trace.Sections)-1]

		totalAssertions := len(section.Assertions)
		for assertionIdx, assertion := range section.Assertions {
			// Check for cancellation before starting an assertion
			if ctx.Err() != nil {
				unwindRemaining(sectionIdx, assertionIdx, config, &trace, observer, runID)
				return trace
			}

			observer.OnAssertionStart(assertion, assertionIdx+1, totalAssertions)
			start := time.Now()
			contextMap := make(map[string]interface{})
			score := 0
			cancelled := false

			assCtx := executor.AssertionContext{
				PlaybookAssertion: assertion,
				Status:            executor.AssertionStatusRunning,
				Context:           make(map[string]interface{}),
			}
			assCtx.Timestamps.Start = start

			// 1. Pre-Commands
			for _, exec := range assertion.PreCmds {
				if ctx.Err() != nil {
					cancelled = true
					break
				}
				cmdLog := executor.CommandLog{
					Exec:   exec,
					Status: executor.CommandStatusRunning,
				}
				cmdLog.Timestamps.Start = time.Now()
				res, err := runExec(ctx, &exec, contextMap)
				cmdLog.Timestamps.End = time.Now()
				cmdLog.Result = res
				cmdLog.Err = err
				if err != nil {
					cmdLog.Status = executor.CommandStatusFailed
					observer.OnLog(fmt.Sprintf("      ⚠️ PreCmd Error (%s): %v", assertion.Code, err))
				} else {
					cmdLog.Status = executor.CommandStatusCompleted
				}
				assCtx.PreCmdLogs = append(assCtx.PreCmdLogs, cmdLog)
			}

			// 2. Main Commands
			var outputs []string
			if !cancelled {
				for _, cmd := range assertion.Cmds {
					if ctx.Err() != nil {
						cancelled = true
						break
					}
					cmdLog := executor.CommandLog{
						Exec:   cmd.Exec,
						Status: executor.CommandStatusRunning,
					}
					cmdLog.Timestamps.Start = time.Now()
					res, err := runExec(ctx, &cmd.Exec, contextMap)
					cmdLog.Timestamps.End = time.Now()
					cmdLog.Result = res
					cmdLog.Err = err

					if err != nil {
						cmdLog.Status = executor.CommandStatusFailed
						score += cmd.GetFailScore()
						assCtx.CmdLogs = append(assCtx.CmdLogs, cmdLog)
						continue
					}
					cmdLog.Status = executor.CommandStatusCompleted
					assCtx.CmdLogs = append(assCtx.CmdLogs, cmdLog)

					// Strictly pre-redacted outputs
					if cmd.Exec.ExcludeFromReport {
						outputs = append(outputs, "[REDACTED]")
					} else {
						if res.Stderr != "" {
							if len(res.Stdout) > 0 {
								outputs = append(outputs, "# --- STDOUT ---")
								outputs = append(outputs, res.Stdout)
							}
							outputs = append(outputs, "# --- STDERR ---")
							outputs = append(outputs, res.Stderr)
						} else {
							outputs = append(outputs, res.Stdout)
						}
					}

					// Evaluation
					result := 0
					foundRule := false
					for _, rule := range cmd.ExitCodeRules {
						match := true
						if rule.Min != nil && res.ExitCode < *rule.Min {
							match = false
						}
						if rule.Max != nil && res.ExitCode > *rule.Max {
							match = false
						}
						if match {
							result = rule.Result
							foundRule = true
							break
						}
					}

					if !foundRule {
						if res.ExitCode == 0 {
							result = 1
						} else {
							result = -1
						}
					}

					if cmd.StdOutRule.Regex != "" || cmd.StdOutRule.Func != "" {
						verdict, _ := executor.EvaluateRule(cmd.StdOutRule, res, contextMap)
						if verdict != 0 {
							result = verdict
						}
					}
					if cmd.StdErrRule.Regex != "" || cmd.StdErrRule.Func != "" {
						verdict, _ := executor.EvaluateRule(cmd.StdErrRule, res, contextMap)
						if verdict != 0 {
							result = verdict
						}
					}

					switch result {
					case 1:
						score += cmd.GetPassScore()
					case -1:
						score += cmd.GetFailScore()
					}
				}
			}

			// 3. Post-Commands
			if !cancelled {
				for _, exec := range assertion.PostCmds {
					if ctx.Err() != nil {
						cancelled = true
						break
					}
					cmdLog := executor.CommandLog{
						Exec:   exec,
						Status: executor.CommandStatusRunning,
					}
					cmdLog.Timestamps.Start = time.Now()
					res, err := runExec(ctx, &exec, contextMap)
					cmdLog.Timestamps.End = time.Now()
					cmdLog.Result = res
					cmdLog.Err = err
					if err != nil {
						cmdLog.Status = executor.CommandStatusFailed
						observer.OnLog(fmt.Sprintf("      ⚠️ PostCmd Error (%s): %v", assertion.Code, err))
					} else {
						cmdLog.Status = executor.CommandStatusCompleted
					}
					assCtx.PostCmdLogs = append(assCtx.PostCmdLogs, cmdLog)
				}
			}

			assCtx.Outputs = outputs

			if cancelled {
				assCtx.Status = executor.AssertionStatusCancelled
				assCtx.Timestamps.End = time.Now()
				sCtx.Assertions = append(sCtx.Assertions, assCtx)
				observer.OnAssertionComplete(assertion, AssertionProgressResult{
					Code:       assertion.Code,
					Status:     "cancelled",
					Passed:     false,
					Score:      score,
					MinScore:   assertion.GetMinPassingScore(),
					DurationMs: assCtx.Timestamps.End.Sub(assCtx.Timestamps.Start).Milliseconds(),
					Output:     strings.Join(outputs, "\n"),
				})
				unwindRemaining(sectionIdx, assertionIdx+1, config, &trace, observer, runID)
				return trace
			}

			passed := score >= assertion.GetMinPassingScore()
			if passed {
				assCtx.Status = executor.AssertionStatusPassed
				totalPassed++
			} else {
				assCtx.Status = executor.AssertionStatusFailed
				totalFailed++
			}

			assCtx.Passed = passed
			assCtx.Score = score
			assCtx.MinScore = assertion.GetMinPassingScore()
			assCtx.Timestamps.End = time.Now()

			// Determine which keys to exclude from report
			excludedKeys := make(map[string]bool)
			for _, exec := range assertion.PreCmds {
				for _, g := range exec.Gather {
					if g.ExcludeFromReport {
						excludedKeys[g.Key] = true
					}
				}
			}
			for _, cmd := range assertion.Cmds {
				for _, g := range cmd.Exec.Gather {
					if g.ExcludeFromReport {
						excludedKeys[g.Key] = true
					}
				}
			}
			for _, exec := range assertion.PostCmds {
				for _, g := range exec.Gather {
					if g.ExcludeFromReport {
						excludedKeys[g.Key] = true
					}
				}
			}

			for k, v := range contextMap {
				if !excludedKeys[k] {
					assCtx.Context[k] = v
				}
			}

			durationMs := assCtx.Timestamps.End.Sub(assCtx.Timestamps.Start).Milliseconds()
			statusStr := "passed"
			if !passed {
				statusStr = "failed"
			}
			observer.OnAssertionComplete(assertion, AssertionProgressResult{
				Code:       assertion.Code,
				Status:     statusStr,
				Passed:     passed,
				Score:      score,
				MinScore:   assCtx.MinScore,
				DurationMs: durationMs,
				Output:     strings.Join(outputs, "\n"),
			})

			sCtx.Assertions = append(sCtx.Assertions, assCtx)
		}
	}

	trace.Timestamps.End = time.Now()
	trace.TotalPassed = totalPassed
	trace.TotalFailed = totalFailed

	rep := report.GenerateReport(trace)
	observer.OnRunComplete(trace, rep.Structured)

	return trace
}

// unwindRemaining marks all unexecuted assertions as AssertionStatusCancelled and dispatches OnRunCancelled.
func unwindRemaining(startSectionIdx, startAssertionIdx int, config playbook.Playbook, trace *executor.ExecutionTrace, observer ExecutionObserver, runID string) {
	for sIdx := startSectionIdx; sIdx < len(config.Sections); sIdx++ {
		section := config.Sections[sIdx]
		var sCtx *executor.SectionContext

		if sIdx < len(trace.Sections) {
			sCtx = &trace.Sections[sIdx]
		} else {
			trace.Sections = append(trace.Sections, executor.SectionContext{
				PlaybookSection: section,
			})
			sCtx = &trace.Sections[len(trace.Sections)-1]
		}

		aStart := 0
		if sIdx == startSectionIdx {
			aStart = startAssertionIdx
		}

		for aIdx := aStart; aIdx < len(section.Assertions); aIdx++ {
			assertion := section.Assertions[aIdx]
			assCtx := executor.AssertionContext{
				PlaybookAssertion: assertion,
				Status:            executor.AssertionStatusCancelled,
				MinScore:          assertion.GetMinPassingScore(),
			}
			sCtx.Assertions = append(sCtx.Assertions, assCtx)
			observer.OnAssertionComplete(assertion, AssertionProgressResult{
				Code:     assertion.Code,
				Status:   "cancelled",
				Passed:   false,
				Score:    0,
				MinScore: assertion.GetMinPassingScore(),
			})
		}
	}

	trace.Timestamps.End = time.Now()
	observer.OnRunCancelled(runID, *trace)
}
