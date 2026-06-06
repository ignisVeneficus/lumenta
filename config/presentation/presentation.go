package presentation

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

type PresentationConfig struct {
	// userd defined templates
	Templates string `yaml:"templates"`
	// config of the image grid, aka masonry
	Grid GridConfig `yaml:"grid"`
	// define minimal access level for metadata
	MetadataACL          MetadataACLConfig `yaml:"metadata_acl"`
	ConvertedMetadataACL MetadataACL       `yaml:"-"`

	TagMeaningConfig *TagMeaningConfig `yaml:"tag_meaning"`
}
type TagMeaningConfig struct {
	MeaningMap TagMeaningMap `yaml:"map"`
	Threshold  int           `yaml:"threshold"`
}

type TagMeaningMap map[TagMeaning]TagDefinion

func (tmm TagMeaningMap) HasConfig(meaning TagMeaning) bool {
	_, ok := tmm[meaning]
	return ok
}
func (tmm TagMeaningMap) HasFeature(meaning TagMeaning, feature TagFeature) bool {
	tagDef, ok := tmm[meaning]
	if !ok {
		return false
	}
	return tagDef.HasFeature(feature)
}
func (tmm TagMeaningMap) GetMeaning(tagName string) TagMeaning {
	for k, cfg := range tmm {
		for _, t := range cfg.TagRoots {
			if tagName == t {
				return k
			}
		}
	}
	return ""
}

type TagDefinion struct {
	TagRoots    []string                `yaml:"roots"`
	Features    []TagFeature            `yaml:"features"`
	FeaturesMap map[TagFeature]struct{} `yaml:"-"`
}

func (td TagDefinion) HasFeature(feature TagFeature) bool {
	_, ok := td.FeaturesMap[feature]
	return ok
}

type TagFeature string

const (
	TagFeatureDaily   TagFeature = "daily"
	TagFeatureSimilar TagFeature = "similar"
)

var TagFeatureSet = map[TagFeature]struct{}{
	TagFeatureDaily:   {},
	TagFeatureSimilar: {},
}

type TagMeaning string

const (
	TagMeaningLocation TagMeaning = "location"
	TagMeaningSubject  TagMeaning = "subject"
	TagMeaningPeople   TagMeaning = "people"
	TagMeaningProject  TagMeaning = "project"
	TagMeaningMood     TagMeaning = "mood"
	TagMeaningTime     TagMeaning = "time"
	TagMeaningGear     TagMeaning = "gear"
)

var TagMeaningSet = map[TagMeaning]struct{}{
	TagMeaningLocation: {},
	TagMeaningSubject:  {},
	TagMeaningPeople:   {},
	TagMeaningProject:  {},
	TagMeaningMood:     {},
	TagMeaningTime:     {},
	TagMeaningGear:     {},
}

var TagMeaningList = []TagMeaning{
	TagMeaningLocation,
	TagMeaningSubject,
	TagMeaningPeople,
	TagMeaningProject,
	TagMeaningMood,
	TagMeaningTime,
	TagMeaningGear,
}

func IsValidTagMeaning(m TagMeaning) bool {
	_, ok := TagMeaningSet[m]
	return ok
}
func IsValidTagFeature(f TagFeature) bool {
	_, ok := TagFeatureSet[f]
	return ok
}

type MetadataACLConfig map[dbo.ACLRole][]string

type MetadataACL map[string]dbo.ACLRole

type GridConfig map[int]RoleConfig

type RoleConfig map[string]AspectConfig
type AspectConfig map[string]gridData.Span

func (g GridConfig) Span(width int, role gridData.Role, aspect gridData.Aspect) (gridData.Span, bool) {
	roleMap, ok := g[width]
	if !ok {
		return gridData.Span{}, false
	}
	aspectMap, ok := roleMap[string(role)]
	if !ok {
		return gridData.Span{}, false
	}
	v, ok := aspectMap[string(aspect)]
	return v, ok
}
