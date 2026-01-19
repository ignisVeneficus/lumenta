package utils

import "strings"

func SplitTagPath(path string) []string {
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
