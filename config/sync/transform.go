package sync

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ignisVeneficus/lumenta/utils"
)

func MergeMetadataConfig(base MetadataConfig, override MetadataConfig) MetadataConfig {

	out := MetadataConfig{
		Fields: make(map[string]MetadataFieldConfig),
	}

	for key, field := range base.Fields {
		out.Fields[key] = field
	}

	for key, field := range override.Fields {
		out.Fields[key] = field
	}

	return out
}

func (sc *SyncConfig) TransformBeforeValidation() error {
	ret := map[string]struct{}{}
	for _, e := range sc.Extensions {
		key := strings.ToLower(e)
		key = strings.TrimPrefix(e, ".")
		ret[key] = struct{}{}
	}
	sc.NormalizedExtensions = ret

	_ = sc.Exiftool.TransformBeforeValidation()

	return nil
}
func (sc *SyncConfig) TransformAfterValidation() error {
	// merge medata config with the hardoded metadata configs
	sc.MergedMetadata = MergeMetadataConfig(DefaultDBMetadataConfig(), sc.Metadata)

	metadataHash, err := utils.HashDataYAML(sc.Metadata)
	if err != nil {
		sc.MetadataHash = metadataHash
	}

	return nil
}

func ResolveExiftoolPath(path string) string {
	if path != "" {
		return path
	}
	if env := os.Getenv("EXIFTOOL_PATH"); env != "" {
		return env
	}

	if p, err := exec.LookPath("exiftool"); err == nil {
		return p
	}

	return ""
}

func (etC *ExiftoolConfig) TransformBeforeValidation() error {
	etC.ResolvedPath = ResolveExiftoolPath(etC.Path)
	return nil
}
