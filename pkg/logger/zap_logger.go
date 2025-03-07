package logger

import "go.uber.org/zap"

type ZapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(l *zap.Logger) Logger {
	return &ZapLogger{
		logger: l,
	}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.logger.Debug(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
	z.logger.Info(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.logger.Warn(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Error(msg string, args ...Field) {
	z.logger.Error(msg, z.toArgs(args)...)
}

func (z *ZapLogger) toArgs(args []Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, ar := range args {
		res = append(res, zap.Any(ar.Key, ar.Value))
	}
	return res
}
