package metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ignisVeneficus/lumenta/config"
	metadataConfig "github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/exif"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog/log"
)

func ExtractMetadata(ctx context.Context, paths ...string) (data.Metadata, error) {
	logg := logging.Enter(ctx, "metadata.extract", map[string]any{"source": paths})
	exiftool, err := exif.GetExiftool(ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	var metadata data.Metadata
	for i, path := range paths {
		rawdata, err := exiftool.Read(ctx, path)
		if err != nil {
			logging.ExitErr(logg, err)
			return nil, err
		}
		mtd := resolveMetadata(rawdata)
		if i == 0 {
			metadata = mtd
		} else {
			for k, d := range mtd {
				metadata[k] = d
			}
		}
	}
	logging.Exit(logg, "ok", nil)
	return metadata, nil
}

func resolveMetadata(raw exif.RawMetadata) data.Metadata {
	cfg := config.Global().Sync.MergedMetadata
	out := make(data.Metadata)

	for alias, field := range cfg.Fields {
		if mv, ok := resolveField(alias, field, raw); ok {
			out[alias] = mv
		}
	}
	return out
}

func resolveField(alias string, field metadataConfig.MetadataFieldConfig, raw exif.RawMetadata) (data.MetadataValue, bool) {
	log.Logger.Trace().Str("alias", alias).Msg("parsing for alias")

	for _, src := range field.Sources {
		ref := strings.ToLower(src.Ref)
		rm, ok := raw[ref]
		if !ok {
			continue
		}
		log.Logger.Trace().Str("alias", alias).Str("source", src.Ref).Msg("Source found")

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
			log.Logger.Trace().Str("alias", alias).Str("source", src.Ref).Interface("value", value).Str("type", string(field.Type)).Msg("Type coercion failed")
			continue
		}

		return data.MetadataValue{
			Alias: alias,
			Ref:   src.Ref,
			Value: val,
			Type:  field.Type,
			Unit:  field.Unit,
		}, true
	}

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
