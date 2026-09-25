package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phant0um/clipscribe/internal/media"
)

func TestFileName(t *testing.T) {
	cases := []struct {
		name string
		v    media.Video
		want string
	}{
		{"youtube title", media.Video{Platform: media.YouTube, ID: "id1", Title: "The science of spiciness - Rose Eveleth"},
			"The science of spiciness - Rose Eveleth"},
		{"forbidden characters", media.Video{Platform: media.YouTube, ID: "id1", Title: `a/b\c:d*e?f"g<h>i|j#k^l[m]n`},
			"a b c d e f g h i j k l m n"},
		{"control chars and spaces", media.Video{Platform: media.YouTube, ID: "id1", Title: "  a\tb\n\x00c   d  "},
			"a b c d"},
		{"no hidden or relative names", media.Video{Platform: media.YouTube, ID: "id1", Title: "../../etc/passwd"},
			"etc passwd"},
		{"emoji kept", media.Video{Platform: media.YouTube, ID: "id1", Title: "Olá 🚀 mundo"}, "Olá 🚀 mundo"},
		{"cut at word boundary", media.Video{Platform: media.YouTube, ID: "id1", Title: strings.Repeat("palavra ", 20)},
			strings.TrimSpace(strings.Repeat("palavra ", 10))},
		{"x uses handle and post text", media.Video{Platform: media.X, ID: "2102", Author: "poteto",
			Title: "lauren - here's how...", Description: "here's how i shipped 2,500 PRs last month"},
			"@poteto — here's how i shipped 2,500 PRs last month"},
		{"empty title falls back to id", media.Video{Platform: media.YouTube, ID: "qD0_yWgifDM", Title: "???"}, "qD0_yWgifDM"},
	}
	for _, c := range cases {
		if got := FileName(c.v); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if n := len([]rune(FileName(c.v))); n > 80 {
			t.Errorf("%s: %d runes", c.name, n)
		}
	}
}

func TestSaveWritesAndAvoidsCollision(t *testing.T) {
	dir := t.TempDir()
	p1, err := Save(dir, "Título", "id1", ".md", []byte("one"))
	if err != nil || p1 != filepath.Join(dir, "Título.md") {
		t.Fatalf("p1=%q err=%v", p1, err)
	}
	p2, err := Save(dir, "Título", "id2", ".md", []byte("two"))
	if err != nil || p2 != filepath.Join(dir, "Título (id2).md") {
		t.Fatalf("p2=%q err=%v", p2, err)
	}
	if b, _ := os.ReadFile(p1); string(b) != "one" {
		t.Errorf("first file overwritten: %q", b)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("leftover temp files: %v", entries)
	}
}

func TestReplaceOverwritesAtomically(t *testing.T) {
	dir := t.TempDir()
	p, _ := Save(dir, "x", "id", ".md", []byte("old"))
	if err := Replace(p, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "new" {
		t.Errorf("got %q", b)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("leftover temp files: %v", entries)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindByFrontmatter(t *testing.T) {
	inbox, archive := t.TempDir(), t.TempDir()
	write(t, filepath.Join(inbox, "renamed by user.md"), "---\ntitle: \"a\"\nplatform: \"youtube\"\nvideo_id: \"qD0_yWgifDM\"\n---\nbody\n")
	write(t, filepath.Join(archive, "2026-09-01", "old.md"), "---\nplatform: x\nvideo_id: '2102'\n---\n")
	write(t, filepath.Join(inbox, "body-only.md"), "---\ntitle: z\n---\nvideo_id: \"zzz\"\nplatform: \"youtube\"\n")
	write(t, filepath.Join(inbox, "other.md"), "---\nplatform: \"youtube\"\nvideo_id: \"other\"\n---\n")

	dirs := []string{inbox, archive, filepath.Join(inbox, "missing")}
	if p, ok, err := Find(dirs, media.YouTube, "qD0_yWgifDM"); err != nil || !ok || filepath.Base(p) != "renamed by user.md" {
		t.Errorf("youtube: %q %v %v", p, ok, err)
	}
	if p, ok, _ := Find(dirs, media.X, "2102"); !ok || filepath.Base(p) != "old.md" {
		t.Errorf("x in archive subdir: %q %v", p, ok)
	}
	if _, ok, _ := Find(dirs, media.YouTube, "zzz"); ok {
		t.Error("matched keys outside frontmatter")
	}
	if _, ok, _ := Find(dirs, media.X, "qD0_yWgifDM"); ok {
		t.Error("matched id on wrong platform")
	}
}
