package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/app"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	a, err := app.Build(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	go func() {
		a.Logger.Info("API iniciada", "addr", cfg.HTTPAddr, "adapterMode", cfg.AdapterMode, "authMode", cfg.AuthMode)
		if err := a.HTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := a.HTTP.Shutdown(shutdown); err != nil {
		a.Logger.Error("falha no graceful shutdown", "error", err)
	}
}
