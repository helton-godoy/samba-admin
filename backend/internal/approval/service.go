// Package approval implements the separation-of-duties gate before a queued
// operation can reach the privileged agent. It does not execute operations.
package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/id"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

type Risk string

const (
	RiskRead        Risk = "read"
	RiskLow         Risk = "low"
	RiskMedium      Risk = "medium"
	RiskHigh        Risk = "high"
	RiskDestructive Risk = "destructive"
	RiskEmergency   Risk = "emergency"
)

type Policy struct {
	Risk               Risk
	RequiresApproval   bool
	RequiresWindow     bool
	RequiresSeparation bool
	RequiredCapability string
}

// Policies are intentionally conservative. Unknown operations are high risk
// and require a separate approver instead of being inferred as safe.
var Policies = map[string]Policy{
	"system.inspect":          {Risk: RiskRead, RequiredCapability: "system.inspect"},
	"filesystem.inspect":      {Risk: RiskRead, RequiredCapability: "filesystem.read"},
	"acl.read":                {Risk: RiskRead, RequiredCapability: "acl.nfsv4.read"},
	"cups.inspect":            {Risk: RiskRead, RequiredCapability: "cups.inspect"},
	"domain.member.diagnose":  {Risk: RiskRead, RequiredCapability: "domain.member.diagnose"},
	"samba.config.validate":   {Risk: RiskLow, RequiredCapability: "samba.testparm"},
	"share.create":            {Risk: RiskMedium, RequiredCapability: "share.write"},
	"share.update":            {Risk: RiskMedium, RequiredCapability: "share.write"},
	"share.delete":            {Risk: RiskDestructive, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "share.write"},
	"acl.model.change":        {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "acl.nfsv4.write"},
	"acl.recursive.apply":     {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "acl.nfsv4.write"},
	"filesystem.fstab.apply":  {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "filesystem.fstab.write"},
	"domain.member.join":      {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "domain.member.join"},
	"domain.member.leave":     {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "domain.member.leave"},
	"samba.profile.apply":     {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "samba.profile.write"},
	"samba.idmap.apply":       {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "domain.idmap.write"},
	"samba.service.restart":   {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "samba.service.restart"},
	"print.driver.remove":     {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "cups.driver.write"},
	"logging.syslog.apply":    {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "syslog.tls"},
	"rollback.global.execute": {Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "rollback.global"},
	"samba.ad_dc.promote":     {Risk: RiskDestructive, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true, RequiredCapability: "samba.ad_dc"},
	"rollback.execute":        {Risk: RiskMedium, RequiredCapability: "rollback.execute"},
}

type RequestInput struct {
	Operation        string         `json:"operation"`
	Resource         string         `json:"resource"`
	Justification    string         `json:"justification"`
	MaintenanceStart *time.Time     `json:"maintenanceStart,omitempty"`
	MaintenanceEnd   *time.Time     `json:"maintenanceEnd,omitempty"`
	Impact           string         `json:"impact"`
	RollbackPlan     string         `json:"rollbackPlan"`
	Payload          map[string]any `json:"payload"`
}

type Service struct{ store *storage.Store }

func New(store *storage.Store) *Service { return &Service{store: store} }

func PolicyFor(operation string) Policy {
	if policy, ok := Policies[operation]; ok {
		return policy
	}
	return Policy{Risk: RiskHigh, RequiresApproval: true, RequiresWindow: true, RequiresSeparation: true}
}

func (s *Service) Request(ctx context.Context, requester models.User, in RequestInput) (models.ChangeRequest, error) {
	in.Operation = strings.TrimSpace(in.Operation)
	in.Resource = strings.TrimSpace(in.Resource)
	in.Justification = strings.TrimSpace(in.Justification)
	in.Impact = strings.TrimSpace(in.Impact)
	in.RollbackPlan = strings.TrimSpace(in.RollbackPlan)
	if err := validateText("operação", in.Operation, 3, 128, true); err != nil {
		return models.ChangeRequest{}, err
	}
	if err := validateText("recurso", in.Resource, 3, 255, true); err != nil {
		return models.ChangeRequest{}, err
	}
	for name, value := range map[string]string{"justificativa": in.Justification, "impacto": in.Impact, "plano de rollback": in.RollbackPlan} {
		if err := validateText(name, value, 10, 4000, false); err != nil {
			return models.ChangeRequest{}, err
		}
	}
	if in.Payload == nil {
		return models.ChangeRequest{}, errors.New("payload estruturado da mudança é obrigatório")
	}
	policy := PolicyFor(in.Operation)
	if policy.RequiresWindow {
		if in.MaintenanceStart == nil || in.MaintenanceEnd == nil || !in.MaintenanceEnd.After(*in.MaintenanceStart) || !in.MaintenanceEnd.After(time.Now().UTC()) {
			return models.ChangeRequest{}, errors.New("a operação exige uma janela de manutenção futura e válida")
		}
	}
	payload, err := json.Marshal(redact(in.Payload))
	if err != nil {
		return models.ChangeRequest{}, errors.New("payload da mudança inválido")
	}
	status := "approved"
	var expires *time.Time
	if policy.RequiresApproval {
		status = "pending_approval"
		if in.MaintenanceEnd != nil {
			value := in.MaintenanceEnd.UTC()
			expires = &value
		} else {
			value := time.Now().UTC().Add(7 * 24 * time.Hour)
			expires = &value
		}
	}
	record := storage.ChangeRequestRecord{
		ID:                 id.New("chg-"),
		Operation:          in.Operation,
		Resource:           in.Resource,
		Risk:               string(policy.Risk),
		Status:             status,
		RequestedBy:        requester.ID,
		RequestedAt:        time.Now().UTC(),
		Justification:      in.Justification,
		MaintenanceStart:   in.MaintenanceStart,
		MaintenanceEnd:     in.MaintenanceEnd,
		Impact:             in.Impact,
		RollbackPlan:       in.RollbackPlan,
		PayloadJSON:        string(payload),
		RequiredCapability: policy.RequiredCapability,
		ExpiresAt:          expires,
	}
	if err := s.store.CreateChangeRequest(ctx, record); err != nil {
		return models.ChangeRequest{}, err
	}
	return fromRecord(record), nil
}

func (s *Service) Approve(ctx context.Context, id string, approver models.User, reason string) (models.ChangeRequest, error) {
	request, err := s.store.GetChangeRequest(ctx, id)
	if err != nil {
		return models.ChangeRequest{}, err
	}
	policy := PolicyFor(request.Operation)
	if !policy.RequiresApproval {
		return models.ChangeRequest{}, errors.New("a operação não exige aprovação")
	}
	if policy.RequiresSeparation && request.RequestedBy == approver.ID {
		return models.ChangeRequest{}, errors.New("o solicitante não pode aprovar a própria mudança")
	}
	if err := validateText("justificativa da aprovação", strings.TrimSpace(reason), 5, 4000, false); err != nil {
		return models.ChangeRequest{}, err
	}
	updated, err := s.store.DecideChangeRequest(ctx, id, approver.ID, "approved", strings.TrimSpace(reason))
	if err != nil {
		return models.ChangeRequest{}, err
	}
	return fromRecord(updated), nil
}

func (s *Service) Reject(ctx context.Context, id string, approver models.User, reason string) (models.ChangeRequest, error) {
	if err := validateText("justificativa da rejeição", strings.TrimSpace(reason), 5, 4000, false); err != nil {
		return models.ChangeRequest{}, err
	}
	request, err := s.store.GetChangeRequest(ctx, id)
	if err != nil {
		return models.ChangeRequest{}, err
	}
	if request.RequestedBy == approver.ID {
		return models.ChangeRequest{}, errors.New("o solicitante não pode decidir a própria mudança")
	}
	updated, err := s.store.DecideChangeRequest(ctx, id, approver.ID, "rejected", strings.TrimSpace(reason))
	if err != nil {
		return models.ChangeRequest{}, err
	}
	return fromRecord(updated), nil
}

func (s *Service) Get(ctx context.Context, id string) (models.ChangeRequest, error) {
	record, err := s.store.GetChangeRequest(ctx, id)
	return fromRecord(record), err
}

func (s *Service) List(ctx context.Context, status string) ([]models.ChangeRequest, error) {
	records, err := s.store.ListChangeRequests(ctx, strings.TrimSpace(status))
	if err != nil {
		return nil, err
	}
	requests := make([]models.ChangeRequest, 0, len(records))
	for _, record := range records {
		requests = append(requests, fromRecord(record))
	}
	return requests, nil
}

func (s *Service) MarkExecuting(ctx context.Context, id, jobID string) error {
	return s.store.SetChangeRequestJob(ctx, id, jobID)
}

func (s *Service) ReleaseExecution(ctx context.Context, id, jobID string) error {
	return s.store.ResetChangeRequestJob(ctx, id, jobID)
}

func (s *Service) Complete(ctx context.Context, id, status string) error {
	return s.store.CompleteChangeRequest(ctx, id, status)
}

func fromRecord(record storage.ChangeRequestRecord) models.ChangeRequest {
	return models.ChangeRequest{
		ID:                 record.ID,
		Operation:          record.Operation,
		Resource:           record.Resource,
		Risk:               record.Risk,
		Status:             record.Status,
		RequestedBy:        record.RequestedBy,
		RequestedAt:        record.RequestedAt,
		Justification:      record.Justification,
		MaintenanceStart:   record.MaintenanceStart,
		MaintenanceEnd:     record.MaintenanceEnd,
		Impact:             record.Impact,
		RollbackPlan:       record.RollbackPlan,
		RequiredCapability: record.RequiredCapability,
		JobID:              record.JobID,
		ApprovedBy:         record.ApprovedBy,
		ApprovedAt:         record.ApprovedAt,
		DecisionReason:     record.DecisionReason,
		ExpiresAt:          record.ExpiresAt,
	}
}

func redact(value any) any {
	switch item := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(item))
		for key, nested := range item {
			if sensitiveKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redact(nested)
		}
		return out
	case []any:
		out := make([]any, len(item))
		for i, nested := range item {
			out[i] = redact(nested)
		}
		return out
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	for _, marker := range []string{"password", "passwd", "secret", "token", "credential", "authorization", "cookie", "keytab", "privatekey", "totp", "otp", "recoverycode"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func validateText(name, value string, minimum, maximum int, singleLine bool) error {
	if len(value) < minimum || len(value) > maximum {
		return errors.New(name + " deve conter entre " + fmt.Sprint(minimum) + " e " + fmt.Sprint(maximum) + " caracteres")
	}
	if strings.ContainsRune(value, '\x00') || singleLine && strings.ContainsAny(value, "\r\n") {
		return errors.New(name + " contém caractere de controle não permitido")
	}
	return nil
}
