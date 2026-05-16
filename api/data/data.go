package data

import (
	"encoding/json"
	"reflect"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/validate"
	"github.com/rs/zerolog"
)

type APIStatus string

const (
	StatuszOK    APIStatus = "ok"
	StatuszError APIStatus = "error"
)

type Error struct {
	Error string `json:"error"`
}

func CreateError(msg string) Error {
	return Error{Error: msg}
}

type APIResponse[T any] struct {
	Status APIStatus `json:"status"`
	Data   T         `json:"data"`
	Paging *Paging   `json:"paging,omitempty"`
	Error  *APIError `json:"error,omitempty"`
}

func (ar *APIResponse[T]) HandleError(message string) {
	err := APIError{
		Message: message,
	}
	ar.Error = &err
	ar.Status = StatuszError
}

type APIStatusResponse struct {
	Status APIStatus `json:"status"`
	Error  *APIError `json:"error,omitempty"`
}

func (ar *APIStatusResponse) HandleError(message string) {
	err := APIError{
		Message: message,
	}
	ar.Error = &err
	ar.Status = StatuszError
}
func (ar *APIStatusResponse) HandleValidateErrors(errors validate.ValidationErrors) {
	err := APIError{
		Message: "validation error",
		Fields:  errors,
	}
	ar.Error = &err
	ar.Status = StatuszError
}

type Paging struct {
}

type APIError struct {
	Message string                    `json:"message"`
	Fields  validate.ValidationErrors `json:"fields,omitempty"`
}

type Field[T any] struct {
	Set   bool
	Valid bool
	Value T
}

func (f Field[T]) IsSet() bool {
	return f.Set
}
func (f Field[T]) Get() (T, bool) {
	return f.Value, f.Valid
}
func (f Field[T]) IsNull() bool {
	return f.Set && !f.Valid
}
func (f Field[T]) CheckMandatory() bool {
	if !f.Set {
		return true
	}
	v, valid := f.Get()
	if !valid {
		return false
	}
	return !reflect.ValueOf(v).IsZero()
}
func (f Field[T]) ApplyPtr(dst **T) {
	if !f.Set {
		return
	}
	if !f.Valid {
		*dst = nil
		return
	}
	v := f.Value
	*dst = &v
}
func (f Field[T]) Apply(dst *T) {
	if !f.Set {
		return
	}
	if !f.Valid {
		var zero T
		dst = &zero
		return
	}
	v := f.Value
	dst = &v
}

func (f *Field[T]) UnmarshalJSON(data []byte) error {
	f.Set = true

	if string(data) == "null" {
		f.Valid = false
		var zero T
		f.Value = zero
		return nil
	}

	f.Valid = true
	return json.Unmarshal(data, &f.Value)
}

func FieldUint64If(e *zerolog.Event, key string, f Field[uint64]) {
	if f.Set && f.Valid {
		e.Uint64(key, f.Value)
	}
}
func FieldAlbumIDIf(e *zerolog.Event, key string, f Field[dbo.AlbumID]) {
	if f.Set && f.Valid {
		e.Uint64(key, uint64(f.Value))
	}
}
func FieldImageIDIf(e *zerolog.Event, key string, f Field[dbo.ImageID]) {
	if f.Set && f.Valid {
		e.Uint64(key, uint64(f.Value))
	}
}
func FieldUserIDIf(e *zerolog.Event, key string, f Field[dbo.UserID]) {
	if f.Set && f.Valid {
		e.Uint64(key, uint64(f.Value))
	}
}
func FieldStringIf(e *zerolog.Event, key string, f Field[string]) {
	if f.Set && f.Valid {
		e.Str(key, f.Value)
	}
}
