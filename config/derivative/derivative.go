package derivative

type DerivateSizeMode string

type DerivativesConfig []DerivativeConfig

var (
	DerivateSizeCrop   DerivateSizeMode = "crop"
	DerivateSizeResize DerivateSizeMode = "fit"
)

type DerivativeConfig struct {
	Name      string           `yaml:"name"`
	Postfix   string           `yaml:"postfix"`
	MaxWidth  int              `yaml:"max_width"`
	MaxHeight int              `yaml:"max_height"`
	Mode      DerivateSizeMode `yaml:"mode"` // crop | fit
}
