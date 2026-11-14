package sl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

type ctxKey int

const (
	traceIDKey ctxKey = iota
	loggerKey
)

// NewTraceID returns 32 hex characters — the same shape as a W3C trace-id, so
// this can be swapped for a real OpenTelemetry trace ID later without changing
// the log schema or any dashboard built on it.
func NewTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail in practice on Linux, and an empty ID is
		// better than failing a request over a log field.
		return ""
	}

	return hex.EncodeToString(b[:])
}

// WithTraceID stores a trace ID in the context. The gRPC logging interceptor
// calls this once per request, reusing the ID from the incoming metadata when
// the caller supplied one.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceID returns the trace ID stored in the context, or "" if there is none.
func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceIDKey).(string)

	return id
}

// Into stores a logger in the context so a gRPC handler can enrich it once
// and every layer below reads the enriched one.
func Into(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

// From returns the logger stored in the context, falling back to the default
// logger so this is always safe to call.
func From(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return log
	}

	return slog.Default()
}

// contextHandler copies the trace ID out of the context into every record, so
// no call site has to remember to attach it. It wraps whichever handler is in
// use — pretty locally, JSON everywhere else.
//
// This only fires for the Context variants of the logging methods:
// log.InfoContext(ctx, ...) rather than log.Info(...), because plain Info
// hands slog a background context that carries nothing.
type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := TraceID(ctx); id != "" {
		r.AddAttrs(slog.String("trace_id", id))
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs and WithGroup have to rewrap. Without them the embedded handler is
// returned bare on the first .With() call, contextHandler drops out of the
// chain, and trace IDs silently stop appearing part-way through a request.
func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}
