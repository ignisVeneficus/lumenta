package ruleengine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ImageFacts struct {
	Path     string
	Filename string
	Ext      string

	TakenAt *time.Time
	Rating  *int
	Width   uint32
	Height  uint32

	// hierarchikus tagek: "Travel/Iceland/Winter"
	Tags []string
}

func (i *ImageFacts) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level < zerolog.DebugLevel {
		e.Str("path", i.Path).
			Str("filename", i.Filename).
			Str("ext", i.Ext).
			Uint32("width", i.Width).
			Uint32("height", i.Height).
			Strs("tags", i.Tags)
		logging.TimeIf(e, "taken", i.TakenAt)
		logging.IntIf(e, "rating", i.Rating)
	}
}

type CompiledFilter func(f ImageFacts) bool

func compileAtomicFilter(flt Rule) (CompiledFilter, error) {
	switch f := flt.(type) {
	case *TagFilter:
		return compileTagFilter(f)
	case *DateFilter:
		return compileDateFilter(f)
	case *NameFilter:
		return compileNameFilter(f)
	case *RatingFilter:
		return compileRatingFilter(f)
	case *PathFilter:
		return compilePathFilter(f)
	case *ExtensionFilter:
		return compileExtensionFilter(f)
	case *AlbumFilter:
		return nil, fmt.Errorf("album filter is not atomic-image-evaluable")
	case *NotInChildAlbumsFilter:
		return nil, fmt.Errorf("notchildren filter is not atomic-image-evaluable")
	case *WidthFilter:
		return compileWidthFilter(f)
	case *HeightFilter:
		return compileHeightFilter(f)
	case *AspectFilter:
		return compileAspectFilter(f)
	default:
		return nil, fmt.Errorf("unknown filter type: %T", flt)
	}
}

func tagMatches(tag, wanted string) bool {
	if tag == wanted {
		return true
	}
	return strings.HasPrefix(tag, wanted+"/")
}

func compileTagFilter(f *TagFilter) (CompiledFilter, error) {
	if len(f.Tags) == 0 {
		return nil, fmt.Errorf("tag filter without tags")
	}

	return func(img ImageFacts) bool {
		matchCount := 0
		for _, wanted := range f.Tags {
			found := false
			for _, t := range img.Tags {
				if tagMatches(t, wanted) {
					found = true
					break
				}
			}
			if found {
				matchCount++
			}
		}
		log.Logger.Debug().Int("found tags", matchCount).Str("mode", string(f.Op)).Msg("")

		switch f.Op {
		case SetAny:
			return matchCount > 0

		case SetAll:
			return matchCount == len(f.Tags)

		case SetNone:
			return matchCount == 0

		case SetOnly:
			if len(img.Tags) == 0 {
				return false
			}
			for _, t := range img.Tags {
				ok := false
				for _, wanted := range f.Tags {
					if tagMatches(t, wanted) {
						ok = true
						break
					}
				}
				if !ok {
					return false
				}
			}
			return true

		default:
			return false
		}
	}, nil
}
func parseDateRange(s string) (time.Time, time.Time, error) {
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date: %s", s)
	}

	y, _ := strconv.Atoi(parts[0])
	m, d := 1, 1

	if len(parts) > 1 {
		m, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		d, _ = strconv.Atoi(parts[2])
	}

	start := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)

	var end time.Time
	switch len(parts) {
	case 1:
		end = start.AddDate(1, 0, 0)
	case 2:
		end = start.AddDate(0, 1, 0)
	case 3:
		end = start.AddDate(0, 0, 1)
	}

	return start, end, nil
}
func compileDateFilter(f *DateFilter) (CompiledFilter, error) {
	start, end, err := parseDateRange(f.Date)
	if err != nil {
		return nil, err
	}

	return func(img ImageFacts) bool {
		if img.TakenAt == nil {
			return false
		}

		t := *img.TakenAt

		switch f.Op {
		case DateOn:
			return !t.Before(start) && t.Before(end)
		case DateBefore:
			return t.Before(start)
		case DateAfter:
			return !t.Before(end)
		default:
			return false
		}
	}, nil
}
func globToRegexp(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")

	for i := 0; i < len(glob); i++ {
		c := glob[i]

		switch c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}

	b.WriteString("$")
	return regexp.Compile(b.String())
}

func compileNameFilter(f *NameFilter) (CompiledFilter, error) {
	re, err := globToRegexp(f.Pattern)
	if err != nil {
		return nil, err
	}

	return func(img ImageFacts) bool {
		return re.MatchString(img.Filename)
	}, nil
}

func compileRatingFilter(f *RatingFilter) (CompiledFilter, error) {
	return func(img ImageFacts) bool {
		rating := 0
		if img.Rating != nil {
			rating = *img.Rating
		}
		switch f.Op {
		case RelationAbove:
			return rating > f.Value
		case RelationBelow:
			return rating < f.Value
		default:
			return false
		}
	}, nil
}

func compileWidthFilter(f *WidthFilter) (CompiledFilter, error) {
	return func(img ImageFacts) bool {
		width := int(img.Width)
		switch f.Op {
		case RelationAbove:
			return width > f.Value
		case RelationBelow:
			return width < f.Value
		default:
			return false
		}
	}, nil
}

func compileHeightFilter(f *HeightFilter) (CompiledFilter, error) {
	return func(img ImageFacts) bool {
		height := int(img.Height)
		switch f.Op {
		case RelationAbove:
			return height > f.Value
		case RelationBelow:
			return height < f.Value
		default:
			return false
		}
	}, nil
}

func compileAspectFilter(f *AspectFilter) (CompiledFilter, error) {
	return func(img ImageFacts) bool {
		h := int(img.Height)
		w := int(img.Width)
		if h == 0 {
			return false
		}
		r := float64(w) / float64(h)

		switch f.Op {
		case RelationAbove:
			return r > f.Value
		case RelationBelow:
			return r < f.Value
		default:
			return false
		}
	}, nil
}

func compilePathFilter(f *PathFilter) (CompiledFilter, error) {
	if len(f.Paths) == 0 {
		return nil, fmt.Errorf("path filter without paths")
	}

	return func(img ImageFacts) bool {
		match := false
		for _, p := range f.Paths {
			if strings.HasPrefix(img.Path, p) {
				match = true
				break
			}
		}

		switch f.Mode {
		case SetAny:
			return match
		case SetNone:
			return !match
		case SetOnly:
			return match // path egyértékű → any == only
		case SetAll:
			return match // path egyértékű → all == any
		default:
			return false
		}
	}, nil
}

func compileExtensionFilter(f *ExtensionFilter) (CompiledFilter, error) {
	set := make(map[string]struct{}, len(f.Extensions))
	for _, e := range f.Extensions {
		set[strings.ToLower(e)] = struct{}{}
	}

	return func(img ImageFacts) bool {
		_, ok := set[strings.ToLower(img.Ext)]

		switch f.Mode {
		case SetAny, SetOnly, SetAll:
			return ok
		case SetNone:
			return !ok
		default:
			return false
		}
	}, nil
}

func CompileGroupFilter(group RuleGroup) (CompiledFilter, error) {
	if len(group.Rules) == 0 {
		return nil, fmt.Errorf("empty filter group")
	}

	var preds []CompiledFilter

	for _, f := range group.Rules {
		p, err := compileAtomicFilter(f)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}

	switch group.Op {
	case OpAll:
		return func(img ImageFacts) bool {
			for _, p := range preds {
				if !p(img) {
					return false
				}
			}
			return true
		}, nil

	case OpAny:
		return func(img ImageFacts) bool {
			for _, p := range preds {
				if p(img) {
					return true
				}
			}
			return false
		}, nil

	default:
		return nil, fmt.Errorf("unknown filter group op: %s", group.Op)
	}
}
