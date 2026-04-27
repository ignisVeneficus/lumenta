package i18n

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var langFS embed.FS

type Translations map[string]string

func LoadAllLangs() (map[string]Translations, error) {
	langs := map[string]Translations{}

	entries, err := langFS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".yaml")

		b, err := langFS.ReadFile(e.Name())
		if err != nil {
			return nil, err
		}

		var data map[string]any
		if err := yaml.Unmarshal(b, &data); err != nil {
			return nil, err
		}
		flat := map[string]string{}
		flatten("", data, flat)

		langs[name] = flat
	}

	return langs, nil
}

func flatten(prefix string, input map[string]any, out map[string]string) {
	for k, v := range input {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			out[key] = val

		case map[string]any:
			flatten(key, val, out)

		default:
			// opcionális: log warning
			// pl. szám, bool stb
		}
	}
}
