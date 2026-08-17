package executor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benedictjohannes/crobe/playbook"
)

type AssertionStatus int

const (
	AssertionStatusPending AssertionStatus = iota
	AssertionStatusRunning
	AssertionStatusPassed
	AssertionStatusFailed
	AssertionStatusCancelled
)

func (s AssertionStatus) String() string {
	switch s {
	case AssertionStatusPending:
		return "pending"
	case AssertionStatusRunning:
		return "running"
	case AssertionStatusPassed:
		return "passed"
	case AssertionStatusFailed:
		return "failed"
	case AssertionStatusCancelled:
		return "cancelled"
	default:
		return fmt.Sprintf("AssertionStatus(%d)", s)
	}
}

func ParseAssertionStatus(s string) (AssertionStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending":
		return AssertionStatusPending, nil
	case "running":
		return AssertionStatusRunning, nil
	case "passed":
		return AssertionStatusPassed, nil
	case "failed":
		return AssertionStatusFailed, nil
	case "cancelled":
		return AssertionStatusCancelled, nil
	default:
		return AssertionStatusPending, fmt.Errorf("unknown AssertionStatus: %q", s)
	}
}

func (s AssertionStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *AssertionStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	parsed, err := ParseAssertionStatus(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

type CommandStatus int

const (
	CommandStatusPending CommandStatus = iota
	CommandStatusRunning
	CommandStatusCompleted
	CommandStatusFailed
	CommandStatusCancelled
)

func (s CommandStatus) String() string {
	switch s {
	case CommandStatusPending:
		return "pending"
	case CommandStatusRunning:
		return "running"
	case CommandStatusCompleted:
		return "completed"
	case CommandStatusFailed:
		return "failed"
	case CommandStatusCancelled:
		return "cancelled"
	default:
		return fmt.Sprintf("CommandStatus(%d)", s)
	}
}

func ParseCommandStatus(s string) (CommandStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending":
		return CommandStatusPending, nil
	case "running":
		return CommandStatusRunning, nil
	case "completed":
		return CommandStatusCompleted, nil
	case "failed":
		return CommandStatusFailed, nil
	case "cancelled":
		return CommandStatusCancelled, nil
	default:
		return CommandStatusPending, fmt.Errorf("unknown CommandStatus: %q", s)
	}
}

func (s CommandStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *CommandStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	parsed, err := ParseCommandStatus(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

type CommandLog struct {
	Exec       playbook.Exec `json:"exec"`
	Status     CommandStatus `json:"status"`
	Timestamps struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"timestamps"`
	Result ExecutionResult `json:"result"`
	Err    error           `json:"err,omitempty"`
}

type AssertionContext struct {
	PlaybookAssertion playbook.Assertion `json:"playbook_assertion"`
	Status            AssertionStatus    `json:"status"`
	Timestamps        struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"timestamps"`
	Passed      bool                   `json:"passed"`
	Score       int                    `json:"score"`
	MinScore    int                    `json:"min_score"`
	Context     map[string]interface{} `json:"context"`
	PreCmdLogs  []CommandLog           `json:"pre_cmd_logs,omitempty"`
	CmdLogs     []CommandLog           `json:"cmd_logs,omitempty"`
	PostCmdLogs []CommandLog           `json:"post_cmd_logs,omitempty"`
	Outputs     []string               `json:"outputs,omitempty"`
}

type SectionContext struct {
	PlaybookSection playbook.Section   `json:"playbook_section"`
	Assertions      []AssertionContext `json:"assertions"`
}

type ExecutionTrace struct {
	Playbook   playbook.Playbook `json:"playbook"`
	Sections   []SectionContext  `json:"sections"`
	Timestamps struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"timestamps"`
	Username    string `json:"username"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	TotalPassed int    `json:"total_passed"`
	TotalFailed int    `json:"total_failed"`
}
