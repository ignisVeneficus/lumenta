package validate

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Standardized error message helpers

func ErrRequired(field string) error {
	return fmt.Errorf("%s is required", field)
}

func ErrMin(field string, min any, value any) error {
	return fmt.Errorf("%s must be at least %v (got %v)", field, min, value)
}

func ErrOneOf(field string, allowed any, value any) error {
	return fmt.Errorf("%s must be one of %v (got %v)", field, allowed, value)
}

func RequireString(v *ValidationErrors, path string, value string) bool {
	if strings.TrimSpace(value) == "" {
		err := ErrRequired(path)
		LogConfigError(path, value, err)
		v.Add(err)
		return false
	}
	LogConfigOK(path, value)
	return true
}

func RequireIntMin(v *ValidationErrors, path string, value int, min int) bool {
	if value < min {
		err := ErrMin(path, min, value)
		LogConfigError(path, value, err)
		v.Add(err)
		return false
	}
	LogConfigOK(path, value)
	return true
}

func RequireOneOf[T comparable](v *ValidationErrors, path string, value T, allowed []T) bool {
	for _, a := range allowed {
		if value == a {
			LogConfigOK(path, value)
			return true
		}
	}
	err := ErrOneOf(path, allowed, value)
	LogConfigError(path, value, err)
	v.Add(err)
	return false
}

type ValidationErrors struct {
	errors []error
}

func (v *ValidationErrors) Add(err error) {
	if err != nil {
		v.errors = append(v.errors, err)
	}
}

func (v *ValidationErrors) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *ValidationErrors) Error() string {
	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range v.errors {
		sb.WriteString(" - ")
		sb.WriteString(err.Error())
		sb.WriteRune('\n')
	}
	return sb.String()
}

func LogConfigOK(path string, value any) {
	log.Logger.Info().
		Str("config", path).
		Interface("value", value).
		Msg("config set")
}

func LogConfigError(path string, value any, err error) {
	log.Logger.Error().
		Str("config", path).
		Interface("value", value).
		Err(err).
		Msg("invalid config value")
}

func CheckDir(pathKey string, dir string, required bool, v *ValidationErrors) {
	if dir == "" {
		if required {
			err := errors.New("directory must be set")
			LogConfigError(pathKey, dir, err)
			v.Add(fmt.Errorf("%s: %w", pathKey, err))
		} else {
			log.Info().Str("config", pathKey).Msg("directory not set (optional)")
		}
		return
	}

	info, err := os.Stat(dir)
	if err != nil {
		if required {
			LogConfigError(pathKey, dir, err)
			v.Add(fmt.Errorf("%s: %w", pathKey, err))
		} else {
			log.Warn().Str("config", pathKey).Str("value", dir).Err(err).Msg("optional directory does not exist")
		}
		return
	}

	if !info.IsDir() {
		err := errors.New("not a directory")
		if required {
			LogConfigError(pathKey, dir, err)
			v.Add(fmt.Errorf("%s: %w", pathKey, err))
		} else {
			log.Warn().
				Str("config", pathKey).
				Str("value", dir).
				Msg("optional path exists but is not a directory")
		}
		return
	}

	// OK
	LogConfigOK(pathKey, dir)
}

func CheckDuration(v *ValidationErrors, path string, d time.Duration) {
	if d <= 0 {
		err := errors.New("must be > 0")
		LogConfigError(path, d, err)
		v.Add(fmt.Errorf("%s %w", path, err))
	} else {
		LogConfigOK(path, d)
	}
}
