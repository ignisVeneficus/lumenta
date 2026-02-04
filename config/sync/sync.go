package sync

import (
	"time"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/ruleengine"
)

type SyncConfig struct {
	Paths                []PathFilterConfig    `yaml:"paths"`
	Extensions           []string              `yaml:"extensions"` // ["jpg","jpeg","png","tif","tiff","heic"]
	Metadata             MetadataConfig        `yaml:"metadata"`
	Exiftool             ExiftoolConfig        `yaml:"exiftool"`
	Panorama             *ruleengine.RuleGroup `yaml:"panorama"`
	NormalizedExtensions map[string]struct{}   `yaml:"-"`
	MergedMetadata       MetadataConfig        `yaml:"-"`
	MetadataHash         string                `yaml:"-"`
}

type PathFilterConfig struct {
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
