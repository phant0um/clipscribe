package vault

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/phant0um/clipscribe/internal/media"
)

// FuzzFileName checks that uploader-controlled title, author and
// description can never produce a path, a hidden file or an unsafe name.
func FuzzFileName(f *testing.F) {
	f.Add("The science of spiciness", "poteto", "here's how", true)
	f.Add("../../etc/passwd", "a/b", ".hidden", false)
	f.Add("‮evil​", "x\x00y", "\n\r\t", true)
	f.Add("..", "..", "..", true)
	f.Fuzz(func(t *testing.T, title, author, desc string, isX bool) {
		v := media.Video{Platform: media.YouTube, ID: "qD0_yWgifDM", Title: title, Author: author, Description: desc}
		if isX {
			v.Platform, v.ID = media.X, "2102050467505430555"
		}
		name := FileName(v)
		switch {
		case name == "", name == ".", name == "..", strings.HasPrefix(name, "."):
			t.Fatalf("unsafe name %q", name)
		case !utf8.ValidString(name):
			t.Fatalf("invalid UTF-8 %q", name)
		case utf8.RuneCountInString(name) > maxNameRunes:
			t.Fatalf("name too long: %d runes", utf8.RuneCountInString(name))
		case strings.ContainsAny(name, forbidden+"/\x00"):
			t.Fatalf("forbidden character in %q", name)
		}
		for _, r := range name {
			if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
				t.Fatalf("control or format character %U in %q", r, name)
			}
		}
	})
}
