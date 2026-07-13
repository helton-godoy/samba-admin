// Package freebsd is the privileged, read-only FreeBSD boundary. It accepts a
// fixed command catalog only; user input never influences executable paths or
// command arguments.
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultMaxOutput = 1 << 20

var ErrCommandDenied = errors.New("comando de leitura não permitido")
var ErrOutputLimit = errors.New("saída do comando excede o limite")

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	Run(context.Context, string, ...string) (CommandResult, error)
}

type ExecRunner struct {
	MaxOutput int
}

// Run uses absolute paths, an empty inherited environment and a fixed working
// directory. The allowlist below intentionally excludes all mutating tools.
func (r ExecRunner) Run(ctx context.Context, path string, args ...string) (CommandResult, error) {
	if !allowedReadOnlyInvocation(path, args) {
		return CommandResult{}, ErrCommandDenied
	}
	maxOutput := r.MaxOutput
	if maxOutput <= 0 {
		maxOutput = defaultMaxOutput
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C", "HOME=/"}
	cmd.Dir = "/"
	cmd.Stdin = nil
	var stdout, stderr limitedBuffer
	stdout.limit = maxOutput
	stderr.limit = maxOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if stdout.exceeded || stderr.exceeded {
		return result, ErrOutputLimit
	}
	if err != nil {
		return result, fmt.Errorf("%s: %w", path, err)
	}
	return result, nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	if b.limit <= 0 {
		return len(value), nil
	}
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.exceeded = true
		return len(value), nil
	}
	if len(value) > remaining {
		_, _ = b.Buffer.Write(value[:remaining])
		b.exceeded = true
		return len(value), nil
	}
	return b.Buffer.Write(value)
}

func allowedReadOnlyInvocation(path string, args []string) bool {
	allowed := map[string]func([]string) bool{
		"/bin/hostname":            exactArguments([][]string{{}, {"-f"}}),
		"/bin/df":                  exactArguments([][]string{{"-k"}}),
		"/bin/cat":                 allowedFile,
		"/bin/date":                exactArguments([][]string{{"+%Z"}}),
		"/sbin/mount":              exactArguments([][]string{{"-p"}}),
		"/sbin/ifconfig":           exactArguments([][]string{{"-l"}}),
		"/sbin/sysctl":             allowedSysctl,
		"/usr/bin/uptime":          exactArguments([][]string{{}}),
		"/usr/local/sbin/pkg":      allowedPkg,
		"/usr/local/sbin/smbd":     exactArguments([][]string{{"-V"}}),
		"/usr/local/bin/testparm":  exactArguments([][]string{{"-s"}}),
		"/usr/local/bin/smbstatus": exactArguments([][]string{{"--shares"}, {"--locks"}, {"--processes"}}),
		"/usr/local/bin/wbinfo":    exactArguments([][]string{{"--ping-dc"}, {"--online-status"}, {"--domain-name"}, {"--own-domain"}}),
		"/usr/local/bin/lpstat":    exactArguments([][]string{{"-p"}, {"-v"}, {"-o"}}),
		"/usr/local/sbin/cupsd":    exactArguments([][]string{{"-t"}}),
		"/usr/bin/ntpq":            exactArguments([][]string{{"-pn"}}),
		"/usr/bin/quota":           exactArguments([][]string{{"-v"}}),
		"/usr/sbin/service":        allowedServiceStatus,
	}
	validate, ok := allowed[path]
	return ok && validate(args)
}

func exactArguments(allowed [][]string) func([]string) bool {
	return func(actual []string) bool {
		for _, expected := range allowed {
			if len(actual) != len(expected) {
				continue
			}
			match := true
			for i := range actual {
				if actual[i] != expected[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
		return false
	}
}

func allowedFile(args []string) bool {
	if len(args) != 1 {
		return false
	}
	switch args[0] {
	case "/etc/resolv.conf", "/etc/fstab", "/usr/local/etc/smb4.conf", "/etc/rc.conf":
		return true
	default:
		return false
	}
}

func allowedSysctl(args []string) bool {
	if len(args) != 2 || args[0] != "-n" {
		return false
	}
	switch args[1] {
	case "kern.osrelease", "hw.machine", "hw.ncpu", "hw.physmem", "vm.loadavg", "kern.boottime", "vm.stats.vm.v_page_count", "vm.stats.vm.v_free_count", "hw.pagesize":
		return true
	default:
		return false
	}
}

func allowedPkg(args []string) bool {
	if len(args) == 3 && args[0] == "query" && args[1] == "%n-%v" {
		return args[2] == "samba423" || args[2] == "cups"
	}
	if len(args) == 3 && args[0] == "info" && args[1] == "-q" {
		return args[2] == "samba423" || args[2] == "cups"
	}
	if len(args) == 2 && args[0] == "info" {
		return args[1] == "samba423" || args[1] == "cups"
	}
	return false
}

func allowedServiceStatus(args []string) bool {
	if len(args) != 2 || args[1] != "status" {
		return false
	}
	return args[0] == "samba_server" || args[0] == "cupsd" || args[0] == "winbindd"
}

func runWithTimeout(ctx context.Context, runner Runner, timeout time.Duration, path string, args ...string) (CommandResult, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return runner.Run(commandContext, path, args...)
}

func commandOutput(ctx context.Context, runner Runner, timeout time.Duration, path string, args ...string) (string, error) {
	result, err := runWithTimeout(ctx, runner, timeout, path, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}
