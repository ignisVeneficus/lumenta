package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

func PtrUint64(v uint64) *uint64 {
	return &v
}

func PtrString(v string) *string {
	return &v
}
func FromStringPtr(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}
func Abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func HashDataYAML(cfg any) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func SortByStringKey[T any](items []T, key func(T) string) {
	c := collate.New(
		language.Und,
		collate.IgnoreCase,
		collate.Numeric,
	)

	sort.Slice(items, func(i, j int) bool {
		return c.CompareString(
			key(items[i]),
			key(items[j]),
		) < 0
	})
}
