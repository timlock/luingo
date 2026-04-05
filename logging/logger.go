package logging

import (
	"context"
	"log/slog"
)

type loggerKeyCtx struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKeyCtx{}, logger)
}

func Logger(ctx context.Context) *slog.Logger {
	logger := ctx.Value(loggerKeyCtx{}).(*slog.Logger)
	if logger == nil {
		logger = slog.Default()
		logger.Warn("ctx has no logger creating default logger")
	}

	return logger
}
