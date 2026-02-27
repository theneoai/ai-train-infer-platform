package logger

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Log is the global logger instance
	Log *zap.Logger
	// SugaredLog is the sugared logger for convenient logging
	SugaredLog *zap.SugaredLogger
)

// Config logger configuration
type Config struct {
	Level      string
	Format     string // json or console
	Output     string // stdout, file, or both
	FilePath   string
	MaxSize    int    // MB
	MaxAge     int    // days
	MaxBackups int
	Compress   bool
}

// DefaultConfig returns default logger config
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		FilePath:   "logs/app.log",
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 3,
		Compress:   true,
	}
}

// Init initializes the global logger
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	level := parseLevel(cfg.Level)

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	} else {
		encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}

	var writers []zapcore.WriteSyncer

	switch cfg.Output {
	case "stdout":
		writers = append(writers, zapcore.AddSync(os.Stdout))
	case "file":
		writers = append(writers, getLogWriter(cfg))
	default: // both
		writers = append(writers, zapcore.AddSync(os.Stdout))
		writers = append(writers, getLogWriter(cfg))
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	SugaredLog = Log.Sugar()

	return nil
}

// InitDevelopment initializes a development-friendly logger
func InitDevelopment() error {
	cfg := &Config{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	}
	return Init(cfg)
}

// InitProduction initializes a production logger
func InitProduction(logPath string) error {
	cfg := DefaultConfig()
	cfg.Level = "info"
	cfg.Format = "json"
	cfg.Output = "both"
	if logPath != "" {
		cfg.FilePath = logPath
	}
	return Init(cfg)
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func getLogWriter(cfg *Config) zapcore.WriteSyncer {
	// Create log directory if not exists
	dir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fallback to stdout if directory creation fails
		return zapcore.AddSync(os.Stdout)
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	return zapcore.AddSync(lumberJackLogger)
}

// WithContext returns a logger with request context
func WithContext(traceID, spanID string) *zap.Logger {
	return Log.With(
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)
}

// WithField returns a logger with a field
func WithField(key string, value interface{}) *zap.Logger {
	return Log.With(zap.Any(key, value))
}

// WithFields returns a logger with multiple fields
func WithFields(fields map[string]interface{}) *zap.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return Log.With(zapFields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal logs a fatal message
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// Sync flushes the logger
func Sync() error {
	return Log.Sync()
}
