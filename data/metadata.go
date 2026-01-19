package data

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	MetaTakenAt      = "taken_at"
	MetaCamera       = "camera"
	MetaLens         = "lens"
	MetaFocalLength  = "focal_length"
	MetaAperture     = "aperture"
	MetaISO          = "iso"
	MetaLatitude     = "latitude"
	MetaLongitude    = "longitude"
	MetaRotation     = "rotation"
	MetaRating       = "rating"
	MetaTitle        = "title"
	MetaSubject      = "subject"
	MetaTags         = "tags"
	MetaHeight       = "height"
	MetaWidth        = "width"
	MetaMaker        = "maker"
	MetaExposureTime = "exposure_time"
)

type MetadataType string

const (
	MetaString   MetadataType = "string"
	MetaInt      MetadataType = "int"
	MetaFloat    MetadataType = "float"
	MetaRational MetadataType = "rational"
	MetaBool     MetadataType = "bool"
	MetaList     MetadataType = "list"
	MetaDateTime MetadataType = "datetime"
)

type Metadata map[string]MetadataValue

func (mt *Metadata) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if mt == nil || *mt == nil {
		return
	}
	if level <= zerolog.DebugLevel {
		arr := zerolog.Arr()
		for _, mv := range *mt {
			arr.Dict(zerolog.Dict().
				Str("alias", mv.Alias).
				Interface("value", mv.Value).
				Str("type", string(mv.Type)).
				Str("ref", mv.Ref))
		}
		e.Array("", arr)
	}
}

type MetadataValue struct {
	Alias string       `json:"alias"` // user-defined (pl "focal_length")
	Ref   string       `json:"ref"`   // EXIF:FocalLength
	Type  MetadataType `json:"type"`  // string, int, float, rational, list
	Value any          `json:"value"`
	Unit  string       `json:"unit,omitempty"`
	/*
		Lang     string   `json:"lang,omitempty"`
		Priority int      `json:"priority"` // config order
	*/
}

func (m Metadata) getString(key string) *string {
	if v, ok := m[key]; ok {
		if s, ok := v.AsString(); ok {
			return &s
		}
	}
	return nil
}
func (m Metadata) getTime(key string) *time.Time {
	if v, ok := m[key]; ok {
		if t, ok := v.Value.(time.Time); ok {
			return &t
		}
	}
	return nil
}
func (m Metadata) getFloat32(key string) *float32 {
	if v, ok := m[key]; ok {
		if f, ok := v.AsFloat(); ok {
			f32 := float32(f)
			return &f32
		}
	}
	return nil
}
func (m Metadata) getFloat64(key string) *float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.AsFloat(); ok {
			return &f
		}
	}
	return nil
}
func (m Metadata) getUint16(key string) *uint16 {
	if v, ok := m[key]; ok {
		if i, ok := v.AsInt(); ok {
			u := uint16(i)
			return &u
		}
	}
	return nil
}
func (m Metadata) getInt8(key string) *int8 {
	if v, ok := m[key]; ok {
		if i, ok := v.AsInt(); ok {
			u := int8(i)
			return &u
		}
	}
	return nil
}
func (m Metadata) GetTitle() *string {
	return m.getString(MetaTitle)
}
func (m Metadata) GetSubject() *string {
	return m.getString(MetaSubject)
}
func (m Metadata) GetTakenAt() *time.Time {
	return m.getTime(MetaTakenAt)
}
func (m Metadata) GetMaker() *string {
	return m.getString(MetaMaker)
}
func (m Metadata) GetCamera() *string {
	return m.getString(MetaCamera)
}
func (m Metadata) GetMakerCamera() *string {
	maker := m.GetMaker()
	cam := m.GetCamera()
	return join(maker, cam, " ")
}

func (m Metadata) GetLens() *string {
	return m.getString(MetaLens)
}
func (m Metadata) GetFocalLength() *float32 {
	return m.getFloat32(MetaFocalLength)
}
func (m Metadata) GetAperture() *float32 {
	return m.getFloat32(MetaAperture)
}
func (m Metadata) GetIso() *uint16 {
	return m.getUint16(MetaISO)
}
func (m Metadata) GetLatitude() *float64 {
	str := m.getString(MetaLatitude)
	if str == nil {
		return nil
	}
	ret, err := parseDMS(*str)
	if err != nil {
		log.Logger.Warn().Err(err).Str("latitude", *str).Msg("can't convert")
		return nil
	}
	return &ret
}
func (m Metadata) GetLongitude() *float64 {
	str := m.getString(MetaLongitude)
	if str == nil {
		return nil
	}
	ret, err := parseDMS(*str)
	if err != nil {
		log.Logger.Warn().Err(err).Str("longitude", *str).Msg("can't convert")
		return nil
	}
	return &ret
}
func (m Metadata) GetRotation() *int16 {
	str := m.getString(MetaRotation)
	if str == nil {
		return nil
	}
	ret := rotationFromExifOrientation(*str)
	return &ret
}
func (m Metadata) GetRating() *uint16 {
	return m.getUint16(MetaRating)
}
func (m Metadata) GetTags() []string {
	if v, ok := m[MetaTags]; ok {
		if tags, ok := v.AsList(); ok {
			return tags
		}
	}
	return nil
}
func (m Metadata) GetExposure() *float64 {
	str := m.getString(MetaExposureTime)
	if str == nil {
		return nil
	}
	value, err := parseFloatOrFraction(*str)
	if err != nil {
		log.Logger.Warn().Err(err).Str("exposure", *str).Msg("can't convert")
	}
	return &value
}

func (m MetadataValue) AsFloat() (float64, bool) {
	if m.Type != MetaFloat {
		return 0, false
	}
	v, ok := m.Value.(float64)
	return v, ok
}

func (m MetadataValue) AsInt() (int64, bool) {
	if m.Type != MetaInt {
		return 0, false
	}
	switch v := m.Value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	}
	return 0, false
}

func (m MetadataValue) AsString() (string, bool) {
	if m.Type != MetaString {
		return "", false
	}
	v, ok := m.Value.(string)
	return v, ok
}

func (m MetadataValue) AsList() ([]string, bool) {
	if m.Type != MetaList {
		return nil, false
	}
	v, ok := m.Value.([]string)
	return v, ok
}

func join(a, b *string, spacer string) *string {
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		s := *a + spacer + *b
		return &s
	}
}

// parseDMS parses coordinates like:
//
//	`47 deg 29' 22.64" N`
//	`47° 29' 22.64" N`
//	`47 29 22.64 N`
//	`47 deg 29' N`          (seconds optional)
//	`47.489622 N`           (already decimal, optional)
//
// Returns decimal degrees. S/W -> negative.
func parseDMS(s string) (float64, error) {
	in := strings.TrimSpace(s)
	if in == "" {
		return 0, fmt.Errorf("empty coordinate")
	}

	// normalize decimal comma and whitespace
	in = strings.ReplaceAll(in, ",", ".")
	in = strings.Join(strings.Fields(in), " ")

	// If it's already a plain decimal with optional hemisphere at end, accept it:
	decRe := regexp.MustCompile(`(?i)^\s*([+-]?\d+(?:\.\d+)?)\s*([NSEW])?\s*$`)
	if m := decRe.FindStringSubmatch(in); m != nil {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal: %w", err)
		}
		if m[2] != "" {
			h := strings.ToUpper(m[2])
			if h == "S" || h == "W" {
				v = -abs(v)
			} else {
				v = abs(v)
			}
		}
		return v, nil
	}

	// DMS pattern (seconds optional)
	// Examples matched:
	// 47 deg 29' 22.64" N
	// 47° 29' 22.64" N
	// 47 29 22.64 N
	dmsRe := regexp.MustCompile(`(?i)^\s*([+-]?\d+(?:\.\d+)?)\s*(?:deg|°)?\s*` +
		`(?:(\d+(?:\.\d+)?)\s*['’m]?\s*)?` +
		`(?:(\d+(?:\.\d+)?)\s*(?:"|”|s)?\s*)?` +
		`([NSEW])\s*$`)

	m := dmsRe.FindStringSubmatch(in)
	if m == nil {
		return 0, fmt.Errorf("unrecognized coordinate format: %q", s)
	}

	deg, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid degrees: %w", err)
	}

	var min, sec float64
	if m[2] != "" {
		min, err = strconv.ParseFloat(m[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
	}
	if m[3] != "" {
		sec, err = strconv.ParseFloat(m[3], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	}

	// Convert. Allow negative degrees input but hemisphere wins.
	v := abs(deg) + (min / 60.0) + (sec / 3600.0)

	h := strings.ToUpper(m[4])
	if h == "S" || h == "W" {
		v = -v
	}
	return v, nil
}

func rotationFromExifOrientation(v string) int16 {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "horizontal (normal)", "normal":
		return 0
	case "rotate 90 cw", "right-top":
		return 90
	case "rotate 180", "bottom-right":
		return 180
	case "rotate 270 cw", "left-bottom":
		return 270
	default:
		// mirror / unknown -> ignore for now
		return 0
	}
}

// parseFloatOrFraction parses strings like:
//
//	"0.15", "3.5", "1/500"
//
// into float64
func parseFloatOrFraction(s string) (float64, error) {
	str := strings.TrimSpace(s)
	if str == "" {
		return 0, errors.New("empty value")
	}

	// Fraction: a/b
	if strings.Contains(str, "/") {
		parts := strings.SplitN(str, "/", 2)
		if len(parts) != 2 {
			return 0, errors.New("invalid fraction")
		}

		num, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, err
		}

		den, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, err
		}
		if den == 0 {
			return 0, errors.New("division by zero")
		}

		return num / den, nil
	}

	// Regular float
	return strconv.ParseFloat(str, 64)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
