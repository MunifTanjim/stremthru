package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/dpotapov/slogpfx"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger/log"
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

	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelInfo)

	return nil
}()

func With(args ...any) *Logger {
	return &Logger{Logger: slog.With(args...)}
}

func Scoped(scope string) *Logger {
	return With("scope", scope)
}
