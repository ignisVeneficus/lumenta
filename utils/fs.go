package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func SplitPath(path string) (string, string, string) {
	dir, name := filepath.Split(path)
	ext := filepath.Ext(name)
	return dir, name, ext
}

func ConcatLocalPath(dir string, name string, ext string) string {
	return filepath.Join(dir, name+ext)
}

func ConcatGlobalPath(root, dir, name, ext string) string {
	return filepath.Join(root, dir, name+ext)
}
func NormalizeExt(ext string) string {
	return strings.TrimPrefix(strings.ToLower(ext), ".")
}
