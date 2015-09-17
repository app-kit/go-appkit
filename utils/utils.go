package utils

import (
	"regexp"
	"strings"
)

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
