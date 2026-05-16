package routes

import (
	"fmt"
	"strings"
)

type ImageID uint64
type AlbumID uint64
type TagID uint64
type SyncRunID uint64
type SyncFileID uint64

func getPath(pattern string, params ...string) string {
	p := strings.ReplaceAll(pattern, "%d", "%s")
	args := make([]any, len(params))
	for i, v := range params {
		args[i] = v
	}
	return fmt.Sprintf(p, args...)
}
