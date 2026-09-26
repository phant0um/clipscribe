package media

import (
	"os"
	"testing"
	"time"
)

func load(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../testdata/ytdlp/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseVideoYouTube(t *testing.T) {
	v, err := ParseVideo(load(t, "youtube-ted-ed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Platform != YouTube || v.ID != "qD0_yWgifDM" {
		t.Errorf("platform/id = %s/%s", v.Platform, v.ID)
	}
	if v.Title != "The science of spiciness - Rose Eveleth" || v.Author != "TED-Ed" {
		t.Errorf("title/author = %q/%q", v.Title, v.Author)
	}
	if !v.Published.Equal(time.Date(2014, 3, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("published = %v", v.Published)
	}
	if v.Duration != 234*time.Second || v.Language != "en" {
		t.Errorf("duration/lang = %v/%q", v.Duration, v.Language)
	}
	if len(v.ManualSubs) != 33 {
		t.Errorf("manual subs = %d, want 33", len(v.ManualSubs))
	}
}

func TestParseVideoXUsesPostIDAndHandle(t *testing.T) {
	v, err := ParseVideo(load(t, "x-poteto.json"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Platform != X || v.ID != "2102050467505430555" {
		t.Errorf("platform/id = %s/%s, want x/post id", v.Platform, v.ID)
	}
	if v.Author != "poteto" || v.Language != "" {
		t.Errorf("author/lang = %q/%q", v.Author, v.Language)
	}
	if v.Duration != 2281984*time.Millisecond {
		t.Errorf("duration = %v", v.Duration)
	}
}

func TestParseVideoRejectsUnknownExtractor(t *testing.T) {
	if _, err := ParseVideo([]byte(`{"id":"1","extractor_key":"Vimeo"}`)); err == nil {
		t.Fatal("want error for unsupported extractor")
	}
}

func TestCaptionFor(t *testing.T) {
	v := Video{ManualSubs: []string{"en", "pt-BR", "pt-PT"}}
	cases := map[string]string{"pt": "pt-BR", "en": "en", "pt-PT": "pt-PT", "es": "", "": ""}
	for lang, want := range cases {
		if got := v.CaptionFor(lang); got != want {
			t.Errorf("CaptionFor(%q) = %q, want %q", lang, got, want)
		}
	}
}

func TestParseVideoLive(t *testing.T) { // v1.1, shield S5
	for status, want := range map[string]bool{"is_live": true, "is_upcoming": true, "was_live": false, "not_live": false} {
		v, err := ParseVideo([]byte(`{"id":"qD0_yWgifDM","extractor_key":"Youtube","live_status":"` + status + `"}`))
		if err != nil || v.Live != want {
			t.Errorf("%s: live=%v err=%v", status, v.Live, err)
		}
	}
}
