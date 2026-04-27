package i18n

import (
	"fmt"
	"strings"
	"sync"
)

type Service struct {
	langs map[string]Translations
}

var (
	instance *Service
	once     sync.Once
	initErr  error
)

func Init() (*Service, error) {
	once.Do(func() {
		langs, err := LoadAllLangs()
		if err != nil {
			initErr = err
			return
		}

		instance = &Service{
			langs: langs,
		}
	})

	return instance, initErr
}
func Get() *Service {
	return instance
}

func (s *Service) T(lang, key string, params map[string]any) string {
	return s.t(lang, key, params, false)
}

func (s *Service) t(lang, key string, params map[string]any, isFallback bool) string {
	for i, l := range s.fallbackChain(lang) {
		if langMap, ok := s.langs[l]; ok {
			if str, ok := langMap[key]; ok {
				prefix := ""
				if i > 0 {
					prefix = "[" + l + "]"
				}
				return prefix + interpolate(str, params)
			}
		}
	}
	return "[" + key + "]"
}

func interpolate(s string, params map[string]any) string {
	for k, v := range params {
		s = strings.ReplaceAll(s, "{"+k+"}", fmt.Sprint(v))
	}
	return s
}

func (s *Service) fallbackChain(local string) []string {
	parts := strings.Split(local, "-")
	var chain []string
	// full
	chain = append(chain, local)
	// hu-HU → hu
	if len(parts) > 1 {
		chain = append(chain, parts[0])
	}
	// default
	chain = append(chain, "en")
	return chain
}

func (s *Service) ExtractKeys(local string, selectors []string) map[string]string {
	result := make(map[string]string)
	langMap := s.langs[local]

	for _, sel := range selectors {
		// prefix: "map.*"
		if strings.HasSuffix(sel, ".*") {
			prefix := strings.TrimSuffix(sel, ".*") + "."

			for k, v := range langMap {
				if strings.HasPrefix(k, prefix) {
					result[k] = v
				}
			}
			continue
		}

		if v, ok := langMap[sel]; ok {
			result[sel] = v
		} else {
			result[sel] = "[" + sel + "]"
		}
	}

	return result
}
