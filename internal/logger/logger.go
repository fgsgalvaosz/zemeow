package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Logger é a interface centralizada para todos os logs da aplicação
type Logger interface {
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event
	Fatal() *zerolog.Event
	With() zerolog.Context
	Level(level zerolog.Level) Logger
}

// AppLogger implementa a interface Logger usando zerolog
type AppLogger struct {
	logger zerolog.Logger
}

// WhatsAppLogger adapta o logger da aplicação para o whatsmeow
type WhatsAppLogger struct {
	logger zerolog.Logger
	module string
}

var (
	// Global logger instance
	globalLogger *AppLogger
)

// Init inicializa o sistema de logging global
func Init(level string, pretty bool) {
	// Configure log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	// Configure output
	var output io.Writer = os.Stdout
	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Create global logger
	logger := zerolog.New(output).Level(logLevel).With().
		Timestamp().
		Caller().
		Logger()

	globalLogger = &AppLogger{logger: logger}
	log.Logger = logger
}

// Get retorna a instância global do logger
func Get() Logger {
	if globalLogger == nil {
		Init("info", true) // Default initialization
	}
	return globalLogger
}

// GetWithSession retorna um logger com contexto de sessão
func GetWithSession(sessionID string) Logger {
	if globalLogger == nil {
		Init("info", true)
	}
	return &AppLogger{
		logger: globalLogger.logger.With().Str("session_id", sessionID).Logger(),
	}
}

// WhatsAppLoggerInterface define a interface para o logger do WhatsApp
// WhatsAppLoggerInterface implementa waLog.Logger do whatsmeow
type WhatsAppLoggerInterface interface {
	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Sub(module string) waLog.Logger
}

// GetWhatsAppLogger retorna um logger compatível com whatsmeow
func GetWhatsAppLogger(module string) waLog.Logger {
	if globalLogger == nil {
		Init("info", true)
	}

	return &WhatsAppLogger{
		logger: globalLogger.logger.With().Str("module", module).Logger(),
		module: module,
	}
}

// AppLogger methods
func (l *AppLogger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

func (l *AppLogger) Info() *zerolog.Event {
	return l.logger.Info()
}

func (l *AppLogger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

func (l *AppLogger) Error() *zerolog.Event {
	return l.logger.Error()
}

func (l *AppLogger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

func (l *AppLogger) With() zerolog.Context {
	return l.logger.With()
}

func (l *AppLogger) Level(level zerolog.Level) Logger {
	return &AppLogger{logger: l.logger.Level(level)}
}

// WhatsAppLogger methods
func (w *WhatsAppLogger) Errorf(msg string, args ...interface{}) {
	w.logger.Error().Msgf(msg, args...)
}

func (w *WhatsAppLogger) Warnf(msg string, args ...interface{}) {
	w.logger.Warn().Msgf(msg, args...)
}

func (w *WhatsAppLogger) Infof(msg string, args ...interface{}) {
	w.logger.Info().Msgf(msg, args...)
}

func (w *WhatsAppLogger) Debugf(msg string, args ...interface{}) {
	w.logger.Debug().Msgf(msg, args...)
}

func (w *WhatsAppLogger) Sub(module string) waLog.Logger {
	return &WhatsAppLogger{
		logger: w.logger.With().Str("submodule", module).Logger(),
		module: w.module + "/" + module,
	}
}

// Context helpers
func WithContext(ctx context.Context) Logger {
	if globalLogger == nil {
		Init("info", true)
	}
	return &AppLogger{logger: globalLogger.logger.With().Ctx(ctx).Logger()}
}

func FromContext(ctx context.Context) Logger {
	if l := zerolog.Ctx(ctx); l != nil {
		return &AppLogger{logger: *l}
	}
	return Get()
}