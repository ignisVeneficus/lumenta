package presentation

import gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"

type PresentationConfig struct {
	Templates string     `yaml:"templates"`
	Grid      GridConfig `yaml:"grid"`
}

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
