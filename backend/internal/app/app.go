package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	agentadapter "github.com/hu-ufcat/samba-admin-backend/internal/adapters/agent"
	fixtureadapter "github.com/hu-ufcat/samba-admin-backend/internal/adapters/fixture"
	"github.com/hu-ufcat/samba-admin-backend/internal/agent"
	"github.com/hu-ufcat/samba-admin-backend/internal/audit"
	"github.com/hu-ufcat/samba-admin-backend/internal/auth"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
	"github.com/hu-ufcat/samba-admin-backend/internal/events"
	"github.com/hu-ufcat/samba-admin-backend/internal/httpapi"
	"github.com/hu-ufcat/samba-admin-backend/internal/jobs"
	"github.com/hu-ufcat/samba-admin-backend/internal/rbac"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

type Application struct {
	Config config.Config
	Store  *storage.Store
	HTTP   *http.Server
	Logger *slog.Logger
}

func Build(ctx context.Context, cfg config.Config) (*Application, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	authSvc := auth.NewWithOptions(store, cfg.SessionTTL, auth.Options{TOTPEncryptionKey: cfg.TOTPEncryptionKey, MFARequiredRoles: cfg.MFARequiredRoles})
	if cfg.AuthMode != "development-bypass" {
		if err = authSvc.Bootstrap(ctx, cfg.BootstrapUser, cfg.BootstrapPassword); err != nil {
			store.Close()
			return nil, err
		}
	}
	broker := events.New()
	auditSvc := audit.New(store)
	var agentClient agent.Client
	if cfg.AdapterMode == "agent" || cfg.AdapterMode == "freebsd-readonly" {
		agentClient = agent.NewUnixClientWithKeyID(cfg.AgentSocket, cfg.AgentSharedKey, cfg.AgentKeyID)
	} else {
		agentClient = agent.MockClient{Scenario: cfg.DevScenario}
	}
	jobsMgr := jobs.New(store, broker, agentClient, auditSvc)
	api := httpapi.New(cfg, store, authSvc, rbac.New(), jobsMgr, broker, auditSvc, logger)
	if cfg.AdapterMode == "agent" || cfg.AdapterMode == "freebsd-readonly" {
		provider, providerErr := agentadapter.New(agentClient)
		if providerErr != nil {
			store.Close()
			return nil, providerErr
		}
		api.SetReadOnlyProvider(provider)
	} else if cfg.AdapterMode == "fixture" {
		provider, openErr := fixtureadapter.Open(cfg.FixturePath)
		if openErr != nil {
			store.Close()
			return nil, openErr
		}
		api.SetReadOnlyProvider(provider)
	}
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Router(), ReadHeaderTimeout: 10e9, ReadTimeout: cfg.RequestTimeout, WriteTimeout: 0, IdleTimeout: 120e9, MaxHeaderBytes: 1 << 20}
	return &Application{Config: cfg, Store: store, HTTP: httpServer, Logger: logger}, nil
}
func (a *Application) Close() error { return a.Store.Close() }
