package ruleengine

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type RuleGroupOp string

const (
	OpAll RuleGroupOp = "all" // AND
	OpAny RuleGroupOp = "any" // OR
)

type RuleGroup struct {
	Op    RuleGroupOp // all | any
	Rules []Rule
}

type Rule interface {
	FilterType() string
}

var filterRegistry = map[string]func() Rule{
	"tag":         func() Rule { return &TagFilter{} },
	"date":        func() Rule { return &DateFilter{} },
	"name":        func() Rule { return &NameFilter{} },
	"album":       func() Rule { return &AlbumFilter{} },
	"rating":      func() Rule { return &RatingFilter{} },
	"path":        func() Rule { return &PathFilter{} },
	"extension":   func() Rule { return &ExtensionFilter{} },
	"notchildren": func() Rule { return &NotInChildAlbumsFilter{} },
	"width":       func() Rule { return &WidthFilter{} },
	"height":      func() Rule { return &HeightFilter{} },
	"aspect":      func() Rule { return &AspectFilter{} },
}

func (g *RuleGroup) UnmarshalYAML(value *yaml.Node) error {
	log.Logger.Warn().Int("Line", value.Line).Msg("RuleGroup UnmarshalYAML called")
	var raw struct {
		Op    RuleGroupOp      `yaml:"op"`
		Rules []map[string]any `yaml:"rules"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	g.Op = raw.Op

	for i, rule := range raw.Rules {
		t, ok := rule["type"].(string)
		if !ok || t == "" {
			return fmt.Errorf("rules[%d]: missing type", i)
		}

		ctor, ok := filterRegistry[t]
		if !ok {
			return fmt.Errorf("rules[%d]: unknown filter type %q", i, t)
		}

		f := ctor()

		// YAML → map → JSON → struct
		b, err := json.Marshal(rule)
		if err != nil {
			return fmt.Errorf("rules[%d]: %w", i, err)
		}
		if err := json.Unmarshal(b, f); err != nil {
			return fmt.Errorf("rules[%d]: %w", i, err)
		}

		g.Rules = append(g.Rules, f)
	}

	return nil
}

func (g *RuleGroup) UnmarshalJSON(data []byte) error {
	var raw struct {
		Op    RuleGroupOp      `json:"op"`
		Rules []map[string]any `json:"rules"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	g.Op = raw.Op

	for i, rule := range raw.Rules {
		t, ok := rule["type"].(string)
		if !ok || t == "" {
			return fmt.Errorf("rules[%d]: missing type", i)
		}

		ctor, ok := filterRegistry[t]
		if !ok {
			return fmt.Errorf("rules[%d]: unknown filter type %q", i, t)
		}

		f := ctor()
		b, _ := json.Marshal(rule)
		if err := json.Unmarshal(b, f); err != nil {
			return fmt.Errorf("rules[%d]: %w", i, err)
		}

		g.Rules = append(g.Rules, f)
	}

	return nil
}

func (g RuleGroup) MarshalJSON() ([]byte, error) {
	type alias struct {
		Op      RuleGroupOp `json:"op"`
		Filters []any       `json:"rules"`
	}
	out := alias{
		Op: g.Op,
	}
	for _, f := range g.Rules {
		out.Filters = append(out.Filters, f)
	}
	return json.Marshal(out)
}

type SetMode string

const (
	SetAll  SetMode = "all"  // all of them
	SetAny  SetMode = "any"  // one of them
	SetNone SetMode = "none" // none of them
	SetOnly SetMode = "only" // only them
)

type DateOp string

const (
	DateOn     DateOp = "on"
	DateBefore DateOp = "before"
	DateAfter  DateOp = "after"
)

type RelationOp string

const (
	RelationBelow RelationOp = "<"
	RelationAbove RelationOp = ">"
)

type TagFilter struct {
	Type string   `json:"type" yaml:"type"`
	Op   SetMode  `json:"op" yaml:"op"`
	Tags []string `json:"tags" yaml:"tags"`
}

func (TagFilter) FilterType() string { return "tag" }

type DateFilter struct {
	Type string `json:"type" yaml:"type"` // "date"
	Op   DateOp `json:"op" yaml:"op"`
	Date string `json:"date" yaml:"date"` // yyyy[.mm[.dd]]
}

func (DateFilter) FilterType() string { return "date" }

type NameFilter struct {
	Type    string `json:"type" yaml:"type"` // "name"
	Pattern string `json:"pattern" yaml:"pattern"`
}

func (NameFilter) FilterType() string { return "name" }

type RatingFilter struct {
	Type  string     `json:"type" yaml:"type"` // "rating"
	Op    RelationOp `json:"op" yaml:"op"`
	Value int        `json:"value" yaml:"value"`
}

func (RatingFilter) FilterType() string { return "rating" }

type WidthFilter struct {
	Type  string     `json:"type" yaml:"type"` // "width"
	Op    RelationOp `json:"op" yaml:"op"`
	Value int        `json:"value" yaml:"value"`
}

func (WidthFilter) FilterType() string { return "width" }

type HeightFilter struct {
	Type  string     `json:"type" yaml:"type"` // "height"
	Op    RelationOp `json:"op" yaml:"op"`
	Value int        `json:"value" yaml:"value"`
}

func (HeightFilter) FilterType() string { return "height" }

type AspectFilter struct {
	Type  string     `json:"type" yaml:"type"` // "aspect"
	Op    RelationOp `json:"op" yaml:"op"`
	Value float64    `json:"value" yaml:"value"`
}

func (AspectFilter) FilterType() string { return "aspect" }

type PathFilter struct {
	Type  string   `json:"type" yaml:"type"` // "path"
	Mode  SetMode  `json:"mode" yaml:"mode"`
	Paths []string `json:"paths" yaml:"paths"` // glob / prefix
}

func (PathFilter) FilterType() string { return "path" }

type ExtensionFilter struct {
	Type       string   `json:"type" yaml:"type"` // "extension"
	Mode       SetMode  `json:"mode" yaml:"mode"`
	Extensions []string `json:"extensions" yaml:"extensions"`
}

func (ExtensionFilter) FilterType() string { return "extension" }

type AlbumFilter struct {
	Type            string   `json:"type" yaml:"type"` // "album"
	Mode            SetMode  `json:"mode" yaml:"mode"`
	Albums          []uint64 `json:"albums" yaml:"albums"`
	IncludeChildren bool     `json:"include_children" yaml:"include_children"`
	ExcludeChildren bool     `json:"exclude_children" yaml:"exclude_children"`
}

func (AlbumFilter) FilterType() string { return "album" }

type NotInChildAlbumsFilter struct {
	Type string `json:"type" yaml:"type"` // "not_in_child_albums"
}

func (NotInChildAlbumsFilter) FilterType() string { return "notchildren" }
