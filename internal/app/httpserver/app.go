package httpserverapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/logger/sl"
)

type App struct {
	log     *slog.Logger
	server  *http.Server
	address string
}

func New(log *slog.Logger, cfg config.Server, handler http.Handler) *App {
	srv := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &App{
		log:     log,
		server:  srv,
		address: cfg.Address,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		a.log.Error("failed to run HTTP server", sl.Err(err))
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "httpserverapp.Run"

	a.log.Info("starting HTTP server", slog.String("address", a.address))

	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop(ctx context.Context) error {
	const op = "httpserverapp.Stop"

	a.log.Info("stopping HTTP server", slog.String("address", a.address))

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
