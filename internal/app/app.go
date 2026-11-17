package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	httpserverapp "github.com/deadsnxcks/l7-shadow-proxy/internal/app/httpserver"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/logger/sl"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/pipeline"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/proxy"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

type App struct {
	server     *httpserverapp.App
	cancel     context.CancelFunc
	log        *slog.Logger
	rtr        *router.Router
	configPath string
}

func New(
	log *slog.Logger,
	cfg *config.Config,
	cfgPath string,
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
		server:     server,
		log:        log,
		rtr:        rtr,
		configPath: cfgPath,
	}, nil
}

func (a *App) MustRun() {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	go a.server.MustRun()
	go a.watchSIGHUP(ctx)
}

func (a *App) Stop(ctx context.Context) error {
	if a.cancel != nil {
		a.cancel()
	}
	return a.server.Stop(ctx)
}

func (a *App) Reload() error {
	const op = "app.Reload"

	log := a.log.With(slog.String("op", op))

	cfg, err := config.Load(a.configPath)
	if err != nil {
		return fmt.Errorf("%s load config: %w", op, err)
	}

	routes, err := router.Build(cfg)
	if err != nil {
		return fmt.Errorf("%s build routes: %w", op, err)
	}

	a.rtr.Update(router.NewIndexed(routes))

	log.Info("routes updated", slog.Int("count", len(routes)))

	return nil
}

func (a *App) watchSIGHUP(ctx context.Context) {
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGHUP)

	defer signal.Stop(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			err := a.Reload()
			if err != nil {
				a.log.Error("failed reload", sl.Err(err))
			}
		}
	}
}
