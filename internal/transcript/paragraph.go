package transcript

import (
	"strings"
	"time"
)

// Paragraph grouping limits (spec 001, clarification C3).
const (
	maxPause       = 1500 * time.Millisecond
	maxParagraph   = 45 * time.Second
	minForSentence = 20 * time.Second
)

// Paragraph is a group of segments opened by a timestamp.
type Paragraph struct {
	Start time.Duration
	Text  string
}

// Paragraphs groups segments. A new paragraph starts at the first of:
// a pause longer than maxPause; the paragraph reaching maxParagraph; or the
// previous segment ending a sentence after the paragraph passed minForSentence.
func Paragraphs(segs []Segment) []Paragraph {
	var out []Paragraph
	var words []string
	var start time.Duration
	for i, s := range segs {
		if i > 0 {
			prev := segs[i-1]
			length := prev.End - start
			if s.Start-prev.End > maxPause || length >= maxParagraph ||
				(endsSentence(prev.Text) && length > minForSentence) {
				out = append(out, Paragraph{Start: start, Text: strings.Join(words, " ")})
				words = nil
			}
		}
		if words == nil {
			start = s.Start
		}
		words = append(words, s.Text)
	}
	if words != nil {
		out = append(out, Paragraph{Start: start, Text: strings.Join(words, " ")})
	}
	return out
}

func endsSentence(s string) bool {
	return strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "!")
}
