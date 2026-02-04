package filesystem

// Originals: read-only filesystem
// Derivatives: writable cache
type FilesystemConfig struct {
	Originals   string `yaml:"originals"`
	Derivatives string `yaml:"derivatives"`
}
