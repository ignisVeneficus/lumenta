package metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	metadataConfig "github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/exif"
)

func buildTimeFormats() []string {
	ret := []string{time.RFC3339}
	dateParts := []string{
		"2006-01-02",
		"2006:01:02",
	}

	sepParts := []string{
		"T",
		" ",
	}

	tzParts := []string{
		"",
		"Z",
		"-07:00",
	}

	for _, date := range dateParts {
		for _, sep := range sepParts {
			for _, tz := range tzParts {
				ret = append(ret,
					date+sep+"15:04:05"+tz)
			}
		}
	}
	return ret
}

var timeFormats = buildTimeFormats()

type PathType string

const (
	PathTypeImage   PathType = "image"
	PathTypeSidecar PathType = "sidecar"
)

type Path struct {
	PathType PathType
	Path     string
}

func ExtractMetadata(exiftool *exif.PersistentExiftool, c context.Context, paths ...Path) (data.Metadata, error) {
	logScope, ctx := logging.Enter(c, "metadata/extract", paths[0], map[string]any{"source": paths})

	var metadata data.Metadata
	for i, path := range paths {
		rawdata, err := exiftool.Read(ctx, path.Path)
		if err != nil {
			logging.ExitErr(logScope, err)
			return nil, err
		}
		mtd := resolveMetadata(ctx, rawdata, path.PathType)
		if i == 0 {
			metadata = mtd
		} else {
			for k, d := range mtd {
				metadata[k] = d
			}
		}
	}
	logging.Exit(logScope, "ok", nil)
	return metadata, nil
}

func resolveMetadata(c context.Context, raw exif.RawMetadata, pathType PathType) data.Metadata {
	cfg := config.Global().Sync.MergedMetadata
	out := make(data.Metadata)

	for alias, field := range cfg.Fields {
		if mv, ok := resolveField(c, alias, field, raw, pathType); ok {
			out[alias] = mv
		}
	}
	return out
}

func resolveField(c context.Context, alias string, field metadataConfig.MetadataFieldConfig, raw exif.RawMetadata, pathType PathType) (data.MetadataValue, bool) {
	logScope, _ := logging.Enter(c, "metadata/extract/field", alias, map[string]any{
		"alias":  alias,
		"source": pathType,
	})

	for _, src := range field.Sources {
		ref := strings.ToLower(src.Ref)
		rm, ok := raw[ref]
		if !ok {
			continue
		}

		if !isMeaningfulValue(rm.Value) {
			continue
		}
		value := rm.Value
		if field.Unit != "" {
			if s, ok := value.(string); ok {
				trimmed := stripUnit(s, field.Unit)
				if isMeaningfulValue(trimmed) {
					value = trimmed
				}
			}
		}

		val, err := coerceType(value, field.Type)
		if err != nil {
			logging.ErrorContinue(logScope, err, map[string]any{
				"alias": alias,
				"ref":   src.Ref,
				"value": value,
				"type":  string(field.Type),
			})
			continue
		}
		logging.Exit(logScope, "ok", nil)
		return data.MetadataValue{
			Alias:  alias,
			Ref:    src.Ref,
			Value:  val,
			Type:   field.Type,
			Unit:   field.Unit,
			Source: data.MetadataSource(pathType),
		}, true
	}
	logging.Exit(logScope, "no found", nil)
	return data.MetadataValue{}, false
}

func coerceType(v any, t data.MetadataType) (any, error) {
	if t == "" {
		return v, nil
	}

	switch t {
	case data.MetaString:
		return fmt.Sprint(v), nil

	case data.MetaInt:
		switch x := v.(type) {
		case int:
			return x, nil
		case float64:
			return int(x), nil
		case string:
			return strconv.Atoi(x)
		}

	case data.MetaFloat:
		switch x := v.(type) {
		case float64:
			return x, nil
		case int:
			return float64(x), nil
		case string:
			return strconv.ParseFloat(x, 64)
		}

	case data.MetaList:
		switch x := v.(type) {
		case []any:
			return x, nil
		case []string:
			return x, nil
		case string:
			return []string{x}, nil
		}
	case data.MetaDateTime:
		switch x := v.(type) {
		case time.Time:
			return x, nil
		case string:
			for _, timeFormat := range timeFormats {
				if t, err := time.Parse(timeFormat, x); err == nil {
					return t, nil
				}
			}

		}

	}

	return nil, fmt.Errorf("cannot coerce %T to %s", v, t)
}
func isMeaningfulValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(x) != ""
	case []any:
		return len(x) > 0
	case []string:
		return len(x) > 0
	default:
		return true
	}
}

// StripUnit removes a configured unit from the end of a metadata value.
// Examples:
//
//	StripUnit("48.0 mm", "mm") -> "48.0"
//	StripUnit("0.3 s", "s")    -> "0.3"
//	StripUnit("72mm", "mm")    -> "72"
//	StripUnit("48.0", "mm")    -> "48.0"
func stripUnit(value string, unit string) string {
	v := strings.TrimSpace(value)
	if unit == "" {
		return v
	}

	u := strings.TrimSpace(unit)

	// case-insensitive, allow optional space before unit
	lv := strings.ToLower(v)
	lu := strings.ToLower(u)

	if strings.HasSuffix(lv, " "+lu) {
		return strings.TrimSpace(v[:len(v)-len(u)-1])
	}
	if strings.HasSuffix(lv, lu) {
		return strings.TrimSpace(v[:len(v)-len(u)])
	}

	return v
}
