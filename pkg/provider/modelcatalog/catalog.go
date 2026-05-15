package modelcatalog

import (
	"embed"
	"fmt"
	"strings"

	"github.com/goccy/go-json"
)

//go:embed data/catalog.json
var catalogFS embed.FS

var catalogs map[string][]string

func init() {
	var err error
	catalogs, err = loadCatalogs()
	if err != nil {
		panic(err)
	}
}

func Load(platformID string) ([]string, error) {
	platformID = strings.TrimSpace(platformID)
	if platformID == "" {
		return nil, fmt.Errorf("model catalog platform id is empty")
	}

	raw, ok := catalogs[platformID]
	if !ok {
		return nil, fmt.Errorf("model catalog %q not found", platformID)
	}

	return append([]string(nil), raw...), nil
}

func MustLoad(platformID string) []string {
	models, err := Load(platformID)
	if err != nil {
		panic(err)
	}
	return models
}

func loadCatalogs() (map[string][]string, error) {
	data, err := catalogFS.ReadFile("data/catalog.json")
	if err != nil {
		return nil, fmt.Errorf("read model catalog file: %w", err)
	}

	var raw map[string][]string
	if err = json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode model catalog file: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("model catalog file is empty")
	}

	catalog := make(map[string][]string, len(raw))
	for platformID, models := range raw {
		normalizedPlatformID := strings.TrimSpace(platformID)
		if normalizedPlatformID == "" {
			return nil, fmt.Errorf("model catalog contains empty platform id")
		}
		if normalizedPlatformID != platformID {
			return nil, fmt.Errorf("model catalog platform id %q contains surrounding whitespace", platformID)
		}

		normalizedModels, normalizeErr := normalizeModels(normalizedPlatformID, models)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		catalog[normalizedPlatformID] = normalizedModels
	}
	return catalog, nil
}

func normalizeModels(platformID string, raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("model catalog %q is empty", platformID)
	}

	seen := make(map[string]struct{}, len(raw))
	models := make([]string, 0, len(raw))
	for idx, item := range raw {
		name := strings.TrimSpace(item)
		if name == "" {
			return nil, fmt.Errorf("model catalog %q contains empty model at index %d", platformID, idx)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("model catalog %q contains duplicate model %q", platformID, name)
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}
	return models, nil
}
