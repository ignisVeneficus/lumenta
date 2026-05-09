package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"time"

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
func StrToPtrUint64(s string) (*uint64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, err
	}
	u := uint64(v)
	return &u, nil
}
func StrToUint64(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	u := uint64(v)
	return u, nil
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

func SortByUint64Key[T any](items []T, key func(T) uint64) {
	sort.Slice(items, func(i, j int) bool {
		return key(items[i]) < key(items[j])
	})
}

func Reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func ComputeDailyHash() string {
	date := time.Now().UTC().Format("2006.01.02") // yyyy.mm.dd

	h := sha256.New()
	h.Write([]byte(date))

	return hex.EncodeToString(h.Sum(nil))
}

func ComputeYAMLHash(cfg any) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
func SameTime(a, b time.Time) bool {
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return diff < time.Second
}
