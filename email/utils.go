package email

import (
	"strings"
	
	"github.com/alexcesaro/quotedprintable"
  "github.com/famz/RFC2047"
)

func DecodeStr(str string) string {
    str = RFC2047.Decode(str)

    reader := quotedprintable.NewReader(strings.NewReader(str))
    buffer := make([]byte, len(str) * 10)
    n, err := reader.Read(buffer)

    if err != nil && err.Error() != "EOF" {
        // TODO: fix the few but still present decoding issues.
    } else {
        str = string(buffer[:n])
    }

    return str
}

func parseFields(raw string) map[string]string {
    data := make(map[string]string)

    for _, line := range strings.Split(raw, "\n") {
        if pos := strings.Index(line, ":"); pos != -1 {
            data[strings.TrimSpace(line[:pos])] = strings.TrimSpace(line[pos+1:])
        }
    }

    return data
}


func StrBefore(str, separator string) string {
    index := strings.Index(str, separator)
    if index == -1 {
        return str
    } else {
        return str[:index]
    }
}

func StrAfter(str, separator string) string {
    index := strings.Index(str, separator)
    if index == -1 {
        return str
    } else {
        return str[index+1:]
    }
}

func ExtractAttributes(str string) map[string]string {
    attrs := make(map[string]string)

    for _, part := range strings.Split(str, ";") {
        parts := strings.Split(part, "=")
        if len(parts) == 2 {
            attrs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
        }
    }

    return attrs
}

func ExtractAttr(str, attr string) string {
    attrs := ExtractAttributes(str)
    return attrs[attr]
}
