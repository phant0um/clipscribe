package transcript

import (
	"strings"
	"testing"

	"github.com/phant0um/clipscribe/internal/media"
)

// FuzzRenderMarkdown checks that uploader-controlled caption text can never
// leave active Markdown or HTML syntax in the note, and that metadata can
// never break out of the frontmatter.
func FuzzRenderMarkdown(f *testing.F) {
	f.Add("hi &lt;img src=x&gt; `$= x` #tag", "title", "desc")
	f.Add(`\[x\](https://e) \`+"`"+`$= x;//`+"` &#92;#t", "a\n---\nb: c", "[[x]]")
	f.Add("<%* code %>", "\"quoted\"", "\\")
	f.Fuzz(func(t *testing.T, cue, title, desc string) {
		segs, err := ParseVTT([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:02.000\n" + cue + "\n"))
		if err != nil {
			return
		}
		d := doc(media.X)
		d.Video.Title, d.Video.Description, d.Segments = title, desc, segs
		out := string(RenderMarkdown(d))
		if !strings.HasPrefix(out, "---\n") {
			t.Fatal("no frontmatter")
		}
		end := strings.Index(out[4:], "\n---\n")
		if end < 0 {
			t.Fatal("frontmatter not closed")
		}
		for _, line := range strings.Split(out[4:4+end], "\n") {
			if line != "author:" && line != "tags:" && !strings.HasPrefix(line, "  - ") && !strings.Contains(line, ": ") {
				t.Fatalf("frontmatter line broken: %q", line)
			}
		}
		for _, line := range strings.Split(out[4+end+5:], "\n") {
			if line == "" {
				continue
			}
			_, text, _ := strings.Cut(line, "] ")
			for i := 0; i < len(text); i++ {
				c := text[i]
				if strings.IndexByte("[]`#\\", c) < 0 {
					if c == '<' || c == '>' {
						t.Fatalf("raw %q in body: %q", c, text)
					}
					continue
				}
				if c != '\\' {
					t.Fatalf("unescaped %q in body: %q", c, text)
				}
				i++ // an escape: skip the escaped byte
				if i >= len(text) || strings.IndexByte("[]`#\\", text[i]) < 0 {
					t.Fatalf("stray backslash in body: %q", text)
				}
			}
		}
	})
}
