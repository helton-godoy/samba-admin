package agent

import (
	"context"
	"fmt"

	fixtureadapter "github.com/hu-ufcat/samba-admin-backend/internal/adapters/fixture"
)

// FixtureExecutor reproduz respostas estruturadas de uma coleta sanitizada
// através do mesmo servidor UDS e das mesmas validações usadas pelo agente.
// Ele existe para testes de integração e nunca executa comandos no host.
type FixtureExecutor struct {
	provider fixtureadapter.Provider
}

func NewFixtureExecutor(path string) (FixtureExecutor, error) {
	provider, err := fixtureadapter.Open(path)
	if err != nil {
		return FixtureExecutor{}, fmt.Errorf("abrir fixture do executor: %w", err)
	}
	return FixtureExecutor{provider: provider}, nil
}

func (e FixtureExecutor) Execute(ctx context.Context, request Request, spec OperationSpec) (Response, error) {
	response := Response{
		RequestID: request.RequestID,
		Success:   true,
		ExitCode:  0,
		Summary:   "Inventário sanitizado reproduzido pelo agente",
		Data:      map[string]any{},
	}

	var err error
	switch spec.Operation {
	case "system.inspect":
		response.Data["system"], err = e.provider.Inspect(ctx)
	case "capabilities.inspect":
		response.Data["capabilities"], err = e.provider.Capabilities(ctx)
	case "filesystem.inspect":
		response.Data["filesystems"], err = e.provider.List(ctx)
	case "samba.inspect":
		response.Data["samba"], err = e.provider.Samba(ctx)
	case "domain.member.diagnose":
		response.Data["domain"], err = e.provider.Domain(ctx)
	case "cups.inspect":
		response.Data["cups"], err = e.provider.Cups(ctx)
	case "service.status":
		response.Data["service"] = map[string]any{"checked": true}
	default:
		return Response{
			RequestID: request.RequestID,
			Success:   false,
			ExitCode:  1,
			Summary:   "Operação ausente no executor de fixture",
			ErrorCode: "fixture_operation_unavailable",
		}, fmt.Errorf("operação %s não é reproduzida pela fixture", spec.Operation)
	}
	if err != nil {
		return Response{
			RequestID: request.RequestID,
			Success:   false,
			ExitCode:  1,
			Summary:   "Fixture não pôde reproduzir o inventário solicitado",
			ErrorCode: "fixture_replay_failed",
		}, err
	}
	return response, nil
}

var _ Executor = FixtureExecutor{}
