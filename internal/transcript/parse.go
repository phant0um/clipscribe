// Package transcript turns captions or Whisper output into segments,
// groups them into paragraphs and renders the result.
package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// Segment is a span of speech with its text.
type Segment struct {
	Start, End time.Duration
	Text       string
}

var (
	cueTiming = regexp.MustCompile(`^((?:\d+:)?\d{2}:\d{2}\.\d{3})\s+-->\s+((?:\d+:)?\d{2}:\d{2}\.\d{3})`)
	vttTag    = regexp.MustCompile(`<[^>]*>`)
	spaces    = regexp.MustCompile(`\s+`)
)

// ParseVTT parses a WebVTT caption file.
func ParseVTT(data []byte) ([]Segment, error) {
	if !bytes.HasPrefix(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), []byte("WEBVTT")) {
		return nil, errors.New("not a WebVTT file")
	}
	var segs []Segment
	var cur *Segment
	var text []string
	flush := func() {
		if cur != nil {
			if t := cleanText(strings.Join(text, " ")); t != "" {
				cur.Text = t
				segs = append(segs, *cur)
			}
		}
		cur, text = nil, nil
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if m := cueTiming.FindStringSubmatch(line); m != nil {
			flush()
			start, err1 := parseVTTTime(m[1])
			end, err2 := parseVTTTime(m[2])
			if err := errors.Join(err1, err2); err != nil {
				return nil, err
			}
			cur = &Segment{Start: start, End: end}
			continue
		}
		if line == "" {
			flush()
			continue
		}
		if cur != nil {
			text = append(text, line)
		}
	}
	flush()
	return segs, sc.Err()
}

func parseVTTTime(s string) (time.Duration, error) {
	var h, m, sec, ms int
	var err error
	if strings.Count(s, ":") == 2 {
		_, err = fmt.Sscanf(s, "%d:%d:%d.%d", &h, &m, &sec, &ms)
	} else {
		_, err = fmt.Sscanf(s, "%d:%d.%d", &m, &sec, &ms)
	}
	if err != nil {
		return 0, fmt.Errorf("bad timestamp %q: %w", s, err)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second + time.Duration(ms)*time.Millisecond, nil
}

func cleanText(s string) string {
	s = html.UnescapeString(vttTag.ReplaceAllString(s, ""))
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

type whisperJSON struct {
	Result struct {
		Language string `json:"language"`
	} `json:"result"`
	Transcription []struct {
		Offsets struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

// ParseWhisperJSON parses the file written by `whisper-cli -oj` and returns
// the segments and the detected language.
func ParseWhisperJSON(data []byte) ([]Segment, string, error) {
	var j whisperJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, "", fmt.Errorf("parse whisper json: %w", err)
	}
	segs := make([]Segment, 0, len(j.Transcription))
	for _, t := range j.Transcription {
		text := cleanText(t.Text)
		if text == "" {
			continue
		}
		segs = append(segs, Segment{
			Start: time.Duration(t.Offsets.From) * time.Millisecond,
			End:   time.Duration(t.Offsets.To) * time.Millisecond,
			Text:  text,
		})
	}
	return segs, j.Result.Language, nil
}
