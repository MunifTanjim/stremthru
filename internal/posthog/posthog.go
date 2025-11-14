package posthog

import (
	"context"
	"log"
	"log/slog"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/posthog/posthog-go"
)

var IsEnabled = config.Posthog.IsEnabled()

var client = func() posthog.Client {
	if !IsEnabled {
		return nil
	}

	client, err := posthog.NewWithConfig(
		config.Posthog.APIKey,
		posthog.Config{},
	)

	if err != nil {
		log.Fatal("failed to initialize", "error", err)
	}

	return client
}()

func Init() posthog.Client {
	return client
}

func Close() error {
	return client.Close()
}

func WrapLogHandler(handler slog.Handler) slog.Handler {
	if !IsEnabled {
		return handler
	}
	posthogHandler := posthog.NewSlogCaptureHandler(handler, client,
		posthog.WithMinCaptureLevel(slog.LevelError),
		posthog.WithDistinctIDFn(func(ctx context.Context, r slog.Record) string {
			return config.Posthog.DistinctId
		}),
		posthog.WithPropertiesFn(func(ctx context.Context, r slog.Record) posthog.Properties {
			prop := posthog.NewProperties()
			if rCtx := server.GetReqCtxFromContext(ctx); rCtx != nil {
				prop.Set("req.id", rCtx.RequestId)
				prop.Set("req.method", rCtx.ReqMethod)
				prop.Set("req.path", rCtx.ReqPath)
			}
			return prop
		}),
	)
	return posthogHandler
}
