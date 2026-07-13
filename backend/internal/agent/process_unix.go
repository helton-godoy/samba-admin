//go:build linux || freebsd

package agent

import (
	"errors"
	"os/exec"
	"syscall"
)

// configureCommandTermination puts each fixed utility in its own process
// group so context cancellation terminates descendants as well as the direct
// child. The agent only supports Unix hosts, but keeping this isolated avoids
// platform-specific code in the executor policy.
func configureCommandTermination(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}
}
