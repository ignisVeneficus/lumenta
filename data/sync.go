package data

type DirtyReason string

const (
	DirtyNewfile         DirtyReason = "new_file"
	DirtyHashChg         DirtyReason = "hash_changed"
	DirtyMetadataHashChg DirtyReason = "metadata_hash_changed"
	DirtySizeChg         DirtyReason = "size_changed"
	DirtyTimeChg         DirtyReason = "mtime_changed"
	DirtyForced          DirtyReason = "forced_refresh"
)

var (
	AllDirtyReason = []DirtyReason{
		DirtyNewfile,
		DirtyHashChg,
		DirtyMetadataHashChg,
		DirtySizeChg,
		DirtyTimeChg,
		DirtyForced,
	}
)
