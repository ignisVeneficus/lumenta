// Package logging provides structured, process-oriented logging helpers
// built on top of zerolog.
//
// It enforces a consistent event-based log format across the application.
package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mau.fi/zeroconfig"
	"gopkg.in/yaml.v3"
)

//
// ===== Standard log field names =====
//

const (
	// FieldEvent is the canonical field name for a logical event identifier.
	FieldEvent string = "event"
	// FieldResult represents the logical outcome of an operation (e.g. ok, error).
	FieldResult string = "result"
	// FieldFunc stores the function name associated with a logging scope.
	FieldFunc string = "func"
	// FieldSpanID identifies a logical execution span within a trace.
	FieldSpanID string = "span_id"
	// FieldTraceID identifies a distributed trace across multiple spans.
	FieldTraceID string = "trace_id"
	// FieldParams contains structured parameters associated with a log entry.
	FieldParams string = "params"
)

// TraceIDKey is the context.Context key used to store and retrieve
// the current trace identifier.
const TraceIDKey string = "logging.traceID"

//
// ===== Internal helpers =====
//

// newSpanID generates a short random span identifier (8 hex characters).
func newSpanID() string {
	b := make([]byte, 4) // 8 hex char
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// loggerFromCtx resolves the zerolog.Logger stored in the context.
// If no valid logger is found, the global logger is returned.
func loggerFromCtx(ctx context.Context) zerolog.Logger {
	base := log.Logger
	if ctx == nil {
		return base
	}
	l := zerolog.Ctx(ctx)
	if l == nil {
		return base
	}
	if l.GetLevel() == zerolog.Disabled {
		return base
	}
	return *l
}

// isDebugOrTraceEnabled reports whether debug-level logging is enabled
// and returns the effective log level.
func isDebugOrTraceEnabled(l zerolog.Logger) (zerolog.Level, bool) {
	lv := l.GetLevel()
	return lv, lv <= zerolog.DebugLevel
}

// AddParams populates the FieldParams log field from the provided parameter map.
//
// Values implementing ObjectWithLevel are marshaled with awareness of
// the current log level.
func AddParams(e *zerolog.Event, level zerolog.Level, params map[string]any) {
	d := zerolog.Dict()
	for k, v := range params {
		if v == nil {
			continue
		}

		if m, ok := v.(ObjectWithLevel); ok {
			d.Object(k, WithLevel(level, m))
			continue
		}

		d.Interface(k, v)
	}
	e.Dict(FieldParams, d)
}

//
// ===== Function-scope logging =====
//

// Enter starts a logical function scope and emits a func.enter event
// at debug or trace log levels.
//
// It returns a span-bound logger for subsequent logging within the function.
func Enter(ctx context.Context, funcName string, params map[string]any) zerolog.Logger {

	logg := loggerFromCtx(ctx)

	level, ok := isDebugOrTraceEnabled(logg)
	if !ok {
		return logg
	}
	spanID := newSpanID()

	w := logg.With().
		Str(FieldFunc, funcName).
		Str(FieldSpanID, spanID)

	if ctx != nil {
		if tid, ok := ctx.Value(TraceIDKey).(string); ok {
			w = w.Str(FieldTraceID, tid)
		}
	}
	l := w.Logger()
	e := l.Debug().Str(FieldEvent, "func.enter")
	if params != nil {
		AddParams(e, level, params)
	}
	e.Msg("")

	return l
}

// Exit closes a function scope with a successful result
// at debug or trace log levels.
func Exit(logg zerolog.Logger, result string, params map[string]any) {
	level, ok := isDebugOrTraceEnabled(logg)
	if !ok {
		return
	}
	e := logg.Debug().
		Str(FieldEvent, "func.exit")
	if result != "" {
		e.Str(FieldResult, result)
	}
	if params != nil {
		AddParams(e, level, params)
	}
	e.Msg("")
}

// ExitErr closes a function scope with an error result.
func ExitErr(logg zerolog.Logger, err error) {
	logg.Error().
		Str(FieldEvent, "func.exit").
		Str(FieldResult, "error").
		Err(err).
		Msg("")
}

// ExitErrParams closes a function scope with an error result
// and additional structured parameters.
func ExitErrParams(logg zerolog.Logger, err error, params map[string]any) {
	e := logg.Error().
		Str(FieldEvent, "func.exit").
		Str(FieldResult, "error").
		Err(err)
	if params != nil {
		AddParams(e, zerolog.DebugLevel, params)
	}
	e.Msg("")
}

// ErrorContinue logs an error without closing the surrounding function scope.
func ErrorContinue(logg zerolog.Logger, err error, params map[string]any) {
	e := logg.Error().
		Str(FieldEvent, "func.error").
		Err(err)
	if params != nil {
		AddParams(e, zerolog.WarnLevel, params)
	}
	e.Msg("")
}

// Inside logs a debug-level intermediate event within a function scope.
func Inside(logg zerolog.Logger, params map[string]any, msg string) {
	level, ok := isDebugOrTraceEnabled(logg)
	if !ok {
		return
	}
	e := logg.Debug().
		Str(FieldEvent, "func.inside")
	if params != nil {
		AddParams(e, level, params)
	}
	e.Msg(msg)
}

// Return logs the function exit status using the appropriate exit helper
// and returns the provided error unchanged.
func Return(logg zerolog.Logger, err error) error {
	if err != nil {
		ExitErr(logg, err)
	} else {
		Exit(logg, "ok", nil)
	}
	return err
}

//
// ===== Info-level event logging =====
//

// Info logs a standalone informational event
// when info-level logging is enabled.
func Info(function string, event string, result string, message string, params map[string]any) {
	logg := log.Logger
	InfoLog(logg, function, event, result, message, params)
}

// InfoLog logs a standalone informational event using the provided logger.
func InfoLog(log zerolog.Logger, function string, event string, result string, message string, params map[string]any) {
	level := log.GetLevel()
	if level > zerolog.InfoLevel {
		return
	}
	e := log.Info()
	if function != "" {
		e.Str(FieldFunc, function)
	}
	if event != "" {
		e.Str(FieldEvent, event)
	}
	if result != "" {
		e.Str(FieldResult, result)
	}
	if params != nil {
		AddParams(e, zerolog.InfoLevel, params)
	}
	e.Msg(message)
}

// Error emits an error log entry using the standard process logging format.
//
// The function exists to enforce a consistent error event structure
// across the application, aligned with other process-level logging helpers
// (Enter, ExitErr, ErrorContinue, etc.).
//
// Not all parameters are necessarily rendered in the current implementation;
// their presence defines the canonical error logging interface.
func Error(err error, event string, result string, message string, params map[string]any) {
	log.Logger.Error().Err(err)
}

//
// ===== Object-level logging =====
//

// ObjectWithLevel allows an object to control how it is marshaled
// depending on the active log level.
type ObjectWithLevel interface {
	MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level)
}

// withLevel wraps an ObjectWithLevel with an explicit log level.
type withLevel struct {
	level zerolog.Level
	obj   ObjectWithLevel
}

// WithLevel wraps an ObjectWithLevel for level-aware marshaling.
func WithLevel(level zerolog.Level, obj ObjectWithLevel) *withLevel {
	if obj == nil {
		return nil
	}
	return &withLevel{level: level, obj: obj}
}

// MarshalZerologObject implements zerolog object marshaling.
func (w *withLevel) MarshalZerologObject(e *zerolog.Event) {
	w.obj.MarshalZerologObjectWithLevel(e, w.level)
}

//
// ===== Typed conditional field helpers =====
//

// IntIf logs an int value if not nil.
func IntIf(e *zerolog.Event, k string, v *int) {
	if v != nil {
		e.Int(k, *v)
	}
}

// Int16If logs an int16 value if not nil.
func Int16If(e *zerolog.Event, k string, v *int16) {
	if v != nil {
		e.Int16(k, *v)
	}
}

// Uint16If logs an uint16 value if not nil.
func Uint16If(e *zerolog.Event, k string, v *uint16) {
	if v != nil {
		e.Uint16(k, *v)
	}
}

// Int64If logs an int64 value if not nil.
func Int64If(e *zerolog.Event, k string, v *int64) {
	if v != nil {
		e.Int64(k, *v)
	}
}

// Uint64If logs an uint64 value if not nil.
func Uint64If(e *zerolog.Event, k string, v *uint64) {
	if v != nil {
		e.Uint64(k, *v)
	}
}

// BoolIf logs a bool value if not nil.
func BoolIf(e *zerolog.Event, k string, v *bool) {
	if v != nil {
		e.Bool(k, *v)
	}
}

// StrIf logs a string value if not nil.
func StrIf(e *zerolog.Event, k string, v *string) {
	if v != nil {
		e.Str(k, *v)
	}
}

// Float64If logs a float64 value if not nil.
func Float64If(e *zerolog.Event, k string, v *float64) {
	if v != nil {
		e.Float64(k, *v)
	}
}

// Float32If logs a float32 value if not nil.
func Float32If(e *zerolog.Event, k string, v *float32) {
	if v != nil {
		e.Float32(k, *v)
	}
}

// TimeIf logs a time.Time value if not nil.
func TimeIf(e *zerolog.Event, k string, v *time.Time) {
	if v != nil {
		e.Time(k, *v)
	}
}

// ObjectIf logs a structured object if present.
// If logNil is true, a nil value is explicitly logged.
func ObjectIf(e *zerolog.Event, key string, w *withLevel, logNil bool) {
	if w == nil {
		if logNil {
			e.Interface(key, nil)
		}
		return
	}
	e.Object(key, w)
}

//
// ===== Logger initialization =====
//

// LoadLogging replaces the global zerolog logger by loading
// a zeroconfig YAML configuration from the given file path.
//
// The function aborts the process if the configuration is invalid.
func LoadLogging(file string) {
	var f *os.File
	f, err := os.Open(file)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(config.LogConfigEnv + " is not readable")
		panic(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(config.LogConfigEnv + " is not readable")
		panic(err)
	}
	var cfg zeroconfig.Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(config.LogConfigEnv + " is not valid yaml")
		panic(err)
	}
	logger, err := cfg.Compile()
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(config.LogConfigEnv + " is not valid for zerolog, see go.mau.fi/zeroconfig documentation")
		panic(err)
	}
	log.Logger = *logger
}
