//go:build windows

package executor

import (
	"os/exec"
)

func prepareCmdCtx(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
