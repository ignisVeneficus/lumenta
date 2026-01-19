package logging

import (
	"io"
	"os"
	"time"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mau.fi/zeroconfig"
	"gopkg.in/yaml.v3"
)

type ObjectWithLevel interface {
	MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level)
}

type withLevel struct {
	level zerolog.Level
	obj   ObjectWithLevel
}

func WithLevel(level zerolog.Level, obj ObjectWithLevel) *withLevel {
	if obj == nil {
		return nil
	}
	return &withLevel{level: level, obj: obj}
}

func (w *withLevel) MarshalZerologObject(e *zerolog.Event) {
	w.obj.MarshalZerologObjectWithLevel(e, w.level)
}

func IntIf(e *zerolog.Event, k string, v *int) {
	if v != nil {
		e.Int(k, *v)
	}
}
func Int16If(e *zerolog.Event, k string, v *int16) {
	if v != nil {
		e.Int16(k, *v)
	}
}
func Uint16If(e *zerolog.Event, k string, v *uint16) {
	if v != nil {
		e.Uint16(k, *v)
	}
}
func Int64If(e *zerolog.Event, k string, v *int64) {
	if v != nil {
		e.Int64(k, *v)
	}
}

func Uint64If(e *zerolog.Event, k string, v *uint64) {
	if v != nil {
		e.Uint64(k, *v)
	}
}

func BoolIf(e *zerolog.Event, k string, v *bool) {
	if v != nil {
		e.Bool(k, *v)
	}
}

func StrIf(e *zerolog.Event, k string, v *string) {
	if v != nil {
		e.Str(k, *v)
	}
}
func FloatIf(e *zerolog.Event, k string, v *float64) {
	if v != nil {
		e.Float64(k, *v)
	}
}
func TimeIf(e *zerolog.Event, k string, v *time.Time) {
	if v != nil {
		e.Time(k, *v)
	}
}
func ObjectIf(e *zerolog.Event, key string, w *withLevel, logNil bool) {
	if w == nil {
		if logNil {
			e.Interface(key, nil)
		}
		return
	}
	e.Object(key, w)
}

func LoadLogging() {

	var f *os.File
	f, err := os.Open(config.GetLogConfigPath())
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
