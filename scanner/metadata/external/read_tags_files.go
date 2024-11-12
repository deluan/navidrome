package external

import (
	"github.com/mitchellh/mapstructure"
	"github.com/navidrome/navidrome/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ReadTagsFiles(filePaths []string) []Tags {
	var tags []Tags
	for _, filePath := range filePaths {
		tags = append(tags, readTagsFile(filePath)...)
	}
	return tags
}

func readTagsFile(filePath string) []Tags {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Error("Could not read tags file", "filePath", filePath, err)
		return nil
	}
	var raw map[string]any
	if err := yaml.Unmarshal(bytes, &raw); err != nil {
		log.Error("Tags file contains invalid YAML", "filePath", filePath, err)
		return nil
	}
	basePath := filepath.Dir(filePath)
	if tags, ok := parseSimpleMap(basePath, "", raw); ok {
		return []Tags{tags}
	}
	if tags, ok := parseFullEntry(basePath, "", raw); ok {
		return []Tags{tags}
	}
	var patternMap map[string]any
	if mapstructure.Decode(raw, &patternMap) != nil {
		log.Error("Tags file format not recognized", "filePath", filePath)
		return nil
	}
	if len(patternMap) == 0 {
		return nil
	}
	tagEntries := make([]Tags, 0, len(patternMap))
	for pattern, rawEntry := range patternMap {
		if tags, ok := parseSimpleMap(basePath, pattern, rawEntry); ok {
			tagEntries = append(tagEntries, tags)
			continue
		}
		if tags, ok := parseFullEntry(basePath, pattern, rawEntry); ok {
			tagEntries = append(tagEntries, tags)
			continue
		}
		log.Error("Tags file entry format not recognized", "filePath", filePath, "pattern", pattern)
		return nil
	}
	return tagEntries
}

func parseSimpleMap(basePath string, pattern string, raw any) (Tags, bool) {
	var simpleMapAny map[string]any
	if mapstructure.Decode(raw, &simpleMapAny) != nil {
		return Tags{}, false
	}
	if len(simpleMapAny) == 0 {
		return Tags{}, false
	}
	setTags, removeTags, ok := parseTagsMap(simpleMapAny)
	if !ok {
		return Tags{}, false
	}
	return Tags{
		BasePath:   basePath,
		Pattern:    pattern,
		SetTags:    setTags,
		RemoveTags: removeTags,
	}, true
}

type rawFullEntry struct {
	SetTags    map[string]any `mapstructure:"setTags"`
	RemoveTags []string       `mapstructure:"removeTags"`
}

func parseFullEntry(basePath string, pattern string, raw any) (Tags, bool) {
	var fullEntry rawFullEntry
	if mapstructure.Decode(raw, &fullEntry) != nil {
		return Tags{}, false
	}
	if len(fullEntry.SetTags) == 0 && len(fullEntry.RemoveTags) == 0 {
		return Tags{}, false
	}
	setTags, removeTags, ok := parseTagsMap(fullEntry.SetTags)
	if !ok {
		return Tags{}, false
	}
	for _, tag := range fullEntry.RemoveTags {
		removeTags[normalizeTagKey(tag)] = struct{}{}
	}
	return Tags{
		BasePath:   basePath,
		Pattern:    pattern,
		SetTags:    setTags,
		RemoveTags: removeTags,
	}, true
}

func parseTagsMap(rawMap map[string]any) (map[string]string, map[string]struct{}, bool) {
	if len(rawMap) == 0 {
		return nil, nil, false
	}
	parsedMap := make(map[string]string)
	removeMap := make(map[string]struct{})
	for key, rawValue := range rawMap {
		value, isNil, ok := parseTagValue(rawValue)
		if !ok {
			return nil, nil, false
		}
		if isNil {
			removeMap[normalizeTagKey(key)] = struct{}{}
		} else {
			parsedMap[normalizeTagKey(key)] = value
		}
	}
	return parsedMap, removeMap, true
}

func parseTagValue(raw any) (string, bool, bool) {
	switch raw := raw.(type) {
	case string:
		return raw, false, true
	case int:
		return strconv.Itoa(raw), false, true
	case bool:
		if raw {
			return "1", false, true
		}
		return "0", false, true
	case nil:
		return "", true, true
	}
	return "", false, false
}

func normalizeTagKey(key string) string {
	return strings.ToLower(key)
}
