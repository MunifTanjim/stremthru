package log

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
)

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
)

var levelByString = map[string]slog.Level{
	"TRACE": LevelTrace,
	"FATAL": LevelFatal,
}
var stringByLevel = map[slog.Level]string{
	LevelTrace: "TRACE",
	LevelFatal: "FATAL",
}

type Level struct {
	l slog.Level
}

func (l Level) Level() slog.Level {
	return l.l
}

func (l Level) String() string {
	if str, ok := stringByLevel[l.l]; ok {
		return str
	}
	return l.l.String()
}

func (l *Level) UnmarshalText(data []byte) error {
	if level, ok := levelByString[strings.ToUpper(string(data))]; ok {
		l.l = level
		return nil
	}
	return l.l.UnmarshalText(data)
}

type Logger struct {
	*slog.Logger
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

func (l *Logger) Trace(msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}

func getReplaceAttr(isPretty bool) func(groups []string, a slog.Attr) slog.Attr {
	if isPretty {
		tintedStringByLevel := map[slog.Level]slog.Attr{
			LevelTrace: tint.Attr(145, slog.String("level", "TRC")),
			LevelFatal: tint.Attr(160, slog.String("level", "FTL")),
		}

		return func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if level, ok := a.Value.Any().(slog.Level); ok {
					if tintedAttr, ok := tintedStringByLevel[level]; ok {
						return tintedAttr
					}
				}
			}
			return a
		}
	}

	stringValueByLevel := map[slog.Level]slog.Value{
		LevelTrace: slog.StringValue("TRACE"),
		LevelFatal: slog.StringValue("FATAL"),
	}

	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey {
			if level, ok := a.Value.Any().(slog.Level); ok {
				if strVal, ok := stringValueByLevel[level]; ok {
					a.Value = strVal
				}
			}
		}
		return a
	}
}

var JSONReplaceAttr = getReplaceAttr(false)
var PrettyReplaceAttr = getReplaceAttr(true)
