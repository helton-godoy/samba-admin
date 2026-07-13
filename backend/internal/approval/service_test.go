package approval

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

func TestHighRiskChangeRequiresSeparateApproverAndWindow(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "approval.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err = store.InsertUser(ctx, "requester", "requester", "Requester", "unused"); err != nil {
		t.Fatal(err)
	}
	if err = store.InsertUser(ctx, "approver", "approver", "Approver", "unused"); err != nil {
		t.Fatal(err)
	}
	service := New(store)
	start := time.Now().UTC().Add(time.Hour)
	end := start.Add(time.Hour)
	request, err := service.Request(ctx, models.User{ID: "requester", Username: "requester"}, RequestInput{
		Operation:        "samba.service.restart",
		Resource:         "service:samba_server",
		Justification:    "Atualização de configuração homologada para janela programada.",
		MaintenanceStart: &start,
		MaintenanceEnd:   &end,
		Impact:           "Sessões SMB ativas podem ser interrompidas durante o restart.",
		RollbackPlan:     "Restaurar configuração versionada e reiniciar novamente em janela aprovada.",
		Payload:          map[string]any{"password": "never-store-this", "nested": map[string]any{"joinCredential": "also-never-store"}, "serviceId": "svc-smbd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != "pending_approval" || request.Risk != string(RiskHigh) {
		t.Fatalf("solicitação de alto risco inesperada: %+v", request)
	}
	if _, err = service.Approve(ctx, request.ID, models.User{ID: "requester", Username: "requester"}, "aprovação própria"); err == nil {
		t.Fatal("solicitante não pode aprovar a própria mudança")
	}
	approved, err := service.Approve(ctx, request.ID, models.User{ID: "approver", Username: "approver"}, "Impacto e rollback revisados.")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "approved" || approved.ApprovedBy != "approver" {
		t.Fatalf("aprovação inesperada: %+v", approved)
	}
	record, err := store.GetChangeRequest(ctx, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(record.PayloadJSON, "never-store-this") || strings.Contains(record.PayloadJSON, "also-never-store") || !strings.Contains(record.PayloadJSON, "[REDACTED]") {
		t.Fatalf("segredo não foi mascarado no payload: %s", record.PayloadJSON)
	}
}

func TestApprovedChangeCanBeClaimedOnlyOnce(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "claim.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store)
	ctx := context.Background()
	if err = store.InsertUser(ctx, "requester", "requester", "Requester", "unused"); err != nil {
		t.Fatal(err)
	}
	if err = store.InsertUser(ctx, "approver", "approver", "Approver", "unused"); err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(-time.Minute)
	end := time.Now().UTC().Add(time.Hour)
	request, err := service.Request(ctx, models.User{ID: "requester"}, RequestInput{
		Operation: "samba.service.restart", Resource: "service:samba_server",
		Justification: "Manutenção operacional previamente autorizada.", MaintenanceStart: &start, MaintenanceEnd: &end,
		Impact: "Sessões SMB podem ser interrompidas temporariamente.", RollbackPlan: "Restaurar configuração anterior e validar o serviço.", Payload: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Approve(ctx, request.ID, models.User{ID: "approver"}, "Impacto e rollback revisados."); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var group sync.WaitGroup
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			if claimErr := service.MarkExecuting(ctx, request.ID, fmt.Sprintf("JOB-%d", index)); claimErr == nil {
				successes.Add(1)
			}
		}(index)
	}
	group.Wait()
	if successes.Load() != 1 {
		t.Fatalf("mudança foi reservada %d vezes", successes.Load())
	}
}

func TestUnknownOperationFailsClosedAsHighRisk(t *testing.T) {
	policy := PolicyFor("future.unknown.operation")
	if policy.Risk != RiskHigh || !policy.RequiresApproval || !policy.RequiresSeparation || !policy.RequiresWindow {
		t.Fatalf("operação desconhecida não falhou fechada: %+v", policy)
	}
}
