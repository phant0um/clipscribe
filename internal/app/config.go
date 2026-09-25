package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config is read from ~/.config/clipscribe/config.json. Flags override it.
type Config struct {
	OutDir    string   `json:"out_dir"`
	DedupDirs []string `json:"dedup_dirs"`
	ModelPath string   `json:"model_path"`
}

const defaultModel = "~/.local/share/clipscribe/models/ggml-large-v3-turbo.bin"

// DefaultConfigPath returns ~/.config/clipscribe/config.json.
func DefaultConfigPath() string {
	return expand("~/.config/clipscribe/config.json")
}

// LoadConfig reads path. A missing file yields the defaults.
func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return c, fmt.Errorf("read config: %w", err)
	default:
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			return c, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	if c.ModelPath == "" {
		c.ModelPath = defaultModel
	}
	c.OutDir, c.ModelPath = expand(c.OutDir), expand(c.ModelPath)
	for i := range c.DedupDirs {
		c.DedupDirs[i] = expand(c.DedupDirs[i])
	}
	if len(c.DedupDirs) == 0 && c.OutDir != "" {
		c.DedupDirs = []string{c.OutDir}
	}
	return c, nil
}

func expand(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return p
}
