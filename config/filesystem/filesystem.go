package filesystem

// Originals: read-only filesystem
// Derivatives: writable cache
type FilesystemConfig struct {
	Originals   RootConfigs `yaml:"originals"`
	Derivatives string      `yaml:"derivatives"`
}
type RootConfigs map[string]RootConfig
type RootConfig struct {
	Root     string   `yaml:"root"`
	Excluded []string `yaml:"excluded"`
}
