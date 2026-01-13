package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type ExecFunc func(e *Exec, context map[string]interface{}) (ExecutionResult, error)

type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Success  bool
}

func runExec(e *Exec, context map[string]interface{}) (ExecutionResult, error) {
	script := e.Script

	// If Func is provided, it wins and generates the script
	if e.Func != "" {
		jsScript, err := runJS(e.Func, context)
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("JS error in Exec.Func: %v", err)
		}
		if jsScript == "" {
			return ExecutionResult{Success: true}, nil // Skip execution if JS returns empty
		}
		script = jsScript
	}
	e.Script = script

	if script == "" {
		return ExecutionResult{Success: true}, nil
	}

	res := runShell(script, e.Shell)

	// Handle Gathering
	for _, g := range e.Gather {
		val, err := performGather(g, res, context)
		if err != nil {
			return res, fmt.Errorf("gather error for key %s: %v", g.Key, err)
		}
		context[g.Key] = val
	}

	return res, nil
}

func runShell(command string, shell string) ExecutionResult {
	var name string
	var args []string

	tmpDir := os.TempDir()
	var tmpFile string

	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "powershell"
		} else {
			shell = "bash"
		}
	}

	switch shell {
	case "powershell":
		name = "powershell"
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d.ps1", time.Now().UnixNano()))
		os.WriteFile(tmpFile, []byte(command), 0644)
		args = []string{"-ExecutionPolicy", "Bypass", "-File", tmpFile}
	case "bash", "sh":
		name = shell
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d.sh", time.Now().UnixNano()))
		script := fmt.Sprintf("#!/bin/%s\nset -o pipefail\n%s\n", shell, command)
		os.WriteFile(tmpFile, []byte(script), 0755)
		args = []string{tmpFile}
	default:
		// Try to run it directly if it's just a command name in PATH
		name = shell
		args = []string{"-c", command}
	}

	defer os.Remove(tmpFile)

	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1", "LANG=en_US.UTF-8")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return ExecutionResult{
		Stdout:   cleanupOutput(stdout.String()),
		Stderr:   cleanupOutput(stderr.String()),
		ExitCode: exitCode,
		Success:  err == nil,
	}
}

func runJS(code string, context map[string]interface{}) (string, error) {
	vm := goja.New()

	// Inject Context
	vm.Set("assertionContext", context)

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}
	vm.Set("os", osName)
	vm.Set("arch", runtime.GOARCH)

	// Inject Env
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}
	vm.Set("env", envMap)

	// Inject User/CWD
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	vm.Set("user", user)
	cwd, _ := os.Getwd()
	vm.Set("cwd", cwd)

	// Run code
	val, err := vm.RunString(code)
	if err != nil {
		return "", err
	}

	// Signature: ({ assertionContext, env, os, arch, user, cwd }) => string
	if fn, ok := goja.AssertFunction(val); ok {
		params := vm.NewObject()
		params.Set("assertionContext", context)
		params.Set("env", envMap)
		params.Set("os", osName)
		params.Set("arch", runtime.GOARCH)
		params.Set("user", user)
		params.Set("cwd", cwd)

		res, err := fn(goja.Undefined(), params)
		if err != nil {
			return "", err
		}
		return res.String(), nil
	}

	if goja.IsUndefined(val) || goja.IsNull(val) {
		return "", nil
	}

	return val.String(), nil
}

func performGather(g GatherSpec, res ExecutionResult, context map[string]interface{}) (string, error) {
	input := res.Stdout
	if g.GetIncludeStdErr() && input == "" {
		input = res.Stderr
	}

	// JS Function wins
	if g.Func != "" {
		vm := goja.New()
		vm.Set("stdout", res.Stdout)
		vm.Set("stderr", res.Stderr)
		vm.Set("assertionContext", context)

		val, err := vm.RunString(g.Func)
		if err != nil {
			return "", err
		}

		if fn, ok := goja.AssertFunction(val); ok {
			// Signature: (stdout, stderr, assertionContext) => string
			res, err := fn(goja.Undefined(), vm.ToValue(res.Stdout), vm.ToValue(res.Stderr), vm.ToValue(context))
			if err != nil {
				return "", err
			}
			return res.String(), nil
		}

		return val.String(), nil
	}

	// Regex
	if g.Regex != "" {
		re, err := regexp.Compile(g.Regex)
		if err != nil {
			return "", err
		}
		matches := re.FindStringSubmatch(input)
		if len(matches) > 1 {
			return matches[1], nil
		} else if len(matches) == 1 {
			return matches[0], nil
		}
	}

	return "", nil
}

func evaluateRule(rule EvaluationRule, res ExecutionResult, context map[string]interface{}) (int, error) {
	input := res.Stdout
	if rule.GetIncludeStdErr() && input == "" {
		input = res.Stderr
	}

	if rule.Func != "" {
		vm := goja.New()
		vm.Set("stdout", res.Stdout)
		vm.Set("stderr", res.Stderr)
		vm.Set("assertionContext", context)

		val, err := vm.RunString(rule.Func)
		if err != nil {
			return 0, err
		}

		if fn, ok := goja.AssertFunction(val); ok {
			// Signature: (stdout, stderr, assertionContext) => -1 | 0 | 1
			out, err := fn(goja.Undefined(), vm.ToValue(res.Stdout), vm.ToValue(res.Stderr), vm.ToValue(context))
			if err != nil {
				return 0, err
			}
			return int(out.ToInteger()), nil
		}

		return int(val.ToInteger()), nil
	}

	if rule.Regex != "" {
		re, err := regexp.Compile(rule.Regex)
		if err != nil {
			return 0, err
		}
		if re.MatchString(input) {
			return 1, nil
		}
		return -1, nil
	}

	return 0, nil
}
