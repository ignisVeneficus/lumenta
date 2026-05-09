package sync

import (
	"time"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/ruleengine"
)

type StepName string

const (
	//WorkerFS WorkerName ="filesystem"
	StepMetadata StepName = "metadata"
	StepHash     StepName = "hash"
)

var ValidStepName = map[StepName]struct{}{
	StepMetadata: {},
	StepHash:     {},
}

type SyncConfig struct {
	Paths                []PathFilterConfig      `yaml:"paths"`
	Extensions           []string                `yaml:"extensions"` // ["jpg","jpeg","png","tif","tiff","heic"]
	Metadata             MetadataConfig          `yaml:"metadata"`
	Exiftool             ExiftoolConfig          `yaml:"exiftool"`
	Panorama             *ruleengine.RuleGroup   `yaml:"panorama"`
	ACLRules             ACLRules                `yaml:"ACL_rules"`
	ACLOverride          bool                    `yaml:"override_ACL_rules"`
	Pipeline             map[StepName]StepConfig `yaml:"pipeline"`
	NormalizedExtensions map[string]struct{}     `yaml:"-"`
	MergedMetadata       MetadataConfig          `yaml:"-"`
	MetadataHash         string                  `yaml:"-"`
}

type PathFilterConfig struct {
	Root    string               `yaml:"root"`
	Path    string               `yaml:"path"` // real FS path (prefix)
	Filters ruleengine.RuleGroup `yaml:"filters"`
}

type MetadataConfig struct {
	Fields map[string]MetadataFieldConfig `yaml:"fields"`
}

type MetadataFieldConfig struct {
	Sources []MetadataSourceConfig `yaml:"sources"`
	Type    data.MetadataType      `yaml:"type,omitempty"`
	Unit    string                 `yaml:"unit,omnitempty"`
}

type MetadataSourceConfig struct {
	Ref string `yaml:"ref"`
}

type ExiftoolConfig struct {
	Path         string        `yaml:"path"`    // pl: "/usr/bin/exiftool"
	Timeout      time.Duration `yaml:"timeout"` // opcionális
	ResolvedPath string        `yaml:"-"`
}

type ACLRules []ACLRule

type ACLRule struct {
	Role  dbo.ACLRole             `yaml:"role"`
	User  *string                 `yaml:"user"`
	Rules []*ruleengine.RuleGroup `yaml:"rules"`
}
type StepConfig struct {
	Workers uint16 `yaml:"workers"`
}
