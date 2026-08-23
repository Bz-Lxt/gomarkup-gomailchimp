package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string, env string) *slog.Logger {
	lv := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		if strings.EqualFold(env, "prod") || strings.EqualFold(env, "production") {
			lv = slog.LevelInfo
		} else {
			lv = slog.LevelDebug
		}
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	return slog.New(h)
}
