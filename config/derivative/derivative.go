package derivative

type DerivativeSizeMode string

type DerivativesConfig []DerivativeConfig

var (
	DerivativeSizeCrop   DerivativeSizeMode = "crop"
	DerivativeSizeResize DerivativeSizeMode = "fit"
)

type DerivativeConfig struct {
	Name      string             `yaml:"name"`
	Postfix   string             `yaml:"postfix"`
	MaxWidth  int                `yaml:"max_width"`
	MaxHeight int                `yaml:"max_height"`
	Mode      DerivativeSizeMode `yaml:"mode"` // crop | fit
}
