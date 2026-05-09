// Package logging provides structured, process-oriented logging helpers
// built on top of zerolog.
//
// It enforces a consistent event-based log format across the application.
package logging

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"time"

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

	FieldDuration string = "duration"
)

// TraceIDKey is the context.Context key used to store and retrieve
// the current trace identifier.
const TraceIDKey string = "logging.traceID"

type Scope struct {
	Start time.Time
	Log   zerolog.Logger
}

type LoggingConfig struct {
	zeroconfig.Config `yaml:",inline"`
	Loggers           map[string]zerolog.Level `yaml:"loggers"`
}

//
// ===== Internal helpers =====
//

// newSpanID generates a short random span identifier (8 hex characters).
func newSpanID() string {
	for {
		b := make([]byte, 8)

		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}

		if binary.BigEndian.Uint64(b) != 0 {
			return hex.EncodeToString(b)
		}
	}
}

func newTraceID() string {
	for {
		b := make([]byte, 16)

		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}

		if binary.BigEndian.Uint64(b[:8]) != 0 ||
			binary.BigEndian.Uint64(b[8:]) != 0 {
			return hex.EncodeToString(b)
		}
	}
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
func Enter(ctx context.Context, funcName string, params map[string]any) Scope {

	start := time.Now()
	logg := loggerFromCtx(ctx)

	level, ok := isDebugOrTraceEnabled(logg)
	if !ok {
		return Scope{
			Start: start,
			Log:   logg,
		}
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

	return Scope{
		Log:   l,
		Start: start,
	}
}

// Exit closes a function scope with a successful result
// at debug or trace log levels.
func Exit(scope Scope, result string, params map[string]any) {
	logg := scope.Log
	level, ok := isDebugOrTraceEnabled(logg)
	if !ok {
		return
	}
	dur := time.Since(scope.Start)
	e := logg.Debug().
		Str(FieldEvent, "func.exit").
		Dur(FieldDuration, dur)
	if result != "" {
		e.Str(FieldResult, result)
	}
	if params != nil {
		AddParams(e, level, params)
	}
	e.Msg("")
}

// ExitWarn closes a function scope with an Warning result.
func ExitWarn(scope Scope, err error) {
	dur := time.Since(scope.Start)
	scope.Log.Warn().
		Str(FieldEvent, "func.exit").
		Str(FieldResult, "warning").
		Dur(FieldDuration, dur).
		Err(err).
		Msg("")
}

// ExitErr closes a function scope with an error result.
func ExitErr(scope Scope, err error) {
	dur := time.Since(scope.Start)
	scope.Log.Error().
		Str(FieldEvent, "func.exit").
		Str(FieldResult, "error").
		Dur(FieldDuration, dur).
		Err(err).
		Msg("")
}

// ExitErrParams closes a function scope with an error result
// and additional structured parameters.
func ExitErrParams(scope Scope, err error, params map[string]any) {
	dur := time.Since(scope.Start)
	e := scope.Log.Error().
		Str(FieldEvent, "func.exit").
		Str(FieldResult, "error").
		Dur(FieldDuration, dur).
		Err(err)
	if params != nil {
		AddParams(e, zerolog.DebugLevel, params)
	}
	e.Msg("")
}

// ErrorContinue logs an error without closing the surrounding function scope.
func ErrorContinue(scope Scope, err error, params map[string]any) {
	dur := time.Since(scope.Start)
	e := scope.Log.Error().
		Str(FieldEvent, "func.error").
		Dur(FieldDuration, dur).
		Err(err)
	if params != nil {
		AddParams(e, zerolog.WarnLevel, params)
	}
	e.Msg("")
}

// Inside logs a debug-level intermediate event within a function scope.
func Inside(scope Scope, params map[string]any, msg string) {
	logg := scope.Log
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
func Return(scope Scope, err error) error {
	if err != nil {
		ExitErr(scope, err)
	} else {
		Exit(scope, "ok", nil)
	}
	return err
}

// Return logs the function exit status using the appropriate exit helper
// and returns the provided error unchanged.
func ReturnParams(scope Scope, err error, params map[string]any) error {
	if err != nil {
		ExitErrParams(scope, err, params)
	} else {
		Exit(scope, "ok", params)
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

func Warning(function string, event string, result string, message string, params map[string]any) {
	logg := log.Logger
	WarningLog(logg, function, event, result, message, params)
}
func WarningLog(log zerolog.Logger, function string, event string, result string, message string, params map[string]any) {
	level := log.GetLevel()
	if level > zerolog.WarnLevel {
		return
	}
	e := log.Warn()
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

func Error(ctx context.Context, err error, function string,
	event string, result string, message string, params map[string]any) {
	logg := loggerFromCtx(ctx)

	level, ok := isDebugOrTraceEnabled(logg)
	if !ok {
		return
	}

	e := logg.Error().
		Str(FieldFunc, function).
		Str(FieldSpanID, "").
		Str(FieldResult, result).
		Str(FieldEvent, event)
	if err != nil {
		e.Err(err)
	}
	if ctx != nil {
		if tid, ok := ctx.Value(TraceIDKey).(string); ok {
			e = e.Str(FieldTraceID, tid)
		}
	}
	if params != nil {
		AddParams(e, level, params)
	}
	e.Msg(message)
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
			Msg(file + " is not readable")
		panic(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not readable")
		panic(err)
	}
	var cfg LoggingConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not valid yaml")
		panic(err)
	}
	logger, err := cfg.Compile()
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not valid for zerolog, see go.mau.fi/zeroconfig documentation")
		panic(err)
	}
	log.Logger = *logger
}
