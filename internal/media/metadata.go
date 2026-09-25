package media

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Video is the normalized metadata of one video, as reported by yt-dlp.
type Video struct {
	Platform    Platform
	ID          string // YouTube video ID or X post ID
	URL         string
	Title       string
	Author      string // YouTube channel name or X handle without "@"
	Published   time.Time
	Duration    time.Duration
	Description string
	Language    string   // spoken language; empty when unknown
	ManualSubs  []string // languages with human-made captions, sorted
}

type ytdlpJSON struct {
	ID           string           `json:"id"`
	DisplayID    string           `json:"display_id"`
	Title        string           `json:"title"`
	Uploader     string           `json:"uploader"`
	UploaderID   string           `json:"uploader_id"`
	Channel      string           `json:"channel"`
	UploadDate   string           `json:"upload_date"`
	Duration     float64          `json:"duration"`
	Description  string           `json:"description"`
	WebpageURL   string           `json:"webpage_url"`
	Language     string           `json:"language"`
	ExtractorKey string           `json:"extractor_key"`
	Subtitles    map[string][]any `json:"subtitles"`
}

// ParseVideo converts the output of `yt-dlp --dump-single-json`.
func ParseVideo(data []byte) (Video, error) {
	var j ytdlpJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return Video{}, fmt.Errorf("parse yt-dlp json: %w", err)
	}
	v := Video{
		ID:          j.ID,
		URL:         j.WebpageURL,
		Title:       j.Title,
		Duration:    time.Duration(j.Duration * float64(time.Second)),
		Description: j.Description,
		Language:    j.Language,
	}
	switch j.ExtractorKey {
	case "Youtube":
		v.Platform = YouTube
		v.Author = firstNonEmpty(j.Channel, j.Uploader, strings.TrimPrefix(j.UploaderID, "@"))
	case "Twitter":
		v.Platform = X
		v.ID = firstNonEmpty(j.DisplayID, j.ID) // post ID, matches the URL
		v.Author = strings.TrimPrefix(j.UploaderID, "@")
	default:
		return Video{}, fmt.Errorf("unsupported extractor %q", j.ExtractorKey)
	}
	if v.ID == "" {
		return Video{}, fmt.Errorf("yt-dlp json without id")
	}
	if t, err := time.Parse("20060102", j.UploadDate); err == nil {
		v.Published = t
	}
	for lang := range j.Subtitles {
		if lang != "live_chat" {
			v.ManualSubs = append(v.ManualSubs, lang)
		}
	}
	sort.Strings(v.ManualSubs)
	return v, nil
}

// CaptionFor returns the manual caption code that matches the spoken
// language lang, or "" when there is none. A base code ("pt") matches a
// regional variant, preferring "pt-BR".
func (v Video) CaptionFor(lang string) string {
	if lang == "" {
		return ""
	}
	var prefixed []string
	for _, s := range v.ManualSubs {
		if strings.EqualFold(s, lang) {
			return s
		}
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(lang)+"-") {
			prefixed = append(prefixed, s)
		}
	}
	for _, s := range prefixed {
		if s == "pt-BR" {
			return s
		}
	}
	if len(prefixed) > 0 {
		return prefixed[0]
	}
	return ""
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}
