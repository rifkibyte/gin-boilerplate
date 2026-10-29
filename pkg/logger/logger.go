// Package logger wraps zap with a small, opinionated surface.
//
// The package provides:
//   - a process-wide base logger,
//   - a request-scoped logger that carries the request ID, stored in the Gin
//     context so any layer can retrieve it,
//   - a sugared logger for human-readable output in development.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Format selects the log encoding.
type Format string

const (
	// FormatJSON emits machine-readable JSON. Use it in production.
	FormatJSON Format = "json"
	// FormatConsole emits human-readable colored output. Use it in development.
	FormatConsole Format = "console"
)

// Config configures the base logger.
type Config struct {
	// Level is the minimum level to emit: debug, info, warn, error, or fatal.
	Level string
	// Format selects JSON or console encoding.
	Format Format
	// Service is recorded on every log line.
	Service string
}

// contextKey is the private type for context keys defined by this package.
type contextKey struct{}

// requestIDKey is the private type for the request ID stored in a Gin context.
type requestIDKey struct{}

func (requestIDKey) String() string {
	return "request_id"
}

// New builds a base logger from cfg.
//
// It never returns nil: a configuration that cannot be parsed falls back to
// info level rather than preventing the process from starting.
func New(cfg Config) *zap.Logger {
	level := parseLevel(cfg.Level)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.TimeKey = "ts"
	encoderCfg.MessageKey = "msg"
	encoderCfg.LevelKey = "level"
	encoderCfg.CallerKey = "caller"

	var encoder zapcore.Encoder
	if cfg.Format == FormatConsole {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), level)

	logger := zap.New(core)
	if cfg.Service != "" {
		logger = logger.With(zap.String("service", cfg.Service))
	}

	return logger
}

// parseLevel maps a level name to a zap level, defaulting to info.
func parseLevel(name string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sugar returns a sugared logger for the base logger.
func Sugar(log *zap.Logger) *zap.SugaredLogger {
	if log == nil {
		return zap.NewNop().Sugar()
	}

	return log.Sugar()
}

// FromContext returns the request-scoped logger stored in ctx.
//
// It falls back to a no-op logger when ctx carries none, so calling code never
// has to nil-check. Use it in code that is not running inside an HTTP request.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.NewNop()
	}

	if logger, ok := ctx.Value(contextKey{}).(*zap.Logger); ok && logger != nil {
		return logger
	}

	return zap.NewNop()
}

// WithContext returns a copy of ctx carrying logger.
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromGin returns the request-scoped logger stored in the Gin context.
//
// It falls back to a no-op logger when the middleware has not run.
func FromGin(c *gin.Context) *zap.Logger {
	if c == nil {
		return zap.NewNop()
	}

	if logger, ok := c.Get(ginLoggerKey); ok {
		if requestLogger, ok := logger.(*zap.Logger); ok && requestLogger != nil {
			return requestLogger
		}
	}

	return zap.NewNop()
}

// ginLoggerKey is the key under which the request logger is stored in the Gin
// context.
const ginLoggerKey = "logger"

// SetGin stores a request-scoped logger in the Gin context.
func SetGin(c *gin.Context, logger *zap.Logger) {
	c.Set(ginLoggerKey, logger)
}

// RequestIDFromGin returns the request ID stored in the Gin context, or an
// empty string when the RequestID middleware has not run.
func RequestIDFromGin(c *gin.Context) string {
	if c == nil {
		return ""
	}

	value, ok := c.Get(requestIDKey{}.String())
	if !ok {
		return ""
	}

	requestID, ok := value.(string)
	if !ok {
		return ""
	}

	return requestID
}

// SetRequestID stores the request ID in the Gin context.
func SetRequestID(c *gin.Context, requestID string) {
	c.Set(requestIDKey{}.String(), requestID)
}

// LogConfig is an alias kept so callers do not need to import both this
// package and log/slog. It exists purely for convenience.
type LogConfig = slog.HandlerOptions
