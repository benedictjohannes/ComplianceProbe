package playbook

import (
	"fmt"
	"strings"
)

// ValidationError represents a structured diagnostic error in playbook validation.
type ValidationError struct {
	Path    string `json:"path"`    // e.g. "sections[0].assertions[1].code", "report_destination.https.url"
	Code    string `json:"code"`    // e.g. "MISSING_CODE", "DUPLICATE_CODE", "PROHIBITED_AGENT_FUNC"
	Message string `json:"message"` // Human-readable explanation
}

func (v ValidationError) Error() string {
	if v.Path != "" {
		return fmt.Sprintf("%s: %s", v.Path, v.Message)
	}
	return v.Message
}

// ValidationErrors is a slice of ValidationError that implements the error interface.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var msgs []string
	for _, v := range ve {
		msgs = append(msgs, v.Error())
	}
	return strings.Join(msgs, "\n")
}

// Validate validates the playbook structure and semantics hierarchically.
// When isAgent is true, external script references (funcFile, shellFuncFile) are disallowed.
func (p Playbook) Validate(isAgent bool) ValidationErrors {
	var errs ValidationErrors

	if strings.TrimSpace(p.Title) == "" {
		errs = append(errs, ValidationError{
			Path:    "title",
			Code:    "MISSING_TITLE",
			Message: "playbook title is required",
		})
	} else if len(strings.TrimSpace(p.Title)) < 3 {
		errs = append(errs, ValidationError{
			Path:    "title",
			Code:    "INVALID_TITLE",
			Message: "playbook title must be at least 3 characters",
		})
	}

	if p.ReportDestination == ReportDestinationHTTPS {
		if p.ReportDestinationHTTPS == nil {
			errs = append(errs, ValidationError{
				Path:    "report_destination.https",
				Code:    "MISSING_HTTPS_CONFIG",
				Message: "https configuration is required when report destination is https",
			})
		} else {
			if strings.TrimSpace(p.ReportDestinationHTTPS.URL) == "" {
				errs = append(errs, ValidationError{
					Path:    "report_destination.https.url",
					Code:    "MISSING_URL",
					Message: "https destination URL is required",
				})
			} else if !strings.HasPrefix(p.ReportDestinationHTTPS.URL, "https://") {
				errs = append(errs, ValidationError{
					Path:    "report_destination.https.url",
					Code:    "INVALID_URL",
					Message: "https destination URL must start with https://",
				})
			}
			if p.ReportDestinationHTTPS.Format != "" &&
				p.ReportDestinationHTTPS.Format != ReportFormatMultipart &&
				p.ReportDestinationHTTPS.Format != ReportFormatJSON {
				errs = append(errs, ValidationError{
					Path:    "report_destination.https.format",
					Code:    "INVALID_FORMAT",
					Message: "report format must be 'json' or 'multipart'",
				})
			}
		}
	} else if p.ReportDestination != "" && p.ReportDestination != ReportDestinationFolder {
		errs = append(errs, ValidationError{
			Path:    "report_destination",
			Code:    "INVALID_DESTINATION",
			Message: "report destination must be 'folder' or 'https'",
		})
	}

	if len(p.Sections) == 0 {
		errs = append(errs, ValidationError{
			Path:    "sections",
			Code:    "EMPTY_SECTIONS",
			Message: "playbook must contain at least one section",
		})
	} else {
		seenCodes := make(map[string]string)
		for i, section := range p.Sections {
			errs = append(errs, section.Validate(i, seenCodes, isAgent)...)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Validate validates the section structure and delegates to its assertions.
func (s Section) Validate(sectionIdx int, seenCodes map[string]string, isAgent bool) ValidationErrors {
	var errs ValidationErrors
	sectionPath := fmt.Sprintf("sections[%d]", sectionIdx)

	if strings.TrimSpace(s.Title) == "" {
		errs = append(errs, ValidationError{
			Path:    fmt.Sprintf("%s.title", sectionPath),
			Code:    "MISSING_TITLE",
			Message: "section title is required",
		})
	} else if len(strings.TrimSpace(s.Title)) < 3 {
		errs = append(errs, ValidationError{
			Path:    fmt.Sprintf("%s.title", sectionPath),
			Code:    "INVALID_TITLE",
			Message: "section title must be at least 3 characters",
		})
	}

	if len(s.Assertions) == 0 {
		errs = append(errs, ValidationError{
			Path:    fmt.Sprintf("%s.assertions", sectionPath),
			Code:    "EMPTY_ASSERTIONS",
			Message: "section must contain at least one assertion",
		})
	} else {
		for j, assertion := range s.Assertions {
			errs = append(errs, assertion.Validate(sectionIdx, j, seenCodes, isAgent)...)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Validate validates an individual assertion, enforces code uniqueness, and checks agent constraints.
func (a Assertion) Validate(sectionIdx, assertionIdx int, seenCodes map[string]string, isAgent bool) ValidationErrors {
	var errs ValidationErrors
	assertionPath := fmt.Sprintf("sections[%d].assertions[%d]", sectionIdx, assertionIdx)

	if strings.TrimSpace(a.Code) == "" {
		errs = append(errs, ValidationError{
			Path:    fmt.Sprintf("%s.code", assertionPath),
			Code:    "MISSING_CODE",
			Message: "assertion code is required",
		})
	} else {
		if prevPath, found := seenCodes[a.Code]; found {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("%s.code", assertionPath),
				Code:    "DUPLICATE_CODE",
				Message: fmt.Sprintf("duplicate assertion code '%s' previously declared at %s", a.Code, prevPath),
			})
		} else {
			seenCodes[a.Code] = fmt.Sprintf("%s.code", assertionPath)
		}
	}

	if isAgent {
		for k, exec := range a.PreCmds {
			if exec.FuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.preCmds[%d].funcFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains funcFile in preCmd", a.Code),
				})
			}
			if exec.ShellFuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.preCmds[%d].shellFuncFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains shellFuncFile in preCmd", a.Code),
				})
			}
		}

		for k, cmd := range a.Cmds {
			if cmd.Exec.FuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.cmds[%d].exec.funcFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains funcFile in cmd", a.Code),
				})
			}
			if cmd.Exec.ShellFuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.cmds[%d].exec.shellFuncFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains shellFuncFile in cmd", a.Code),
				})
			}
			if cmd.StdOutRule.FuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.cmds[%d].stdOutRule.funcFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains funcFile in stdOutRule", a.Code),
				})
			}
			if cmd.StdErrRule.FuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.cmds[%d].stdErrRule.funcFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains funcFile in stdErrRule", a.Code),
				})
			}
			for gIdx, g := range cmd.Exec.Gather {
				if g.FuncFile != "" {
					errs = append(errs, ValidationError{
						Path:    fmt.Sprintf("%s.cmds[%d].exec.gather[%d].funcFile", assertionPath, k, gIdx),
						Code:    "PROHIBITED_AGENT_FUNC",
						Message: fmt.Sprintf("agent error: assertion %s contains funcFile in gather", a.Code),
					})
				}
			}
		}

		for k, exec := range a.PostCmds {
			if exec.FuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.postCmds[%d].funcFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains funcFile in postCmd", a.Code),
				})
			}
			if exec.ShellFuncFile != "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.postCmds[%d].shellFuncFile", assertionPath, k),
					Code:    "PROHIBITED_AGENT_FUNC",
					Message: fmt.Sprintf("agent error: assertion %s contains shellFuncFile in postCmd", a.Code),
				})
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
