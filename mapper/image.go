package mapper

import (
	"strings"

	"github.com/rs/zerolog/log"
)

func SplitTagPath(path string) []string {
	log.Logger.Debug().Str("path", path).Msg("splitPath")
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}

	return out
}
