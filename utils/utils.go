package utils

import (
	"regexp"
	"strings"
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
