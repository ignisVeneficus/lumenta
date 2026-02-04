package sync

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func (m *MetadataSourceConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		m.Ref = value.Value
		return nil
	}
	return fmt.Errorf("invalid metadata source")
}
