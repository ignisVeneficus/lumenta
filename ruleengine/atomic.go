package ruleengine

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ErrEmptyFilter = errors.New("empty filter group")

type AlbumStruct map[uint64]struct{}
type AlbumsStruct map[uint64]AlbumStruct

type ImageFacts struct {
	Path     string
	Filename string
	Ext      string
	Root     string

	TakenAt *time.Time
	Rating  *int
	Width   uint32
	Height  uint32

	Tags []string

	// nill-> not given
	Albums AlbumsStruct
}
type RuleContext struct {
	RefAlbum *uint64
	NameMap  map[uint64]string
}

func (rc *RuleContext) AlbumName(value uint64) string {
	if rc == nil {
		return strconv.FormatUint(value, 10)
	}
	if name, ok := rc.NameMap[value]; ok {
		return name
	}
	return strconv.FormatUint(value, 10)
}
func (rc *RuleContext) ReflAbumName() string {
	if rc == nil {
		return ""
	}
	if rc.RefAlbum == nil {
		return ""
	}
	return rc.AlbumName(*rc.RefAlbum)
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

type CompiledFilter func(f ImageFacts, ruleContext *RuleContext) (TriState, RuleResult)

type CompiledGroupFilter func(f ImageFacts, ruleContext *RuleContext) (bool, GroupRuleResult)

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
		return compileAlbumFilter(f)
	case *NotInChildAlbumsFilter:
		return compileNotInChildAlbumsFilter(f)
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

func returnValue(ruleLog RuleResult, result TriState) (TriState, RuleResult) {
	ruleLog.Result = result
	return result, ruleLog
}
func returnBool(ruleLog RuleResult, result bool) (TriState, RuleResult) {
	ret := EvalResultFalse
	if result {
		ret = EvalResultTrue
	}
	ruleLog.Result = ret
	return ret, ruleLog
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
	name := fmt.Sprintf("tag:%s:%s", f.Op, strings.Join(f.Tags, ";"))
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "tag",
		Params: []RuleParam{
			CreateRuleParamStrings("tags", f.Tags),
		},
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		rr.Actual = append(rr.Actual, CreateRuleParamStrings("tags", img.Tags))
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
			return returnBool(rr, matchCount > 0)

		case SetAll:
			return returnBool(rr, matchCount == len(f.Tags))

		case SetNone:
			return returnBool(rr, matchCount == 0)

		case SetOnly:
			if len(img.Tags) == 0 {
				return returnBool(rr, false)
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
					return returnBool(rr, false)
				}
			}
			return returnBool(rr, true)
		default:
			return returnBool(rr, false)
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
	name := fmt.Sprintf("date:%s:%s", f.Op, f.Date)
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "date",
		Params: []RuleParam{
			CreateRuleParamString("date", f.Date),
		},
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		if img.TakenAt == nil {
			rr.Actual = append(rr.Actual,
				CreateRuleParamEmpty("date"))
			return returnValue(rr, EvalResultUnknow)
		}
		rr.Actual = append(rr.Actual,
			CreateRuleParamDate("date", (*img.TakenAt)))

		t := *img.TakenAt

		switch f.Op {
		case DateOn:
			return returnBool(rr, !t.Before(start) && t.Before(end))
		case DateBefore:
			return returnBool(rr, t.Before(start))
		case DateAfter:
			return returnBool(rr, !t.Before(end))
		default:
			return returnBool(rr, false)
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
	name := fmt.Sprintf("name::%s", f.Pattern)
	base := RuleResult{
		Name: name,
		Op:   "",
		Type: "name",
		Params: []RuleParam{
			CreateRuleParamString("pattern", f.Pattern),
		},
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		rr.Actual = append(rr.Actual,
			CreateRuleParamString("filename", img.Filename))
		return returnBool(rr, re.MatchString(img.Filename))
	}, nil
}

func compileRatingFilter(f *RatingFilter) (CompiledFilter, error) {
	name := fmt.Sprintf("rating:%s:%d", f.Op, f.Value)
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "rating",
		Params: []RuleParam{
			CreateRuleParamInt("rating", f.Value),
		},
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		rating := 0
		if img.Rating != nil {
			rating = *img.Rating
			rr.Actual = append(rr.Actual, CreateRuleParamInt("rating", rating))
		} else {
			rr.Actual = append(rr.Actual, CreateRuleParamEmpty("rating"))
			returnValue(rr, EvalResultUnknow)
		}
		switch f.Op {
		case RelationAbove:
			return returnBool(rr, rating > f.Value)
		case RelationBelow:
			return returnBool(rr, rating < f.Value)
		default:
			return returnBool(rr, false)
		}
	}, nil
}

func compileWidthFilter(f *WidthFilter) (CompiledFilter, error) {
	name := fmt.Sprintf("width:%s:%d", f.Op, f.Value)
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "width",
		Params: []RuleParam{
			CreateRuleParamInt("width", f.Value),
		},
	}
	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		width := int(img.Width)
		rr.Actual = append(rr.Actual, CreateRuleParamInt("width", width))
		switch f.Op {
		case RelationAbove:
			return returnBool(rr, width > f.Value)
		case RelationBelow:
			return returnBool(rr, width < f.Value)
		default:
			return returnBool(rr, false)
		}
	}, nil
}

func compileHeightFilter(f *HeightFilter) (CompiledFilter, error) {
	name := fmt.Sprintf("height:%s:%d", f.Op, f.Value)
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "height",
		Params: []RuleParam{
			CreateRuleParamInt("height", f.Value),
		},
	}
	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		height := int(img.Height)
		rr.Actual = append(rr.Actual, CreateRuleParamInt("height", height))
		switch f.Op {
		case RelationAbove:
			return returnBool(rr, height > f.Value)
		case RelationBelow:
			return returnBool(rr, height < f.Value)
		default:
			return returnBool(rr, false)
		}
	}, nil
}

func compileAspectFilter(f *AspectFilter) (CompiledFilter, error) {
	name := fmt.Sprintf("aspect:%s:%f", f.Op, f.Value)
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "aspect",
		Params: []RuleParam{
			CreateRuleParamFloat64("aspect", f.Value),
		},
	}
	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		h := int(img.Height)
		w := int(img.Width)
		if h == 0 {
			return returnValue(rr, EvalResultUnknow)
		}
		r := float64(w) / float64(h)
		rr.Actual = append(rr.Actual, CreateRuleParamFloat64("aspect", r))
		switch f.Op {
		case RelationAbove:
			return returnBool(rr, r > f.Value)
		case RelationBelow:
			return returnBool(rr, r < f.Value)
		default:
			return returnBool(rr, false)
		}
	}, nil
}

func compilePathFilter(f *PathFilter) (CompiledFilter, error) {
	if len(f.Paths) == 0 {
		return nil, fmt.Errorf("path filter without paths")
	}
	name := fmt.Sprintf("path:%s:%s:%s", f.Op, f.Root, strings.Join(f.Paths, ";"))
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "path",
		Params: []RuleParam{
			CreateRuleParamString("root", f.Root),
			CreateRuleParamStrings("path", f.Paths),
		},
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		rr.Actual = append(rr.Actual, CreateRuleParamString("root", img.Root),
			CreateRuleParamString("path", img.Path))
		match := false
		matchRoot := true
		if f.Root != "" {
			matchRoot = (img.Root == f.Root)
		}
		for _, p := range f.Paths {
			if strings.HasPrefix(img.Path, p) {
				match = matchRoot && true
				break
			}
		}

		switch f.Op {
		case SetAny:
			return returnBool(rr, match)
		case SetNone:
			return returnBool(rr, !match)
		case SetOnly:
			return returnBool(rr, match)
		case SetAll:
			return returnBool(rr, match)
		default:
			return returnBool(rr, false)
		}
	}, nil
}

func compileExtensionFilter(f *ExtensionFilter) (CompiledFilter, error) {
	set := make(map[string]struct{}, len(f.Extensions))
	for _, e := range f.Extensions {
		set[strings.ToLower(e)] = struct{}{}
	}
	name := fmt.Sprintf("extension:%s:%s", f.Op, strings.Join(f.Extensions, ";"))
	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "extension",
		Params: []RuleParam{
			CreateRuleParamStrings("extension", f.Extensions),
		},
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		rr.Actual = append(rr.Actual, CreateRuleParamString("extension", img.Ext))
		_, ok := set[strings.ToLower(img.Ext)]

		switch f.Op {
		case SetAny, SetOnly, SetAll:
			return returnBool(rr, ok)
		case SetNone:
			return returnBool(rr, !ok)
		default:
			return returnBool(rr, false)
		}
	}, nil
}

func compileAlbumFilter(f *AlbumFilter) (CompiledFilter, error) {
	if len(f.Albums) == 0 {
		return nil, fmt.Errorf("album filter without albums")
	}

	name := fmt.Sprintf("album:%s:%v:%t", f.Op, f.Albums, f.IncludeChildren)

	base := RuleResult{
		Name: name,
		Op:   string(f.Op),
		Type: "album",
		Params: []RuleParam{
			CreateRuleParamInts("albums", f.Albums),
			CreateRuleParamString("include_children", strconv.FormatBool(f.IncludeChildren)),
		},
	}
	targetSet := make(map[uint64]struct{}, len(f.Albums))
	for _, t := range f.Albums {
		targetSet[t] = struct{}{}
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base

		if img.Albums == nil {
			rr.Actual = append(rr.Actual, CreateRuleParamEmpty("albums"))
			return returnValue(rr, EvalResultUnknow)
		}

		var actual []string
		for k := range img.Albums {
			actual = append(actual, ruleContext.AlbumName(k))
		}
		rr.Actual = append(rr.Actual, CreateRuleParamStrings("image_albums", actual))

		// helper: target -> matched?
		matchedTargets := make(map[uint64]struct{}, len(f.Albums))
		matchedAlbumCount := 0

		for k, imgAlbum := range img.Albums {
			if len(imgAlbum) == 0 {
				continue
			}

			matched := false

			if f.IncludeChildren {
				for _, fa := range f.Albums {
					if _, ok := imgAlbum[fa]; ok {
						matched = true
						matchedTargets[fa] = struct{}{}
					}
				}
			} else {
				if _, ok := targetSet[k]; ok {
					matched = true
					matchedTargets[k] = struct{}{}
				}
			}

			if matched {
				matchedAlbumCount++
			}
		}

		targetCount := len(f.Albums)
		matchedCount := len(matchedTargets)

		anyMatched := matchedCount > 0
		allMatched := matchedCount == targetCount
		noneMatched := matchedCount == 0

		onlyMatched := len(img.Albums) > 0 && matchedAlbumCount == len(img.Albums)

		var result bool

		switch f.Op {
		case SetAny:
			result = anyMatched
		case SetAll:
			result = allMatched
		case SetNone:
			result = noneMatched
		case SetOnly:
			result = onlyMatched
		default:
			result = false
		}

		return returnBool(rr, result)
	}, nil
}

func compileNotInChildAlbumsFilter(f *NotInChildAlbumsFilter) (CompiledFilter, error) {
	name := "not_in_child_albums"

	base := RuleResult{
		Name: name,
		Op:   "",
		Type: "notinchildren",
	}

	return func(img ImageFacts, ruleContext *RuleContext) (TriState, RuleResult) {
		rr := base
		rr.Actual = append(rr.Actual, CreateRuleParamString("reference_album", ruleContext.ReflAbumName()))
		if ruleContext == nil {
			return returnValue(rr, EvalResultUnknow)
		}
		if ruleContext.RefAlbum == nil {
			return returnValue(rr, EvalResultUnknow)
		}
		if img.Albums == nil {
			return returnValue(rr, EvalResultTrue)
		}

		var actual []string
		for k := range img.Albums {
			actual = append(actual, ruleContext.AlbumName(k))
		}
		rr.Actual = append(rr.Actual, CreateRuleParamStrings("image_albums", actual))

		inChild := false
		for k, a := range img.Albums {
			if k == *ruleContext.RefAlbum {
				continue
			}
			_, ok := a[*ruleContext.RefAlbum]
			if ok {
				inChild = true
				break
			}
		}
		return returnBool(rr, !inChild)
	}, nil
}

func CompileGroupFilter(group RuleGroup, name string) (CompiledGroupFilter, error) {
	if len(group.Rules) == 0 {
		return nil, ErrEmptyFilter
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
		return func(img ImageFacts, ruleContext *RuleContext) (bool, GroupRuleResult) {
			ret := GroupRuleResult{
				Name: name,
				Op:   group.Op,
			}
			for _, p := range preds {
				ts, rr := p(img, ruleContext)
				ret.RuleResults = append(ret.RuleResults, rr)
				if !ts.Bool() {
					ret.Result = false
					return false, ret
				}
			}
			ret.Result = true
			return true, ret
		}, nil

	case OpAny:
		return func(img ImageFacts, ruleContext *RuleContext) (bool, GroupRuleResult) {
			ret := GroupRuleResult{
				Name: name,
				Op:   group.Op,
			}
			for _, p := range preds {
				ts, rr := p(img, ruleContext)
				ret.RuleResults = append(ret.RuleResults, rr)

				if ts.Bool() {
					ret.Result = true
					return true, ret
				}
			}
			ret.Result = false
			return false, ret
		}, nil

	default:
		return nil, fmt.Errorf("unknown filter group op: %s", group.Op)
	}
}
