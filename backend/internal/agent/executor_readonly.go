package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters/freebsd"
)

var (
	ErrMutationBlocked = errors.New("operações mutáveis permanecem bloqueadas até homologação")
	ErrNotImplemented  = errors.New("operação somente leitura ainda não implementada")
	ErrOutputLimit     = errors.New("limite de saída excedido")
)

type commandSpec struct {
	path string
	args []string
}

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

type commandRunner interface {
	Run(context.Context, commandSpec, int) (commandResult, error)
}

// ReadOnlyExecutor is the only real executor available in this increment. It
// uses fixed, absolute command vectors and never interpolates a request value
// into a shell command. Mutable catalog entries are rejected before execution.
type ReadOnlyExecutor struct {
	runner      commandRunner
	outputLimit int
}

func NewReadOnlyExecutor(outputLimit int) *ReadOnlyExecutor {
	if outputLimit <= 0 {
		outputLimit = 128 << 10
	}
	return &ReadOnlyExecutor{runner: execCommandRunner{}, outputLimit: outputLimit}
}

func newReadOnlyExecutorForTest(runner commandRunner, outputLimit int) *ReadOnlyExecutor {
	if outputLimit <= 0 {
		outputLimit = 128 << 10
	}
	return &ReadOnlyExecutor{runner: runner, outputLimit: outputLimit}
}

func (e *ReadOnlyExecutor) Execute(ctx context.Context, request Request, spec OperationSpec) (Response, error) {
	if spec.IsMutation() {
		return Response{RequestID: request.RequestID, Success: false, Summary: "Operação mutável bloqueada até homologação.", ErrorCode: "mutation_blocked"}, ErrMutationBlocked
	}
	if inventoryField(request.Operation) != "" {
		return e.executeInventory(ctx, request, spec)
	}
	command, err := readOnlyCommand(request.Operation, request.Payload)
	if err != nil {
		return Response{RequestID: request.RequestID, Success: false, Summary: "Operação somente leitura ainda não está disponível.", ErrorCode: "capability_unavailable"}, err
	}
	if !commandAllowed(spec, command) {
		return Response{RequestID: request.RequestID, Success: false, Summary: "Política de executável recusou a operação.", ErrorCode: "policy_denied"}, errors.New("comando fora do catálogo")
	}
	result, err := e.runner.Run(ctx, command, e.outputLimit)
	response := Response{
		RequestID: request.RequestID,
		Success:   err == nil,
		ExitCode:  result.exitCode,
		Summary:   "Coleta somente leitura concluída.",
		Stdout:    redactOutput(result.stdout),
		Stderr:    redactOutput(result.stderr),
	}
	if err != nil {
		response.Success = false
		response.ErrorCode = commandErrorCode(ctx, err)
		response.Summary = "Coleta somente leitura falhou de forma controlada."
	}
	return response, err
}

func (e *ReadOnlyExecutor) executeInventory(ctx context.Context, request Request, spec OperationSpec) (Response, error) {
	runner := catalogRunner{runner: e.runner, spec: spec, outputLimit: e.outputLimit}
	provider := freebsd.New(runner)
	var value any
	var err error
	switch request.Operation {
	case "system.inspect":
		value, err = provider.Inspect(ctx)
	case "capabilities.inspect":
		value, err = provider.Capabilities(ctx)
	case "filesystem.inspect":
		value, err = provider.List(ctx)
	case "samba.inspect":
		value, err = provider.Samba(ctx)
	case "domain.member.diagnose":
		value, err = provider.Domain(ctx)
	case "cups.inspect":
		value, err = provider.Cups(ctx)
	default:
		return Response{RequestID: request.RequestID, Success: false, Summary: "Inventário somente leitura indisponível.", ErrorCode: "capability_unavailable"}, ErrNotImplemented
	}
	if err != nil {
		return Response{RequestID: request.RequestID, Success: false, Summary: "Coleta somente leitura falhou de forma controlada.", ErrorCode: commandErrorCode(ctx, err)}, err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return Response{RequestID: request.RequestID, Success: false, Summary: "Falha ao estruturar inventário.", ErrorCode: "encoding_failed"}, err
	}
	var normalized any
	if err = json.Unmarshal(encoded, &normalized); err != nil {
		return Response{RequestID: request.RequestID, Success: false, Summary: "Falha ao estruturar inventário.", ErrorCode: "encoding_failed"}, err
	}
	return Response{
		RequestID: request.RequestID,
		Success:   true,
		Summary:   "Inventário somente leitura concluído.",
		Data:      map[string]any{inventoryField(request.Operation): normalized},
	}, nil
}

func inventoryField(operation string) string {
	switch operation {
	case "system.inspect":
		return "system"
	case "capabilities.inspect":
		return "capabilities"
	case "filesystem.inspect":
		return "filesystems"
	case "samba.inspect":
		return "samba"
	case "domain.member.diagnose":
		return "domain"
	case "cups.inspect":
		return "cups"
	default:
		return ""
	}
}

type catalogRunner struct {
	runner      commandRunner
	spec        OperationSpec
	outputLimit int
}

func (r catalogRunner) Run(ctx context.Context, path string, args ...string) (freebsd.CommandResult, error) {
	command := commandSpec{path: path, args: append([]string(nil), args...)}
	if !commandAllowed(r.spec, command) {
		return freebsd.CommandResult{}, freebsd.ErrCommandDenied
	}
	result, err := r.runner.Run(ctx, command, r.outputLimit)
	return freebsd.CommandResult{Stdout: result.stdout, Stderr: result.stderr, ExitCode: result.exitCode}, err
}

func commandAllowed(spec OperationSpec, command commandSpec) bool {
	if !contains(spec.Executables, command.path) {
		return false
	}
	allowed, exists := spec.AllowedArguments[command.path]
	if !exists {
		return len(command.args) == 0
	}
	for _, argument := range command.args {
		if !contains(allowed, argument) {
			return false
		}
	}
	return true
}

func readOnlyCommand(operation string, payload map[string]any) (commandSpec, error) {
	switch operation {
	case "system.inspect":
		return commandSpec{path: "/usr/bin/uname", args: []string{"-a"}}, nil
	case "filesystem.inspect":
		return commandSpec{path: "/sbin/mount", args: []string{"-p"}}, nil
	case "samba.inspect":
		return commandSpec{path: "/usr/local/sbin/smbd", args: []string{"-V"}}, nil
	case "samba.config.validate":
		return commandSpec{path: "/usr/local/bin/testparm", args: []string{"-s"}}, nil
	case "domain.member.diagnose":
		return commandSpec{path: "/usr/local/bin/wbinfo", args: []string{"--ping-dc"}}, nil
	case "cups.inspect":
		return commandSpec{path: "/usr/local/bin/lpstat", args: []string{"-t"}}, nil
	case "quota.read":
		return commandSpec{path: "/usr/bin/quota", args: []string{"-v"}}, nil
	case "service.status":
		serviceID, _ := payload["serviceId"].(string)
		serviceName, ok := map[string]string{
			"svc-smbd":     "samba_server",
			"svc-winbindd": "winbindd",
			"svc-cupsd":    "cupsd",
		}[serviceID]
		if !ok {
			return commandSpec{}, errors.New("serviço fora da allowlist")
		}
		return commandSpec{path: "/usr/sbin/service", args: []string{serviceName, "status"}}, nil
	default:
		return commandSpec{}, ErrNotImplemented
	}
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, command commandSpec, outputLimit int) (commandResult, error) {
	if command.path == "" || !strings.HasPrefix(command.path, "/") {
		return commandResult{}, errors.New("caminho de executável inválido")
	}
	info, err := os.Lstat(command.path)
	if err != nil {
		return commandResult{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return commandResult{}, errors.New("executável deve ser arquivo regular e não pode ser symlink")
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stdout := newBoundedBuffer(outputLimit, cancel)
	stderr := newBoundedBuffer(outputLimit, cancel)
	cmd := exec.CommandContext(runCtx, command.path, command.args...)
	cmd.Args = append([]string{command.path}, command.args...)
	cmd.Dir = "/"
	cmd.Env = []string{
		"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/var/empty",
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
	}
	configureCommandTermination(cmd)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	result := commandResult{stdout: stdout.String(), stderr: stderr.String()}
	if cmd.ProcessState != nil {
		result.exitCode = cmd.ProcessState.ExitCode()
	}
	if stdout.Exceeded() || stderr.Exceeded() {
		return result, ErrOutputLimit
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

type boundedBuffer struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	limit    int
	exceeded bool
	cancel   context.CancelFunc
}

func newBoundedBuffer(limit int, cancel context.CancelFunc) *boundedBuffer {
	return &boundedBuffer{limit: limit, cancel: cancel}
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			_, _ = b.buffer.Write(data[:remaining])
		} else {
			_, _ = b.buffer.Write(data)
		}
	}
	if len(data) > remaining {
		b.exceeded = true
		b.cancel()
		return len(data), ErrOutputLimit
	}
	return len(data), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func (b *boundedBuffer) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exceeded
}

func commandErrorCode(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, ErrOutputLimit):
		return "output_limit"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "timeout"
	case errors.Is(ctx.Err(), context.Canceled):
		return "cancelled"
	default:
		return "execution_failed"
	}
}

var sensitiveOutput = regexp.MustCompile(`(?im)^([^\r\n]*(?:password|passwd|secret|token|credential|authorization|cookie|keytab|private[_ -]?key|totp|otp|recovery[_ -]?code)[^\r\n]*[=:]).*$`)

func redactOutput(value string) string {
	return sensitiveOutput.ReplaceAllString(value, "$1[REDACTED]")
}

func (e *ReadOnlyExecutor) String() string {
	return fmt.Sprintf("readonly-executor(output-limit=%d)", e.outputLimit)
}
