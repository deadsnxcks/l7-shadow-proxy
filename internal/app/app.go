package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	httpserverapp "github.com/deadsnxcks/l7-shadow-proxy/internal/app/httpserver"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/pipeline"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/proxy"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

type App struct {
	server *httpserverapp.App
}

func New(
	log *slog.Logger,
	cfg *config.Config,
) (*App, error) {
	const op = "app.New"

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.MaxIdleConnsPerHost = 100

	routes, err := router.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	fw := proxy.New(transport)

	rtr := router.New(router.NewIndexed(routes))
	handler := pipeline.New(rtr, fw)

	server := httpserverapp.New(log, cfg.Server, handler)

	return &App{
		server: server,
	}, nil
}

func (a *App) MustRun() {
	go func() {
		a.server.MustRun()
	}()
}

func (a *App) Stop(ctx context.Context) error {
	if err := a.server.Stop(ctx); err != nil {
		return err
	}

	return nil
}
