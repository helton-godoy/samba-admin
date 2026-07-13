package httpapi

import (
	"net/http"

	openapi "github.com/hu-ufcat/samba-admin-backend/api/generated"
)

// openAPIHandler conecta o roteador gerado aos handlers existentes. A
// asserção abaixo faz a compilação falhar quando o contrato ganhar uma rota
// sem implementação explícita na API.
type openAPIHandler struct {
	server *Server
}

var _ openapi.ServerInterface = (*openAPIHandler)(nil)

func (h *openAPIHandler) authorized(action, resource string, handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	h.server.authorize(action, resource, handler).ServeHTTP(w, r)
}

func (h *openAPIHandler) ListAclsLegacy(w http.ResponseWriter, r *http.Request) {
	h.server.legacyACLs(w, r)
}
func (h *openAPIHandler) SimulateAclConversion(w http.ResponseWriter, r *http.Request) {
	h.authorized("write", "acl", h.server.simulateACLConversion, w, r)
}
func (h *openAPIHandler) CalculateEffectiveAcl(w http.ResponseWriter, r *http.Request) {
	h.server.effectiveACL(w, r)
}
func (h *openAPIHandler) ListAcls(w http.ResponseWriter, r *http.Request) {
	h.server.acls(w, r)
}
func (h *openAPIHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	h.server.auditEvents(w, r)
}
func (h *openAPIHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.server.login(w, r)
}
func (h *openAPIHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.server.logout(w, r)
}
func (h *openAPIHandler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	h.server.me(w, r)
}
func (h *openAPIHandler) RotateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	h.server.rotateRecoveryCodes(w, r)
}
func (h *openAPIHandler) ConfirmTotpEnrollment(w http.ResponseWriter, r *http.Request) {
	h.server.confirmTOTPEnrollment(w, r)
}
func (h *openAPIHandler) BeginTotpEnrollment(w http.ResponseWriter, r *http.Request) {
	h.server.beginTOTPEnrollment(w, r)
}
func (h *openAPIHandler) RevokeTotp(w http.ResponseWriter, r *http.Request) {
	h.server.revokeTOTP(w, r)
}
func (h *openAPIHandler) VerifyMfaLogin(w http.ResponseWriter, r *http.Request) {
	h.server.verifyMFA(w, r)
}
func (h *openAPIHandler) RenewSession(w http.ResponseWriter, r *http.Request) {
	h.server.renewSession(w, r)
}
func (h *openAPIHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	h.server.backups(w, r)
}
func (h *openAPIHandler) ListCapabilities(w http.ResponseWriter, r *http.Request) {
	h.server.capabilities(w, r)
}
func (h *openAPIHandler) GetSamba(w http.ResponseWriter, r *http.Request) {
	h.server.samba(w, r)
}
func (h *openAPIHandler) GetCups(w http.ResponseWriter, r *http.Request) {
	h.server.cups(w, r)
}
func (h *openAPIHandler) ListChangeRequests(w http.ResponseWriter, r *http.Request, _ openapi.ListChangeRequestsParams) {
	h.authorized("read", "change", h.server.listChangeRequests, w, r)
}
func (h *openAPIHandler) CreateChangeRequest(w http.ResponseWriter, r *http.Request, _ openapi.CreateChangeRequestParams) {
	h.authorized("write", "change", h.server.withIdempotency(h.server.createChangeRequest), w, r)
}
func (h *openAPIHandler) GetChangeRequest(w http.ResponseWriter, r *http.Request, _ openapi.ChangeRequestID) {
	h.authorized("read", "change", h.server.getChangeRequest, w, r)
}
func (h *openAPIHandler) ApproveChangeRequest(w http.ResponseWriter, r *http.Request, _ openapi.ChangeRequestID, _ openapi.ApproveChangeRequestParams) {
	h.authorized("approve", "change", h.server.withIdempotency(h.server.approveChangeRequest), w, r)
}
func (h *openAPIHandler) ExecuteApprovedChangeRequest(w http.ResponseWriter, r *http.Request, _ openapi.ChangeRequestID, _ openapi.ExecuteApprovedChangeRequestParams) {
	h.authorized("operate", "change", h.server.withIdempotency(h.server.executeChangeRequest), w, r)
}
func (h *openAPIHandler) RejectChangeRequest(w http.ResponseWriter, r *http.Request, _ openapi.ChangeRequestID, _ openapi.RejectChangeRequestParams) {
	h.authorized("approve", "change", h.server.withIdempotency(h.server.rejectChangeRequest), w, r)
}
func (h *openAPIHandler) GetConfigurationFile(w http.ResponseWriter, r *http.Request, _ openapi.GetConfigurationFileParams) {
	h.server.configurationFile(w, r)
}
func (h *openAPIHandler) ValidateConfiguration(w http.ResponseWriter, r *http.Request) {
	h.authorized("write", "file", h.server.validateConfiguration, w, r)
}
func (h *openAPIHandler) ListDfs(w http.ResponseWriter, r *http.Request) {
	h.server.dfs(w, r)
}
func (h *openAPIHandler) GetDomain(w http.ResponseWriter, r *http.Request) {
	h.server.domain(w, r)
}
func (h *openAPIHandler) TestDomain(w http.ResponseWriter, r *http.Request) {
	h.authorized("write", "domain", h.server.domainTest, w, r)
}
func (h *openAPIHandler) StreamEvents(w http.ResponseWriter, r *http.Request, _ openapi.StreamEventsParams) {
	h.server.events(w, r)
}
func (h *openAPIHandler) ListFilesystems(w http.ResponseWriter, r *http.Request) {
	h.server.filesystems(w, r)
}
func (h *openAPIHandler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	h.server.principals(w, r)
}
func (h *openAPIHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	h.server.listJobs(w, r)
}
func (h *openAPIHandler) CancelJob(w http.ResponseWriter, r *http.Request, _ string, _ openapi.CancelJobParams) {
	h.authorized("operate", "job", h.server.withIdempotency(h.server.cancelJob), w, r)
}
func (h *openAPIHandler) RollbackJob(w http.ResponseWriter, r *http.Request, _ string, _ openapi.RollbackJobParams) {
	h.authorized("rollback", "job", h.server.withIdempotency(h.server.rollbackJob), w, r)
}
func (h *openAPIHandler) GetLogging(w http.ResponseWriter, r *http.Request) {
	h.server.logging(w, r)
}
func (h *openAPIHandler) ListMounts(w http.ResponseWriter, r *http.Request) {
	h.server.filesystems(w, r)
}
func (h *openAPIHandler) ListPrincipalsLegacy(w http.ResponseWriter, r *http.Request) {
	h.server.legacyPrincipals(w, r)
}
func (h *openAPIHandler) ListPrintDrivers(w http.ResponseWriter, r *http.Request) {
	h.server.drivers(w, r)
}
func (h *openAPIHandler) ListPrinters(w http.ResponseWriter, r *http.Request) {
	h.server.printers(w, r)
}
func (h *openAPIHandler) ListQuotas(w http.ResponseWriter, r *http.Request) {
	h.server.quotas(w, r)
}
func (h *openAPIHandler) ExecuteRollback(w http.ResponseWriter, r *http.Request, _ openapi.ExecuteRollbackParams) {
	h.authorized("rollback", "configuration", h.server.withIdempotency(h.server.rollbackByBackup), w, r)
}
func (h *openAPIHandler) ListSambaProfiles(w http.ResponseWriter, r *http.Request) {
	h.server.sambaProfiles(w, r)
}
func (h *openAPIHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	h.server.services(w, r)
}
func (h *openAPIHandler) ServiceAction(w http.ResponseWriter, r *http.Request, _ string, _ openapi.ServiceActionParams) {
	h.authorized("operate", "samba", h.server.withIdempotency(h.server.serviceAction), w, r)
}
func (h *openAPIHandler) ListShares(w http.ResponseWriter, r *http.Request, _ openapi.ListSharesParams) {
	h.server.listShares(w, r)
}
func (h *openAPIHandler) CreateShare(w http.ResponseWriter, r *http.Request, _ openapi.CreateShareParams) {
	h.authorized("write", "share", h.server.withIdempotency(h.server.createShare), w, r)
}
func (h *openAPIHandler) PreviewShare(w http.ResponseWriter, r *http.Request) {
	h.authorized("write", "share", h.server.previewShare, w, r)
}
func (h *openAPIHandler) GetSystem(w http.ResponseWriter, r *http.Request) {
	h.server.system(w, r)
}
func (h *openAPIHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.server.health(w, r)
}
func (h *openAPIHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	h.server.ready(w, r)
}
