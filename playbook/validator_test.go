package playbook

import (
	"strings"
	"testing"
)

func TestPlaybook_Validate_Success(t *testing.T) {
	tests := []struct {
		name    string
		config  Playbook
		isAgent bool
	}{
		{
			name: "Valid Basic Config",
			config: Playbook{
				Title: "Test Playbook",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "A01", Title: "Assertion 1"},
						},
					},
				},
			},
			isAgent: false,
		},
		{
			name: "Valid Agent Config with Inlined Scripts",
			config: Playbook{
				Title: "Agent Playbook",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{
								Code:  "A01",
								Title: "Assertion 1",
								PreCmds: []Exec{
									{Script: "echo 1"},
								},
								Cmds: []Cmd{
									{
										Exec: Exec{Script: "echo 2"},
										StdOutRule: EvaluationRule{
											Regex: ".*",
										},
									},
								},
								PostCmds: []Exec{
									{Script: "echo 3"},
								},
							},
						},
					},
				},
			},
			isAgent: true,
		},
		{
			name: "Valid HTTPS Destination Config",
			config: Playbook{
				Title:             "HTTPS Playbook",
				ReportDestination: ReportDestinationHTTPS,
				ReportDestinationHTTPS: &ReportDestinationConfig{
					URL:    "https://example.com/api/reports",
					Format: ReportFormatJSON,
				},
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "A01", Title: "Assertion 1"},
						},
					},
				},
			},
			isAgent: false,
		},
		{
			name: "Valid Folder Destination Config",
			config: Playbook{
				Title:                   "Folder Playbook",
				ReportDestination:       ReportDestinationFolder,
				ReportDestinationFolder: "/custom/reports",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "A01", Title: "Assertion 1"},
						},
					},
				},
			},
			isAgent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate(tt.isAgent)
			if len(errs) > 0 {
				t.Fatalf("expected no validation errors, got: %v", errs)
			}
		})
	}
}

func TestPlaybook_Validate_TitleErrors(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		wantCode string
		wantPath string
	}{
		{
			name:     "Missing Title",
			title:    "",
			wantCode: "MISSING_TITLE",
			wantPath: "title",
		},
		{
			name:     "Whitespace Title",
			title:    "   ",
			wantCode: "MISSING_TITLE",
			wantPath: "title",
		},
		{
			name:     "Title Too Short",
			title:    "AB",
			wantCode: "INVALID_TITLE",
			wantPath: "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := Playbook{
				Title: tt.title,
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "A01", Title: "Assertion 1"},
						},
					},
				},
			}
			errs := pb.Validate(false)
			if len(errs) == 0 {
				t.Fatalf("expected validation errors, got none")
			}
			found := false
			for _, e := range errs {
				if e.Code == tt.wantCode && e.Path == tt.wantPath {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error with code %q and path %q, got: %v", tt.wantCode, tt.wantPath, errs)
			}
		})
	}
}

func TestPlaybook_Validate_SectionsErrors(t *testing.T) {
	pb := Playbook{
		Title:    "Valid Title",
		Sections: []Section{},
	}
	errs := pb.Validate(false)
	if len(errs) != 1 || errs[0].Code != "EMPTY_SECTIONS" || errs[0].Path != "sections" {
		t.Fatalf("expected EMPTY_SECTIONS at 'sections', got: %v", errs)
	}
}

func TestPlaybook_Validate_SectionErrors(t *testing.T) {
	tests := []struct {
		name     string
		section  Section
		wantCode string
		wantPath string
	}{
		{
			name: "Missing Section Title",
			section: Section{
				Title: "",
				Assertions: []Assertion{
					{Code: "A01", Title: "Assertion 1"},
				},
			},
			wantCode: "MISSING_TITLE",
			wantPath: "sections[0].title",
		},
		{
			name: "Section Title Too Short",
			section: Section{
				Title: "S1",
				Assertions: []Assertion{
					{Code: "A01", Title: "Assertion 1"},
				},
			},
			wantCode: "INVALID_TITLE",
			wantPath: "sections[0].title",
		},
		{
			name: "Section Empty Assertions",
			section: Section{
				Title:      "Valid Section Title",
				Assertions: []Assertion{},
			},
			wantCode: "EMPTY_ASSERTIONS",
			wantPath: "sections[0].assertions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := Playbook{
				Title:    "Valid Title",
				Sections: []Section{tt.section},
			}
			errs := pb.Validate(false)
			if len(errs) == 0 {
				t.Fatalf("expected validation errors, got none")
			}
			found := false
			for _, e := range errs {
				if e.Code == tt.wantCode && e.Path == tt.wantPath {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error with code %q and path %q, got: %v", tt.wantCode, tt.wantPath, errs)
			}
		})
	}
}

func TestPlaybook_Validate_AssertionErrors(t *testing.T) {
	pb := Playbook{
		Title: "Valid Title",
		Sections: []Section{
			{
				Title: "Section 1",
				Assertions: []Assertion{
					{Code: "", Title: "Missing Code"},
				},
			},
		},
	}
	errs := pb.Validate(false)
	if len(errs) != 1 || errs[0].Code != "MISSING_CODE" || errs[0].Path != "sections[0].assertions[0].code" {
		t.Fatalf("expected MISSING_CODE at 'sections[0].assertions[0].code', got: %v", errs)
	}
}

func TestPlaybook_Validate_DuplicateCodes(t *testing.T) {
	t.Run("Duplicate code within same section", func(t *testing.T) {
		pb := Playbook{
			Title: "Test Duplicate",
			Sections: []Section{
				{
					Title: "Section 1",
					Assertions: []Assertion{
						{Code: "DUP01", Title: "A1"},
						{Code: "DUP01", Title: "A2"},
					},
				},
			},
		}
		errs := pb.Validate(false)
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got: %v", errs)
		}
		e := errs[0]
		if e.Code != "DUPLICATE_CODE" || e.Path != "sections[0].assertions[1].code" {
			t.Errorf("unexpected error: %v", e)
		}
		if !strings.Contains(e.Message, "sections[0].assertions[0].code") {
			t.Errorf("expected message to cite origin path 'sections[0].assertions[0].code', got %q", e.Message)
		}
	})

	t.Run("Duplicate code across different sections", func(t *testing.T) {
		pb := Playbook{
			Title: "Test Duplicate Across Sections",
			Sections: []Section{
				{
					Title: "Section 1",
					Assertions: []Assertion{
						{Code: "CROSS_DUP", Title: "A1"},
					},
				},
				{
					Title: "Section 2",
					Assertions: []Assertion{
						{Code: "CROSS_DUP", Title: "A2"},
					},
				},
			},
		}
		errs := pb.Validate(false)
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got: %v", errs)
		}
		e := errs[0]
		if e.Code != "DUPLICATE_CODE" || e.Path != "sections[1].assertions[0].code" {
			t.Errorf("unexpected error: %v", e)
		}
		if !strings.Contains(e.Message, "sections[0].assertions[0].code") {
			t.Errorf("expected message to cite origin path 'sections[0].assertions[0].code', got %q", e.Message)
		}
	})
}

func TestPlaybook_Validate_ReportDestinationErrors(t *testing.T) {
	tests := []struct {
		name     string
		config   Playbook
		wantCode string
		wantPath string
	}{
		{
			name: "Invalid destination type",
			config: Playbook{
				Title:             "Test",
				ReportDestination: "ftp",
				Sections: []Section{
					{Title: "Section 1", Assertions: []Assertion{{Code: "A01"}}},
				},
			},
			wantCode: "INVALID_DESTINATION",
			wantPath: "report_destination",
		},
		{
			name: "HTTPS destination with nil config",
			config: Playbook{
				Title:                  "Test",
				ReportDestination:      ReportDestinationHTTPS,
				ReportDestinationHTTPS: nil,
				Sections: []Section{
					{Title: "Section 1", Assertions: []Assertion{{Code: "A01"}}},
				},
			},
			wantCode: "MISSING_HTTPS_CONFIG",
			wantPath: "report_destination.https",
		},
		{
			name: "HTTPS destination with missing URL",
			config: Playbook{
				Title:             "Test",
				ReportDestination: ReportDestinationHTTPS,
				ReportDestinationHTTPS: &ReportDestinationConfig{
					URL: "",
				},
				Sections: []Section{
					{Title: "Section 1", Assertions: []Assertion{{Code: "A01"}}},
				},
			},
			wantCode: "MISSING_URL",
			wantPath: "report_destination.https.url",
		},
		{
			name: "HTTPS destination with non-https URL",
			config: Playbook{
				Title:             "Test",
				ReportDestination: ReportDestinationHTTPS,
				ReportDestinationHTTPS: &ReportDestinationConfig{
					URL: "http://insecure.example.com/api",
				},
				Sections: []Section{
					{Title: "Section 1", Assertions: []Assertion{{Code: "A01"}}},
				},
			},
			wantCode: "INVALID_URL",
			wantPath: "report_destination.https.url",
		},
		{
			name: "HTTPS destination with invalid format",
			config: Playbook{
				Title:             "Test",
				ReportDestination: ReportDestinationHTTPS,
				ReportDestinationHTTPS: &ReportDestinationConfig{
					URL:    "https://example.com/api",
					Format: "xml",
				},
				Sections: []Section{
					{Title: "Section 1", Assertions: []Assertion{{Code: "A01"}}},
				},
			},
			wantCode: "INVALID_FORMAT",
			wantPath: "report_destination.https.format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate(false)
			if len(errs) == 0 {
				t.Fatalf("expected validation errors, got none")
			}
			found := false
			for _, e := range errs {
				if e.Code == tt.wantCode && e.Path == tt.wantPath {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error code %q and path %q, got: %v", tt.wantCode, tt.wantPath, errs)
			}
		})
	}
}

func TestPlaybook_Validate_AgentModeProhibitions(t *testing.T) {
	tests := []struct {
		name     string
		config   Playbook
		wantCode string
		wantPath string
		wantSub  string
	}{
		{
			name: "Agent Mode funcFile Error - PreCmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "P01",
								PreCmds: []Exec{
									{FuncFile: "test.ts"},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].preCmds[0].funcFile",
			wantSub:  "contains funcFile in preCmd",
		},
		{
			name: "Agent Mode shellFuncFile Error - PreCmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "SP01",
								PreCmds: []Exec{
									{ShellFuncFile: "test.ts"},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].preCmds[0].shellFuncFile",
			wantSub:  "contains shellFuncFile in preCmd",
		},
		{
			name: "Agent Mode funcFile Error - Cmd Exec",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "C01",
								Cmds: []Cmd{
									{Exec: Exec{FuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].cmds[0].exec.funcFile",
			wantSub:  "contains funcFile in cmd",
		},
		{
			name: "Agent Mode shellFuncFile Error - Cmd Exec",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "SC01",
								Cmds: []Cmd{
									{Exec: Exec{ShellFuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].cmds[0].exec.shellFuncFile",
			wantSub:  "contains shellFuncFile in cmd",
		},
		{
			name: "Agent Mode funcFile Error - StdOutRule",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "R01",
								Cmds: []Cmd{
									{StdOutRule: EvaluationRule{FuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].cmds[0].stdOutRule.funcFile",
			wantSub:  "contains funcFile in stdOutRule",
		},
		{
			name: "Agent Mode funcFile Error - StdErrRule",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "R02",
								Cmds: []Cmd{
									{StdErrRule: EvaluationRule{FuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].cmds[0].stdErrRule.funcFile",
			wantSub:  "contains funcFile in stdErrRule",
		},
		{
			name: "Agent Mode funcFile Error - Gather",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "G01",
								Cmds: []Cmd{
									{
										Exec: Exec{
											Gather: []GatherSpec{
												{FuncFile: "test.ts"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].cmds[0].exec.gather[0].funcFile",
			wantSub:  "contains funcFile in gather",
		},
		{
			name: "Agent Mode funcFile Error - PostCmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "P02",
								PostCmds: []Exec{
									{FuncFile: "test.ts"},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].postCmds[0].funcFile",
			wantSub:  "contains funcFile in postCmd",
		},
		{
			name: "Agent Mode shellFuncFile Error - PostCmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "SP02",
								PostCmds: []Exec{
									{ShellFuncFile: "test.ts"},
								},
							},
						},
					},
				},
			},
			wantCode: "PROHIBITED_AGENT_FUNC",
			wantPath: "sections[0].assertions[0].postCmds[0].shellFuncFile",
			wantSub:  "contains shellFuncFile in postCmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate(true)
			if len(errs) == 0 {
				t.Fatalf("expected validation error, got none")
			}
			found := false
			for _, e := range errs {
				if e.Code == tt.wantCode && e.Path == tt.wantPath && strings.Contains(e.Message, tt.wantSub) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error code %q, path %q, sub %q, got: %v", tt.wantCode, tt.wantPath, tt.wantSub, errs)
			}
		})
	}
}

func TestPlaybook_Validate_MultiError(t *testing.T) {
	// A playbook with multiple simultaneous validation errors across sections and assertions
	pb := Playbook{
		Title: "AB", // INVALID_TITLE
		Sections: []Section{
			{
				Title: "S1", // INVALID_TITLE
				Assertions: []Assertion{
					{Code: ""}, // MISSING_CODE
				},
			},
			{
				Title:      "Valid Section 2",
				Assertions: []Assertion{}, // EMPTY_ASSERTIONS
			},
		},
	}

	errs := pb.Validate(false)
	if len(errs) != 4 {
		t.Fatalf("expected 4 errors, got %d: %v", len(errs), errs)
	}

	expectedCodes := []string{"INVALID_TITLE", "INVALID_TITLE", "MISSING_CODE", "EMPTY_ASSERTIONS"}
	for i, code := range expectedCodes {
		if errs[i].Code != code {
			t.Errorf("error[%d] expected code %q, got %q", i, code, errs[i].Code)
		}
	}
}

func TestValidationError_Formatting(t *testing.T) {
	e1 := ValidationError{
		Path:    "sections[0].code",
		Code:    "MISSING_CODE",
		Message: "code is required",
	}
	if e1.Error() != "sections[0].code: code is required" {
		t.Errorf("unexpected error format: %s", e1.Error())
	}

	e2 := ValidationError{
		Code:    "GENERAL_ERROR",
		Message: "general failure",
	}
	if e2.Error() != "general failure" {
		t.Errorf("unexpected error format without path: %s", e2.Error())
	}

	var emptyErrs ValidationErrors
	if emptyErrs.Error() != "" {
		t.Errorf("expected empty string for empty ValidationErrors, got %q", emptyErrs.Error())
	}

	errs := ValidationErrors{e1, e2}
	expected := "sections[0].code: code is required\ngeneral failure"
	if errs.Error() != expected {
		t.Errorf("unexpected joined error string:\ngot:\n%s\nwant:\n%s", errs.Error(), expected)
	}
}
