package ruleengine

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type FilterGroupOp string

const (
	OpAll FilterGroupOp = "all" // AND
	OpAny FilterGroupOp = "any" // OR
)

type FilterGroup struct {
	Op      FilterGroupOp // all | any
	Filters []Filter      // legalább 1
}

type Filter interface {
	FilterType() string
}

var filterRegistry = map[string]func() Filter{
	"tag":         func() Filter { return &TagFilter{} },
	"date":        func() Filter { return &DateFilter{} },
	"name":        func() Filter { return &NameFilter{} },
	"album":       func() Filter { return &AlbumFilter{} },
	"rating":      func() Filter { return &RatingFilter{} },
	"path":        func() Filter { return &PathFilter{} },
	"extension":   func() Filter { return &ExtensionFilter{} },
	"notchildren": func() Filter { return &NotInChildAlbumsFilter{} },
}

func (g *FilterGroup) UnmarshalYAML(value *yaml.Node) error {
	// 1️⃣ ideiglenes, YAML-barát struktúra
	var raw struct {
		Op    FilterGroupOp    `yaml:"op"`
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

		g.Filters = append(g.Filters, f)
	}

	return nil
}

func (g *FilterGroup) UnmarshalJSON(data []byte) error {
	var raw struct {
		Op    FilterGroupOp    `json:"op"`
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

		g.Filters = append(g.Filters, f)
	}

	return nil
}

func (g FilterGroup) MarshalJSON() ([]byte, error) {
	type alias struct {
		Op      FilterGroupOp `json:"op"`
		Filters []any         `json:"rules"`
	}
	out := alias{
		Op: g.Op,
	}
	for _, f := range g.Filters {
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

type RatingOp string

const (
	RatingBelow RatingOp = "<"
	RatingAbove RatingOp = ">"
)

type TagFilter struct {
	Type string   `json:"type" yaml:"type"`
	Mode SetMode  `json:"mode" yaml:"mode"`
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
	Type  string   `json:"type" yaml:"type"` // "rating"
	Op    RatingOp `json:"op" yaml:"op"`
	Value int      `json:"value" yaml:"value"`
}

func (RatingFilter) FilterType() string { return "rating" }

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
