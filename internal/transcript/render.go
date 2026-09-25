package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/phant0um/clipscribe/internal/media"
)

const descriptionLimit = 160

// Doc is everything needed to render one clipping.
type Doc struct {
	Video       media.Video
	Lang        string
	Transcriber string // "captions-manual" or "whisper-large-v3-turbo"
	AudioSHA256 string // empty when the text came from captions
	Created     time.Time
	Segments    []Segment
}

// RenderMarkdown renders the clipping in the Obsidian Web Clipper schema
// plus the transcript fields defined in spec 001.
func RenderMarkdown(d Doc) []byte {
	var b bytes.Buffer
	v := d.Video
	author := "[[" + v.Author + "]]"
	if v.Platform == media.X {
		author = "[[@" + v.Author + "]]"
	}
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %s\n", quote(v.Title))
	fmt.Fprintf(&b, "source: %s\n", quote(v.URL))
	fmt.Fprintf(&b, "author:\n  - %s\n", quote(author))
	if !v.Published.IsZero() {
		fmt.Fprintf(&b, "published: %s\n", v.Published.Format(time.DateOnly))
	}
	fmt.Fprintf(&b, "created: %s\n", d.Created.Format(time.DateOnly))
	fmt.Fprintf(&b, "description: %s\n", quote(description(d)))
	fmt.Fprintf(&b, "platform: %s\n", quote(string(v.Platform)))
	fmt.Fprintf(&b, "video_id: %s\n", quote(v.ID))
	fmt.Fprintf(&b, "duration: %s\n", quote(clock(v.Duration)))
	fmt.Fprintf(&b, "lang: %s\n", quote(d.Lang))
	fmt.Fprintf(&b, "transcriber: %s\n", quote(d.Transcriber))
	if d.AudioSHA256 != "" {
		fmt.Fprintf(&b, "audio_sha256: %s\n", quote(d.AudioSHA256))
	}
	b.WriteString("tags:\n  - \"clippings\"\n  - \"transcript\"\n---\n")
	for i, p := range Paragraphs(d.Segments) {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s %s\n", stamp(v, p.Start), p.Text)
	}
	return b.Bytes()
}

// RenderSRT renders segments as a SubRip file.
func RenderSRT(segs []Segment) []byte {
	var b bytes.Buffer
	for i, s := range segs {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n", i+1, srtTime(s.Start), srtTime(s.End), s.Text)
	}
	return b.Bytes()
}

// RenderText renders paragraphs as plain text without timestamps.
func RenderText(segs []Segment) []byte {
	var parts []string
	for _, p := range Paragraphs(segs) {
		parts = append(parts, p.Text)
	}
	return []byte(strings.Join(parts, "\n\n") + "\n")
}

func stamp(v media.Video, at time.Duration) string {
	c := clock(at)
	if v.Platform == media.YouTube {
		return fmt.Sprintf("[%s](https://youtu.be/%s?t=%d)", c, v.ID, int(at.Seconds()))
	}
	return "[" + c + "]"
}

func description(d Doc) string {
	s := d.Video.Description
	if strings.TrimSpace(s) == "" {
		var parts []string
		for _, seg := range d.Segments {
			parts = append(parts, seg.Text)
		}
		s = strings.Join(parts, " ")
	}
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > descriptionLimit {
		s = strings.TrimSpace(string(r[:descriptionLimit])) + "…"
	}
	return s
}

// quote returns s as a JSON string, which is a valid YAML double-quoted
// scalar: quotes, backslashes and newlines are escaped.
func quote(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // encoding a string cannot fail
	return strings.TrimSuffix(b.String(), "\n")
}

func clock(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s%3600/60, s%60)
}

func srtTime(d time.Duration) string {
	ms := d.Milliseconds()
	return fmt.Sprintf("%s,%03d", clock(d), ms%1000)
}
