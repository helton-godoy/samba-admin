//go:build !linux && !freebsd

package agent

import "os/exec"

func configureCommandTermination(command *exec.Cmd) {}
