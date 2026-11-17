package reloader

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/logger/sl"
	"github.com/fsnotify/fsnotify"
)

type Reloader struct {
	log      *slog.Logger
	path     string
	reload   func() error
	debounce time.Duration
}

func New(log *slog.Logger, path string, reload func() error) *Reloader {
	return &Reloader{
		log:      log,
		path:     path,
		reload:   reload,
		debounce: 300 * time.Millisecond,
	}
}

func (r *Reloader) Run(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	var events <-chan fsnotify.Event
	var watchErrs <-chan error

	watcher, err := r.watchConfigDir()
	if err != nil {
		r.log.Warn("config file watching disabled", sl.Err(err))
	} else {
		defer watcher.Close()
		events, watchErrs = watcher.Events, watcher.Errors
	}

	configName := filepath.Base(r.path)

	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			r.reloadAndLog()
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}

			if filepath.Base(ev.Name) != configName {
				continue
			}

			if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) {
				continue
			}

			if timer == nil {
				timer = time.NewTimer(r.debounce)
			} else {
				timer.Reset(r.debounce)
			}
			timerC = timer.C

		case <-timerC:
			r.reloadAndLog()
		case err, ok := <-watchErrs:
			if !ok {
				watchErrs = nil
				continue
			}
			r.log.Error("config watcher error", sl.Err(err))
		}
	}
}

func (r *Reloader) watchConfigDir() (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(filepath.Dir(r.path)); err != nil {
		watcher.Close()
		return nil, err
	}
	return watcher, nil
}

func (r *Reloader) reloadAndLog() {
	if err := r.reload(); err != nil {
		r.log.Error("config reload failed", sl.Err(err))
	}
}