package vault

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/phant0um/clipscribe/internal/media"
)

const maxFrontmatterLines = 200

// Find looks for an existing clipping of the same video in dirs (searched
// recursively) by reading only the frontmatter of each .md file.
// Missing directories are skipped.
func Find(dirs []string, p media.Platform, id string) (string, bool, error) {
	var found string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return fs.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			if matches(path, p, id) {
				found = path
				return fs.SkipAll
			}
			return nil
		})
		if err != nil {
			return "", false, err
		}
		if found != "" {
			return found, true, nil
		}
	}
	return "", false, nil
}

func matches(path string, p media.Platform, id string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return false
	}
	var platform, videoID string
	for n := 0; sc.Scan() && n < maxFrontmatterLines; n++ {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch strings.TrimSpace(key) {
		case "platform":
			platform = val
		case "video_id":
			videoID = val
		}
	}
	return platform == string(p) && videoID == id
}
