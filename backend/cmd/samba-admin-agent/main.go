package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/agent"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
)

func main() {
	// The privileged process never creates group/world-readable artifacts.
	_ = syscall.Umask(0o077)
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err = agent.PrepareSocketDirectory(filepath.Dir(cfg.AgentSocket), cfg.AgentSocketOwner, cfg.AgentSocketGroup); err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	executor, err := buildExecutor(cfg)
	if err != nil {
		log.Fatal(err)
	}
	capabilities := make(map[string]bool, len(cfg.AgentCapabilities))
	for _, capability := range cfg.AgentCapabilities {
		capabilities[capability] = true
	}
	srvImpl, err := agent.NewServerWithOptions(agent.ServerOptions{
		KeyRing: agent.KeyRing{
			CurrentKey:  cfg.AgentSharedKey,
			CurrentID:   cfg.AgentKeyID,
			PreviousKey: cfg.AgentPreviousSharedKey,
			PreviousID:  cfg.AgentPreviousKeyID,
		},
		Executor:                executor,
		Logger:                  logger,
		MaxMessageBytes:         cfg.AgentMaxMessageBytes,
		MaxOutputBytes:          cfg.AgentMaxOutputBytes,
		MaxConcurrent:           cfg.AgentMaxConcurrent,
		MaxClockSkew:            cfg.AgentMaxClockSkew,
		NonceTTL:                cfg.AgentNonceTTL,
		NonceCacheLimit:         cfg.AgentNonceCacheLimit,
		NonceStorePath:          cfg.AgentNonceStorePath,
		RequireCapabilities:     cfg.AgentExecutorMode == "readonly-freebsd",
		EnabledCapabilities:     capabilities,
		EnableMutableOperations: cfg.EnableMutableOperations,
	})
	if err != nil {
		log.Fatal(err)
	}
	ln, err := agent.ListenWithOptions(cfg.AgentSocket, agent.SocketOptions{
		Mode:                   cfg.AgentSocketMode,
		Owner:                  cfg.AgentSocketOwner,
		Group:                  cfg.AgentSocketGroup,
		ExpectedPeerUID:        cfg.AgentExpectedPeerUID,
		EnforcePeerCredentials: cfg.AgentExpectedPeerUID >= 0,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(cfg.AgentSocket)
	httpServer := &http.Server{Handler: srvImpl.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 5 * time.Minute, IdleTimeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		logger.Info("agente iniciado", "socket", cfg.AgentSocket, "mode", cfg.AdapterMode)
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdown)
}

func buildExecutor(cfg config.Config) (agent.Executor, error) {
	switch cfg.AgentExecutorMode {
	case "mock":
		return agent.MockExecutor{Scenario: cfg.DevScenario}, nil
	case "fixture":
		return agent.NewFixtureExecutor(cfg.FixturePath)
	case "readonly-freebsd":
		return agent.NewReadOnlyExecutor(cfg.AgentMaxOutputBytes), nil
	default:
		return nil, errors.New("executor do agente não reconhecido")
	}
}
