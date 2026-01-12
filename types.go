package main

type Assertion struct {
	Code            string   `yaml:"code" json:"code" jsonschema:"description=Unique code for the assertion,minLength=3"`
	Title           string   `yaml:"title" json:"title" jsonschema:"description=Title of the assertion,minLength=3"`
	Description     string   `yaml:"description" json:"description" jsonschema:"description=Detailed description of what is being checked,minLength=3"`
	Cmd             []string `yaml:"cmd" json:"cmd" jsonschema:"description=List of commands to execute for the assertion,minItems=1"`
	PreCmd          []string `yaml:"preCmd,omitempty" json:"preCmd,omitempty" jsonschema:"description=Commands to run before the main check"`
	PostCmd         []string `yaml:"postCmd,omitempty" json:"postCmd,omitempty" jsonschema:"description=Commands to run after the main check"`
	PassDescription string   `yaml:"passDescription" json:"passDescription" jsonschema:"description=Message shown if all commands in 'cmd' exit with 0.,minLength=3"`
	FailDescription string   `yaml:"failDescription" json:"failDescription" jsonschema:"description=Message shown if any command in 'cmd' exits with non-zero.,minLength=3"`
}

type Section struct {
	Title       string      `yaml:"title" json:"title" jsonschema:"description=Title of the section,minLength=3"`
	Description []string    `yaml:"description" json:"description" jsonschema:"description=List of descriptions for the section,minItems=1"`
	Assertions  []Assertion `yaml:"assertions" json:"assertions" jsonschema:"description=List of assertions within this section,minItems=1"`
}

type ReportConfig struct {
	Title    string    `yaml:"title" json:"title" jsonschema:"description=Title of the report,minLength=3"`
	Sections []Section `yaml:"sections" json:"sections" jsonschema:"description=List of sections in the report,minItems=1"`
}
