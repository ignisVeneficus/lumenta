package routes

import (
	"fmt"
	"strings"
)

func getPath(pattern string, params ...string) string {
	p := strings.ReplaceAll(pattern, "%d", "%s")
	args := make([]any, len(params))
	for i, v := range params {
		args[i] = v
	}
	return fmt.Sprintf(p, args...)
}
