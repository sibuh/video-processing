package initiator

import (
	"log/slog"

	slogzap "github.com/samber/slog-zap"
	"go.uber.org/zap"
)

func NewLogger() *slog.Logger {
	zapLogger, _ := zap.NewProduction()
	handler := slogzap.Option{Logger: zapLogger}.NewZapHandler()
	logger := slog.New(handler)

	return logger
}
