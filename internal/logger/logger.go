package logger

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dpotapov/slogpfx"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger/log"
	"github.com/MunifTanjim/stremthru/internal/posthog"
)

type Logger = log.Logger

var _ = func() *struct{} {
	w := os.Stderr

	var handler slog.Handler

	if config.LogFormat == "json" {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:       config.LogLevel,
			ReplaceAttr: log.JSONReplaceAttr,
		})
	} else {
		handler = slogpfx.NewHandler(
			tint.NewHandler(w, &tint.Options{
				Level:       config.LogLevel,
				NoColor:     !isatty.IsTerminal(w.Fd()),
				ReplaceAttr: log.PrettyReplaceAttr,
				TimeFormat:  time.DateTime,
			}),
			&slogpfx.HandlerOptions{
				PrefixKeys: []string{"scope"},
			},
		)
	}

	handler = posthog.WrapLogHandler(handler)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelInfo)

	return nil
}()

func New(ctx context.Context, args ...any) *Logger {
	return log.New(ctx, args...)
}

func Scoped(scope string) *Logger {
	return New(context.Background(), "scope", scope)
}
