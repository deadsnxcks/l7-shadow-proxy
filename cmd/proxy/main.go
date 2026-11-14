package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/app"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/logger/sl"
)

const service = "l7-shadow-proxy"

var version = "dev"

func main() {
	configPath := flag.String("config", "configs/example.yaml", "path to proxy config")
	flag.Parse()

	cfg := config.MustLoad(*configPath)

	log := sl.NewLogger(sl.Config{
		Env:     cfg.Env,
		Service: service,
		Version: version,
	})

	log.Info("starting app...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	application, err := app.New(log, cfg)
	if err != nil {
		log.Error("failed to init app", sl.Err(err))
		cancel()
		//nolint:gocritic
		os.Exit(1)
	}

	application.MustRun()

	<-ctx.Done()

	log.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := application.Stop(shutdownCtx); err != nil {
		log.Error("failed to stop server gracefully", sl.Err(err))
	}

	log.Info("server stopped")
}
