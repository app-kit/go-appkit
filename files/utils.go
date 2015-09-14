package files

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	kit "github.com/theduke/go-appkit"
)

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

func GetMimeType(path string) string {
	output, err := exec.Command("file", "-b", "--mime-type", path).Output()
	if err != nil {
		return ""
	}
	return string(output)
}

type ImageInfo struct {
	Width  int64
	Height int64
	Format string
}

func GetImageInfo(path string) (*ImageInfo, kit.ApiError) {
	output, err := exec.Command("identify", "-verbose", path).Output()
	if err != nil {
		return nil, kit.Error{
			Code:     "identify_failed",
			Message:  err.Error(),
			Internal: true,
		}
	}

	data := string(output)

	info := &ImageInfo{}

	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		switch strings.TrimSpace(parts[0]) {
		case "Format":
			rawFormat := strings.TrimSpace(parts[1])
			format := strings.Split(rawFormat, " ")
			if len(format) > 0 {
				info.Format = format[0]
			}
		case "Geometry":
			parts := strings.Split(strings.TrimSpace(parts[1]), "+")
			parts = strings.Split(strings.TrimSpace(parts[0]), "x")

			if len(parts) == 2 {
				width, err := strconv.ParseInt(parts[0], 10, 64)
				height, err2 := strconv.ParseInt(parts[1], 10, 64)
				if err == nil && err2 == nil {
					info.Width = width
					info.Height = height
				}
			}
		}

	}

	return info, nil
}
