package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger interface abstracts the logging functionality
type Logger interface {
	Debug() Event
	Info() Event
	Warn() Event
	Error() Event
	Fatal() Event
	Panic() Event

	With() Context
	WithContext(ctx context.Context) Logger
	Level(level zerolog.Level) Logger
}

// Event interface for log events
type Event interface {
	Str(key, val string) Event
	Int(key string, i int) Event
	Int64(key string, i int64) Event
	Float64(key string, f float64) Event
	Bool(key string, b bool) Event
	Err(err error) Event
	Dict(key string, dict *zerolog.Event) Event
	Dur(key string, d time.Duration) Event
	Time(key string, t time.Time) Event
	Any(key string, i interface{}) Event
	Interface(key string, i interface{}) Event
	Msg(msg string)
	Msgf(format string, v ...interface{})
	Send()
}

// Context interface for logger context
type Context interface {
	Str(key, val string) Context
	Int(key string, i int) Context
	Int64(key string, i int64) Context
	Float64(key string, f float64) Context
	Bool(key string, b bool) Context
	Err(err error) Context
	Dict(key string, dict *zerolog.Event) Context
	Dur(key string, d time.Duration) Context
	Time(key string, t time.Time) Context
	Any(key string, i interface{}) Context
	Logger() Logger
}

// zerologLogger wraps zerolog.Logger to implement our Logger interface
type zerologLogger struct {
	logger zerolog.Logger
}

// zerologEvent wraps zerolog.Event to implement our Event interface
type zerologEvent struct {
	event *zerolog.Event
}

// zerologContext wraps zerolog.Context to implement our Context interface
type zerologContext struct {
	context zerolog.Context
}

// New creates a new logger instance
func New(level string, pretty bool) Logger {
	var output io.Writer = os.Stdout

	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	// Create logger with global configuration
	logger := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set global logger
	log.Logger = logger

	return &zerologLogger{logger: logger}
}

// NewWithWriter creates a logger with a specific writer
func NewWithWriter(writer io.Writer, level string) Logger {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	logger := zerolog.New(writer).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	return &zerologLogger{logger: logger}
}

// Implementation of Logger interface
func (l *zerologLogger) Debug() Event {
	return &zerologEvent{event: l.logger.Debug()}
}

func (l *zerologLogger) Info() Event {
	return &zerologEvent{event: l.logger.Info()}
}

func (l *zerologLogger) Warn() Event {
	return &zerologEvent{event: l.logger.Warn()}
}

func (l *zerologLogger) Error() Event {
	return &zerologEvent{event: l.logger.Error()}
}

func (l *zerologLogger) Fatal() Event {
	return &zerologEvent{event: l.logger.Fatal()}
}

func (l *zerologLogger) Panic() Event {
	return &zerologEvent{event: l.logger.Panic()}
}

func (l *zerologLogger) With() Context {
	return &zerologContext{context: l.logger.With()}
}

func (l *zerologLogger) WithContext(ctx context.Context) Logger {
	contextLogger := zerolog.Ctx(ctx)
	if contextLogger.GetLevel() == zerolog.Disabled {
		// If no logger in context, return current logger
		return l
	}
	return &zerologLogger{logger: *contextLogger}
}

func (l *zerologLogger) Level(level zerolog.Level) Logger {
	return &zerologLogger{logger: l.logger.Level(level)}
}

// Implementation of Event interface
func (e *zerologEvent) Str(key, val string) Event {
	return &zerologEvent{event: e.event.Str(key, val)}
}

func (e *zerologEvent) Int(key string, i int) Event {
	return &zerologEvent{event: e.event.Int(key, i)}
}

func (e *zerologEvent) Int64(key string, i int64) Event {
	return &zerologEvent{event: e.event.Int64(key, i)}
}

func (e *zerologEvent) Float64(key string, f float64) Event {
	return &zerologEvent{event: e.event.Float64(key, f)}
}

func (e *zerologEvent) Bool(key string, b bool) Event {
	return &zerologEvent{event: e.event.Bool(key, b)}
}

func (e *zerologEvent) Err(err error) Event {
	return &zerologEvent{event: e.event.Err(err)}
}

func (e *zerologEvent) Dict(key string, dict *zerolog.Event) Event {
	return &zerologEvent{event: e.event.Dict(key, dict)}
}

func (e *zerologEvent) Dur(key string, d time.Duration) Event {
	return &zerologEvent{event: e.event.Dur(key, d)}
}

func (e *zerologEvent) Time(key string, t time.Time) Event {
	return &zerologEvent{event: e.event.Time(key, t)}
}

func (e *zerologEvent) Any(key string, i interface{}) Event {
	return &zerologEvent{event: e.event.Interface(key, i)}
}

func (e *zerologEvent) Interface(key string, i interface{}) Event {
	return &zerologEvent{event: e.event.Interface(key, i)}
}

func (e *zerologEvent) Msg(msg string) {
	e.event.Msg(msg)
}

func (e *zerologEvent) Msgf(format string, v ...interface{}) {
	e.event.Msgf(format, v...)
}

func (e *zerologEvent) Send() {
	e.event.Send()
}

// Implementation of Context interface
func (c *zerologContext) Str(key, val string) Context {
	return &zerologContext{context: c.context.Str(key, val)}
}

func (c *zerologContext) Int(key string, i int) Context {
	return &zerologContext{context: c.context.Int(key, i)}
}

func (c *zerologContext) Int64(key string, i int64) Context {
	return &zerologContext{context: c.context.Int64(key, i)}
}

func (c *zerologContext) Float64(key string, f float64) Context {
	return &zerologContext{context: c.context.Float64(key, f)}
}

func (c *zerologContext) Bool(key string, b bool) Context {
	return &zerologContext{context: c.context.Bool(key, b)}
}

func (c *zerologContext) Err(err error) Context {
	return &zerologContext{context: c.context.Err(err)}
}

func (c *zerologContext) Dict(key string, dict *zerolog.Event) Context {
	return &zerologContext{context: c.context.Dict(key, dict)}
}

func (c *zerologContext) Dur(key string, d time.Duration) Context {
	return &zerologContext{context: c.context.Dur(key, d)}
}

func (c *zerologContext) Time(key string, t time.Time) Context {
	return &zerologContext{context: c.context.Time(key, t)}
}

func (c *zerologContext) Any(key string, i interface{}) Context {
	return &zerologContext{context: c.context.Interface(key, i)}
}

func (c *zerologContext) Logger() Logger {
	return &zerologLogger{logger: c.context.Logger()}
}

// Global logger functions for convenience
var globalLogger Logger

func InitGlobal(level string, pretty bool) {
	globalLogger = New(level, pretty)
}

func Debug() Event {
	if globalLogger == nil {
		globalLogger = New("info", false)
	}
	return globalLogger.Debug()
}

func Info() Event {
	if globalLogger == nil {
		globalLogger = New("info", false)
	}
	return globalLogger.Info()
}

func Warn() Event {
	if globalLogger == nil {
		globalLogger = New("info", false)
	}
	return globalLogger.Warn()
}

func Error() Event {
	if globalLogger == nil {
		globalLogger = New("info", false)
	}
	return globalLogger.Error()
}

func Fatal() Event {
	if globalLogger == nil {
		globalLogger = New("info", false)
	}
	return globalLogger.Fatal()
}

func With() Context {
	if globalLogger == nil {
		globalLogger = New("info", false)
	}
	return globalLogger.With()
}
