package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/agent"
	"github.com/hu-ufcat/samba-admin-backend/internal/audit"
	"github.com/hu-ufcat/samba-admin-backend/internal/events"
	"github.com/hu-ufcat/samba-admin-backend/internal/id"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

type Manager struct {
	store  *storage.Store
	broker *events.Broker
	agent  agent.Client
	audit  *audit.Service
	mu     sync.Mutex
	cancel map[string]context.CancelFunc
}

func New(store *storage.Store, broker *events.Broker, client agent.Client, a *audit.Service) *Manager {
	return &Manager{store: store, broker: broker, agent: client, audit: a, cancel: map[string]context.CancelFunc{}}
}
func (m *Manager) Create(ctx context.Context, actor, operation, resource, correlation, agentOperation string, payload map[string]any, rollback bool) (models.Job, error) {
	return m.CreateWithID(ctx, id.New("JOB-"), actor, operation, resource, correlation, agentOperation, payload, rollback)
}

func (m *Manager) CreateWithID(ctx context.Context, jobID, actor, operation, resource, correlation, agentOperation string, payload map[string]any, rollback bool) (models.Job, error) {
	if strings.TrimSpace(jobID) == "" || len(jobID) > 128 || strings.ContainsAny(jobID, "\x00\r\n") {
		return models.Job{}, fmt.Errorf("ID de tarefa inválido")
	}
	j := models.Job{ID: jobID, RequestedBy: actor, RequestedAt: time.Now().UTC(), Operation: operation, Resource: resource, Status: "queued", Progress: 0, Summary: "Tarefa criada e aguardando validação.", Cancellable: true, RollbackAvailable: rollback, CorrelationID: correlation}
	lockKey := resource
	if err := m.store.InsertJob(ctx, j, payload, lockKey); err != nil {
		return j, err
	}
	_ = m.store.PutEvent(ctx, "job.created", j.ID, j, correlation)
	m.broker.Publish(events.Event{Type: "job.created", ResourceID: j.ID, CorrelationID: correlation, Payload: j})
	go m.run(j, agentOperation, payload, lockKey)
	return j, nil
}
func (m *Manager) run(j models.Job, op string, payload map[string]any, lockKey string) {
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancel[j.ID] = cancel
	m.mu.Unlock()
	defer func() { m.mu.Lock(); delete(m.cancel, j.ID); m.mu.Unlock() }()
	if err := m.store.AcquireLock(ctx, lockKey, j.ID, 10*time.Minute); err != nil {
		m.fail(ctx, &j, "resource_conflict", "Outro trabalho mantém lock sobre o recurso.")
		return
	}
	defer m.store.ReleaseLock(ctx, lockKey, j.ID)
	steps := []struct {
		status        string
		progress      int
		step, summary string
		wait          time.Duration
	}{{"validating", 10, "Validação local", "Validando política, schema e pré-condições.", 120 * time.Millisecond}, {"running", 25, "Backup", "Backup lógico registrado antes da alteração.", 120 * time.Millisecond}, {"running", 45, "Aplicação", "Enviando intenção estruturada ao agente local.", 100 * time.Millisecond}}
	for _, s := range steps {
		if !m.advance(ctx, &j, s.status, s.progress, s.step, s.summary, s.wait) {
			return
		}
	}
	res, err := m.agent.Execute(ctx, op, payload)
	if err != nil {
		m.fail(ctx, &j, "agent_failure", res.Summary)
		return
	}
	if !m.advance(ctx, &j, "testing", 75, "Testes técnicos", "Executando validação pós-aplicação e health check.", 180*time.Millisecond) {
		return
	}
	if !m.advance(ctx, &j, "running", 90, "Auditoria", "Registrando resultado e cadeia de auditoria.", 100*time.Millisecond) {
		return
	}
	j.Status = "success"
	j.Progress = 100
	j.CurrentStep = "Concluída"
	j.Summary = "Alteração simulada aplicada, testada e auditada."
	j.Cancellable = false
	_ = m.store.UpdateJob(ctx, j)
	_ = m.audit.Record(ctx, audit.Entry{Actor: j.RequestedBy, Roles: []string{"administrador Samba"}, Source: "api", Operation: j.Operation, Resource: j.Resource, Result: "success", CorrelationID: j.CorrelationID, JobID: j.ID, After: payload})
	m.publish(ctx, "job.completed", j)
}
func (m *Manager) advance(ctx context.Context, j *models.Job, status string, progress int, step, summary string, wait time.Duration) bool {
	select {
	case <-ctx.Done():
		j.Status = "cancelled"
		j.Summary = "Tarefa cancelada antes da conclusão."
		j.Cancellable = false
		_ = m.store.UpdateJob(context.Background(), *j)
		m.publish(context.Background(), "job.cancelled", *j)
		return false
	case <-time.After(wait):
	}
	j.Status = status
	j.Progress = progress
	j.CurrentStep = step
	j.Summary = summary
	_ = m.store.UpdateJob(ctx, *j)
	m.publish(ctx, "job.updated", *j)
	return true
}
func (m *Manager) fail(ctx context.Context, j *models.Job, code, detail string) {
	j.Status = "failed"
	j.Cancellable = false
	j.Error = code
	if detail == "" {
		detail = "A operação simulada falhou."
	}
	j.Summary = detail
	_ = m.store.UpdateJob(ctx, *j)
	_ = m.audit.Record(ctx, audit.Entry{Actor: j.RequestedBy, Operation: j.Operation, Resource: j.Resource, Result: "failure", CorrelationID: j.CorrelationID, JobID: j.ID, Reason: detail})
	m.publish(ctx, "job.failed", *j)
}
func (m *Manager) publish(ctx context.Context, t string, j models.Job) {
	_ = m.store.PutEvent(ctx, t, j.ID, j, j.CorrelationID)
	m.broker.Publish(events.Event{Type: t, ResourceID: j.ID, CorrelationID: j.CorrelationID, Payload: j})
}
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.cancel[id]
	if !ok {
		return fmt.Errorf("tarefa não está em execução")
	}
	c()
	return nil
}
func (m *Manager) Rollback(ctx context.Context, id, actor, correlation string) (models.Job, error) {
	orig, err := m.store.GetJob(ctx, id)
	if err != nil {
		return models.Job{}, err
	}
	if !orig.RollbackAvailable {
		return models.Job{}, fmt.Errorf("rollback indisponível")
	}
	j, err := m.Create(ctx, actor, "Rollback de "+orig.Operation, orig.Resource, correlation, "rollback.execute", map[string]any{"sourceJobId": id}, false)
	if err == nil {
		j.Status = "queued"
	}
	return j, err
}
