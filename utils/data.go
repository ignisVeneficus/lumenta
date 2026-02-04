package utils

import (
	"crypto/sha256"
	"encoding/hex"

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
