package logger

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Logger interface {
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event
	Fatal() *zerolog.Event
	With() zerolog.Context
	Level(level zerolog.Level) Logger
}

type AppLogger struct {
	logger zerolog.Logger
}

type WhatsAppLogger struct {
	logger zerolog.Logger
	module string
}

var (
	globalLogger *AppLogger
)

func Init(level string, pretty bool) {

	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	var output io.Writer = os.Stdout
	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			FormatLevel: func(i interface{}) string {
				switch i {
				case "trace":
					return "\x1b[36mTRACE\x1b[0m" // Cyan
				case "debug":
					return "\x1b[35mDEBUG\x1b[0m" // Magenta
				case "info":
					return "\x1b[32mINFO \x1b[0m" // Green
				case "warn":
					return "\x1b[33mWARN \x1b[0m" // Yellow
				case "error":
					return "\x1b[31mERROR\x1b[0m" // Red
				case "fatal":
					return "\x1b[37;41mFATAL\x1b[0m" // White on Red
				default:
					return "\x1b[37m?????\x1b[0m"
				}
			},
			FormatCaller: func(i interface{}) string {
				return "\x1b[90m" + i.(string) + "\x1b[0m" // Dark gray
			},
			FormatFieldName: func(i interface{}) string {
				return "\x1b[36m" + i.(string) + "\x1b[0m=" // Cyan
			},
			FormatFieldValue: func(i interface{}) string {
				return "\x1b[37m" + fmt.Sprintf("%v", i) + "\x1b[0m" // White
			},
		}
	}

	logger := zerolog.New(output).Level(logLevel).With().
		Timestamp().
		Caller().
		Logger()

	globalLogger = &AppLogger{logger: logger}
	log.Logger = logger
}

func Get() Logger {
	if globalLogger == nil {
		Init("info", true)
	}
	return globalLogger
}

func GetWithSession(sessionID string) Logger {
	if globalLogger == nil {
		Init("info", true)
	}
	return &AppLogger{
		logger: globalLogger.logger.With().Str("session_id", sessionID).Logger(),
	}
}

type WhatsAppLoggerInterface interface {
	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Sub(module string) waLog.Logger
}

func GetWhatsAppLogger(module string) waLog.Logger {
	if globalLogger == nil {
		Init("info", true)
	}

	return &WhatsAppLogger{
		logger: globalLogger.logger.With().Str("module", module).Logger(),
		module: module,
	}
}

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
