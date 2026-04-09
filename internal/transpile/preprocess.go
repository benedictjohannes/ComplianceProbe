package transpile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/playbook"
	"gopkg.in/yaml.v3"
)

// BakeFile loads a raw playbook from inputPath, transpiles all external scripts,
// validates the result, and saves it to outputPath.
func BakeFile(inputPath, outputPath string) error {
	if strings.HasPrefix(inputPath, "https://") {
		return fmt.Errorf("baking remote playbooks is not supported as relative paths to external script files would break")
	}

	config, _, err := configsource.LoadConfig(inputPath, nil)
	if err != nil {
		return fmt.Errorf("failed to load input: %w", err)
	}

	if err := Preprocess(config, filepath.Dir(inputPath)); err != nil {
		return err
	}

	if err := playbook.ValidateConfig(*config, false); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	outData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	err = os.WriteFile(outputPath, outData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// Preprocess walks through the playbook configuration and transpiles any external script files
// (funcFile, shellFuncFile) into inlined JavaScript code.
func Preprocess(config *playbook.Playbook, baseDir string) error {
	for i := range config.Sections {
		for j := range config.Sections[i].Assertions {
			if err := processAssertion(&config.Sections[i].Assertions[j], baseDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func processAssertion(a *playbook.Assertion, baseDir string) error {
	for i := range a.PreCmds {
		if err := processExec(&a.PreCmds[i], baseDir); err != nil {
			return err
		}
	}
	for i := range a.Cmds {
		if err := processExec(&a.Cmds[i].Exec, baseDir); err != nil {
			return err
		}
		if err := processEvalRule(&a.Cmds[i].StdOutRule, baseDir); err != nil {
			return err
		}
		if err := processEvalRule(&a.Cmds[i].StdErrRule, baseDir); err != nil {
			return err
		}
	}
	for i := range a.PostCmds {
		if err := processExec(&a.PostCmds[i], baseDir); err != nil {
			return err
		}
	}
	return nil
}

func processExec(e *playbook.Exec, baseDir string) error {
	if e.ShellFuncFile != "" {
		code, err := Transpile(filepath.Join(baseDir, e.ShellFuncFile))
		if err != nil {
			return fmt.Errorf("transpilation error for shellFuncFile (%s): %v", e.ShellFuncFile, err)
		}
		e.ShellFunc = code
		e.ShellFuncFile = ""
	}
	if e.FuncFile != "" {
		code, err := Transpile(filepath.Join(baseDir, e.FuncFile))
		if err != nil {
			return fmt.Errorf("transpilation error for funcFile (%s): %v", e.FuncFile, err)
		}
		e.Func = code
		e.FuncFile = ""
	}
	for i := range e.Gather {
		if e.Gather[i].FuncFile != "" {
			code, err := Transpile(filepath.Join(baseDir, e.Gather[i].FuncFile))
			if err != nil {
				return fmt.Errorf("transpilation error for gather funcFile (%s): %v", e.Gather[i].FuncFile, err)
			}
			e.Gather[i].Func = code
			e.Gather[i].FuncFile = ""
		}
	}
	return nil
}

func processEvalRule(r *playbook.EvaluationRule, baseDir string) error {
	if r.FuncFile != "" {
		code, err := Transpile(filepath.Join(baseDir, r.FuncFile))
		if err != nil {
			return fmt.Errorf("transpilation error for evaluationRule funcFile (%s): %v", r.FuncFile, err)
		}
		r.Func = code
		r.FuncFile = ""
	}
	return nil
}
