// Package vault names, writes and finds clippings on disk.
package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"clipscribe/internal/media"
)

const maxNameRunes = 80

// forbidden holds characters that break file systems or Obsidian links.
const forbidden = `/\:*?"<>|#^[]`

// FileName returns the clipping base name (without extension) for v,
// following spec 001 clarification C2.
func FileName(v media.Video) string {
	name := sanitize(v.Title)
	if v.Platform == media.X {
		name = "@" + sanitize(v.Author) + " — " + sanitize(v.Description)
	}
	name = cut(name, maxNameRunes)
	if strings.Trim(name, "@— ") == "" {
		return sanitize(v.ID)
	}
	return name
}

func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(forbidden, r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimLeft(s, ". ")
}

func cut(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	head := string(r[:limit+1])
	if i := strings.LastIndex(head, " "); i > 0 {
		return strings.TrimSpace(head[:i])
	}
	return string(r[:limit])
}

// Save writes data to dir/name+ext atomically without replacing an
// existing file. If the name is taken, " (id)" is appended.
func Save(dir, name, id, ext string, data []byte) (string, error) {
	for _, candidate := range []string{name, name + " (" + sanitize(id) + ")"} {
		path := filepath.Join(dir, candidate+ext)
		if filepath.Dir(path) != filepath.Clean(dir) {
			return "", fmt.Errorf("refusing to write outside %s", dir)
		}
		if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
			return path, writeAtomic(path, data)
		}
	}
	return "", fmt.Errorf("file names %q and %q (%s) already exist", name, name, id)
}

// Replace atomically overwrites an existing clipping (used by --force).
func Replace(path string, data []byte) error {
	return writeAtomic(path, data)
}

// writeAtomic writes to a temporary file in the same directory and renames
// it, so readers never see a partial clipping.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".clipscribe-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", tmp.Name(), err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
