package app

import (
	"errors"
	"os"
	"path"

	"github.com/olebedev/config"

	kit "github.com/theduke/go-appkit"
)

type Config struct {
	*config.Config
}

// Ensure Config implements appkit.Config.
var _ kit.Config = (*Config)(nil)

func NewConfig(data interface{}) kit.Config {
	return &Config{
		Config: &config.Config{Root: data},
	}
}

func (c Config) GetData() interface{} {
	return c.Root
}

func (c Config) Get(path string) (kit.Config, error) {
	newC, err := c.Config.Get(path)
	if err != nil {
		return nil, err
	}

	return NewConfig(newC.Root), nil
}

func (c Config) ENV() string {
	return c.UString("ENV", "dev")
}

func (c Config) Debug() bool {
	return c.UBool("debug", false)
}

func (c Config) TmpDir() string {
	return c.UPath("tmpDir", "tmp")
}

func (c Config) DataDir() string {
	return c.UPath("dataDir", "data")
}

func (c Config) buildPath(p string) (string, error) {
	if len(p) < 1 {
		return "", errors.New("empty_path")
	}

	if p[0] == '/' {
		return p, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	root := c.UString("rootDir", wd)

	fullPath := path.Clean(path.Join(root, p))
	return fullPath, nil
}

func (c Config) Path(path string) (string, error) {
	p, err := c.String(path)
	if err != nil {
		return "", err
	}

	return c.buildPath(p)
}

func (c Config) UPath(path string, defaults ...string) string {
	p, err := c.Path(path)

	if err == nil && len(p) > 0 {
		return p
	}

	if len(defaults) == 0 {
		return ""
	}

	fullPath, err := c.buildPath(defaults[0])
	if err != nil {
		return ""
	}

	return fullPath
}
