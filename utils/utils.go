package utils

import (
	"regexp"
	"strings"

	"github.com/twinj/uuid"
)

func GetMapKey(rawData interface{}, key string) (interface{}, bool) {
	data, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, false
	}

	val, ok := data[key]
	if !ok {
		return nil, false
	}
	return val, true
}

func GetMapDictKey(rawData interface{}, key string) (map[string]interface{}, bool) {
	raw, ok := GetMapKey(rawData, key)
	if !ok {
		return nil, false
	}

	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, false
	}

	return m, true
}

func GetMapFloat64Key(rawData interface{}, key string) (float64, bool) {
	val, ok := GetMapKey(rawData, key)
	if !ok {
		return float64(0), false
	}

	f, ok := val.(float64)
	if !ok {
		return float64(0), false
	}

	return f, true
}

func GetMapStringKey(rawData interface{}, key string) string {
	val, ok := GetMapKey(rawData, key)
	if !ok {
		return ""
	}

	str, ok := val.(string)
	if !ok {
		return ""
	}

	return str
}

func StrIn(haystack []string, needle string) bool {
	if haystack == nil {
		return false
	}

	for _, str := range haystack {
		if str == needle {
			return true
		}
	}

	return false
}

func Canonicalize(str string) string {
	str = strings.TrimSpace(strings.ToLower(str))
	// Remove spaces.
	str = regexp.MustCompile("\\s+").ReplaceAllString(str, "_")

	// Replace german umlaute.
	str = strings.Replace(str, "ö", "oe", -1)
	str = strings.Replace(str, "ä", "ae", -1)
	str = strings.Replace(str, "ü", "ue", -1)
	str = strings.Replace(str, "ß", "ss", -1)

	str = regexp.MustCompile("[^a-z0-9\\._\\-]").ReplaceAllString(str, "")

	return str
}

func UUIDv4() string {
	return uuid.NewV4().String()
}
