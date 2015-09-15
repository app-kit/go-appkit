package appkit

import (
	"regexp"
	"strings"
	"os"
	"fmt"
	"io/ioutil"

	. "github.com/theduke/go-appkit/error"
)

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

func StrIn(needle string, haystack []string) bool {
	for _, str := range haystack {
		if str == needle {
			return true
		}
	}

	return false
}

func FileExists(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if err == os.ErrNotExist {
			return false, nil
		} else {
			return false, err
		}
	}
	f.Close()

	return true, nil
}

func ReadFile(path string) ([]byte, Error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, AppError{
			Code: "file_open_error",
			Message: fmt.Sprintf("Could not open file at %v: %v", path, err),
			Errors: []error{err},
			Internal: true,
		}
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, AppError{
			Code: "file_read_error",
			Message: fmt.Sprintf("Could not read file at %v: %v", path, err),
			Errors: []error{err},
			Internal: true,
		}
	}

	return content, nil
}
