package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// ComponentLogger - Logger com contexto de componente fixo
type ComponentLogger struct {
	logger    zerolog.Logger
	component string
}

// OperationLogger - Logger para rastreamento de operações com duração
type OperationLogger struct {
	logger    zerolog.Logger
	component string
	operation string
	startTime time.Time
}

// RequestLogger - Logger com contexto completo de requisição
type RequestLogger struct {
	logger    zerolog.Logger
	component string
	sessionID string
	requestID string
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
	InitWithConfig(level, pretty, true, true) // Mantém backward compatibility
}

func InitWithConfig(level string, pretty bool, color bool, includeCaller bool) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	var output io.Writer = os.Stdout
	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    !color,
			FormatLevel: func(i interface{}) string {
				if !color {
					return strings.ToUpper(fmt.Sprintf("%-5s", i))
				}
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
				// Extrai apenas o nome do arquivo sem o caminho completo
				if str, ok := i.(string); ok {
					// Separar arquivo:linha
					parts := strings.Split(str, ":")
					if len(parts) >= 2 {
						// Pegar apenas o nome do arquivo
						filename := filepath.Base(parts[0])
						// Retornar formato filename:linha
						if color {
							return "\x1b[90m" + filename + ":" + parts[1] + "\x1b[0m" // Dark gray
						} else {
							return filename + ":" + parts[1]
						}
					}
				}
				if color {
					return "\x1b[90m" + fmt.Sprintf("%v", i) + "\x1b[0m" // Dark gray fallback
				} else {
					return fmt.Sprintf("%v", i)
				}
			},
			FormatFieldName: func(i interface{}) string {
				if color {
					return "\x1b[36m" + i.(string) + "\x1b[0m=" // Cyan
				} else {
					return i.(string) + "="
				}
			},
			FormatFieldValue: func(i interface{}) string {
				if color {
					return "\x1b[37m" + fmt.Sprintf("%v", i) + "\x1b[0m" // White
				} else {
					return fmt.Sprintf("%v", i)
				}
			},
		}
	}

	loggerBuilder := zerolog.New(output).Level(logLevel).With().Timestamp()
	if includeCaller {
		loggerBuilder = loggerBuilder.Caller()
	}
	logger := loggerBuilder.Logger()

	globalLogger = &AppLogger{logger: logger}
	log.Logger = logger
}

// InitFromConfig inicializa o logger usando a configuração completa do projeto
func InitFromConfig(cfg interface{}) {
	// Tentativa de extrair configuração de logging
	if config, ok := cfg.(interface {
		Level() string
		Pretty() bool
		Color() bool
		IncludeCaller() bool
	}); ok {
		InitWithConfig(config.Level(), config.Pretty(), config.Color(), config.IncludeCaller())
	} else {
		// Fallback para configuração padrão
		InitWithConfig("info", true, true, false) // Desabilita caller por padrão
	}
}

// InitSimple é uma versão simplificada que desabilita o caller por padrão
func InitSimple(level string, pretty bool) {
	InitWithConfig(level, pretty, true, false) // Desabilita caller
}

func Get() Logger {
	if globalLogger == nil {
		InitSimple("info", true) // Usa versão sem caller
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

// === ComponentLogger Methods ===

// ForComponent creates a new ComponentLogger with the specified component
func ForComponent(component string) *ComponentLogger {
	if globalLogger == nil {
		Init("info", true)
	}
	return &ComponentLogger{
		logger:    globalLogger.logger.With().Str("component", component).Logger(),
		component: component,
	}
}

// ForOperation creates an OperationLogger for tracking operations with duration
func (cl *ComponentLogger) ForOperation(operation string) *OperationLogger {
	return &OperationLogger{
		logger:    cl.logger.With().Str("operation", operation).Logger(),
		component: cl.component,
		operation: operation,
		startTime: time.Now(),
	}
}

// WithSession adds session context to the ComponentLogger
func (cl *ComponentLogger) WithSession(sessionID string) *ComponentLogger {
	return &ComponentLogger{
		logger:    cl.logger.With().Str("session_id", sessionID).Logger(),
		component: cl.component,
	}
}

// WithRequest adds request context to the ComponentLogger
func (cl *ComponentLogger) WithRequest(requestID string) *ComponentLogger {
	return &ComponentLogger{
		logger:    cl.logger.With().Str("request_id", requestID).Logger(),
		component: cl.component,
	}
}

// Standard logging methods for ComponentLogger
func (cl *ComponentLogger) Debug() *zerolog.Event {
	return cl.logger.Debug()
}

func (cl *ComponentLogger) Info() *zerolog.Event {
	return cl.logger.Info()
}

func (cl *ComponentLogger) Warn() *zerolog.Event {
	return cl.logger.Warn()
}

func (cl *ComponentLogger) Error() *zerolog.Event {
	return cl.logger.Error()
}

func (cl *ComponentLogger) Fatal() *zerolog.Event {
	return cl.logger.Fatal()
}

func (cl *ComponentLogger) With() zerolog.Context {
	return cl.logger.With()
}

// === OperationLogger Methods ===

// Success logs successful operation completion with duration
func (ol *OperationLogger) Success() *zerolog.Event {
	return ol.logger.Info().
		Str("status", "success").
		Dur("duration", time.Since(ol.startTime))
}

// Failed logs failed operation with error code and duration
func (ol *OperationLogger) Failed(errorCode string) *zerolog.Event {
	return ol.logger.Error().
		Str("status", "failed").
		Str("error_code", errorCode).
		Dur("duration", time.Since(ol.startTime))
}

// WithDuration adds duration to any log event
func (ol *OperationLogger) WithDuration() *zerolog.Event {
	return ol.logger.Info().
		Dur("duration", time.Since(ol.startTime))
}

// InProgress logs operation in progress
func (ol *OperationLogger) InProgress() *zerolog.Event {
	return ol.logger.Info().
		Str("status", "in_progress")
}

// Starting logs operation start
func (ol *OperationLogger) Starting() *zerolog.Event {
	return ol.logger.Info().
		Str("status", "starting")
}

// Standard logging methods for OperationLogger
func (ol *OperationLogger) Debug() *zerolog.Event {
	return ol.logger.Debug()
}

func (ol *OperationLogger) Info() *zerolog.Event {
	return ol.logger.Info()
}

func (ol *OperationLogger) Warn() *zerolog.Event {
	return ol.logger.Warn()
}

func (ol *OperationLogger) Error() *zerolog.Event {
	return ol.logger.Error()
}

func (ol *OperationLogger) Fatal() *zerolog.Event {
	return ol.logger.Fatal()
}

// === RequestLogger Methods ===

// ForRequestContext creates a RequestLogger with complete request context
func ForRequestContext(component, sessionID, requestID string) *RequestLogger {
	if globalLogger == nil {
		Init("info", true)
	}
	return &RequestLogger{
		logger: globalLogger.logger.With().
			Str("component", component).
			Str("session_id", sessionID).
			Str("request_id", requestID).
			Logger(),
		component: component,
		sessionID: sessionID,
		requestID: requestID,
	}
}

// ForOperation creates an OperationLogger with request context
func (rl *RequestLogger) ForOperation(operation string) *OperationLogger {
	return &OperationLogger{
		logger: rl.logger.With().Str("operation", operation).Logger(),
		component: rl.component,
		operation: operation,
		startTime: time.Now(),
	}
}

// WithUser adds user context to the RequestLogger
func (rl *RequestLogger) WithUser(userID string) *RequestLogger {
	return &RequestLogger{
		logger:    rl.logger.With().Str("user_id", userID).Logger(),
		component: rl.component,
		sessionID: rl.sessionID,
		requestID: rl.requestID,
	}
}

// Standard logging methods for RequestLogger
func (rl *RequestLogger) Debug() *zerolog.Event {
	return rl.logger.Debug()
}

func (rl *RequestLogger) Info() *zerolog.Event {
	return rl.logger.Info()
}

func (rl *RequestLogger) Warn() *zerolog.Event {
	return rl.logger.Warn()
}

func (rl *RequestLogger) Error() *zerolog.Event {
	return rl.logger.Error()
}

func (rl *RequestLogger) Fatal() *zerolog.Event {
	return rl.logger.Fatal()
}

func (rl *RequestLogger) With() zerolog.Context {
	return rl.logger.With()
}

// === Utility Functions ===

// GetStandardizedMessage returns standardized message based on operation and status
func GetStandardizedMessage(component, operation, status string) string {
	switch status {
	case "starting":
		return fmt.Sprintf("%s %s starting", strings.Title(component), operation)
	case "success":
		return fmt.Sprintf("%s %s completed successfully", strings.Title(component), operation)
	case "failed":
		return fmt.Sprintf("%s %s failed", strings.Title(component), operation)
	case "in_progress":
		return fmt.Sprintf("%s %s in progress", strings.Title(component), operation)
	default:
		return fmt.Sprintf("%s %s %s", strings.Title(component), operation, status)
	}
}
