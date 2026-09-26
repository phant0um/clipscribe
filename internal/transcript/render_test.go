package transcript

import (
	"strings"
	"testing"
	"time"

	"github.com/phant0um/clipscribe/internal/media"
)

func doc(p media.Platform) Doc {
	v := media.Video{
		Platform:    p,
		ID:          "qD0_yWgifDM",
		URL:         "https://www.youtube.com/watch?v=qD0_yWgifDM",
		Title:       `Say "hi" \ bye`,
		Author:      "TED-Ed",
		Published:   time.Date(2014, 3, 10, 0, 0, 0, 0, time.UTC),
		Duration:    time.Hour + 2*time.Minute + 3*time.Second,
		Description: "Line one.\nLine two.",
	}
	if p == media.X {
		v.ID, v.URL, v.Author, v.Published = "2102050467505430555", "https://x.com/poteto/status/2102050467505430555", "poteto", time.Time{}
	}
	return Doc{
		Video:       v,
		Lang:        "pt",
		Transcriber: "whisper-large-v3-turbo",
		AudioSHA256: "abc123",
		Created:     time.Date(2026, 9, 25, 15, 0, 0, 0, time.UTC),
		Segments:    []Segment{seg(0, 2, "Olá."), seg(34.6, 36, "Tchau.")},
	}
}

func TestRenderMarkdownYouTube(t *testing.T) {
	want := `---
title: "Say \"hi\" \\ bye"
source: "https://www.youtube.com/watch?v=qD0_yWgifDM"
author:
  - "[[TED-Ed]]"
published: 2014-03-10
created: 2026-09-25
description: "Line one. Line two."
platform: "youtube"
video_id: "qD0_yWgifDM"
duration: "01:02:03"
lang: "pt"
transcriber: "whisper-large-v3-turbo"
audio_sha256: "abc123"
tags:
  - "clippings"
  - "transcript"
---
[00:00:00](https://youtu.be/qD0_yWgifDM?t=0) Olá.

[00:00:34](https://youtu.be/qD0_yWgifDM?t=34) Tchau.
`
	if got := string(RenderMarkdown(doc(media.YouTube))); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderMarkdownXOmitsUnknownFieldsAndLinks(t *testing.T) {
	d := doc(media.X)
	d.AudioSHA256 = ""
	d.Transcriber = "captions-manual"
	want := `---
title: "Say \"hi\" \\ bye"
source: "https://x.com/poteto/status/2102050467505430555"
author:
  - "[[@poteto]]"
created: 2026-09-25
description: "Line one. Line two."
platform: "x"
video_id: "2102050467505430555"
duration: "01:02:03"
lang: "pt"
transcriber: "captions-manual"
tags:
  - "clippings"
  - "transcript"
---
[00:00:00] Olá.

[00:00:34] Tchau.
`
	if got := string(RenderMarkdown(d)); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderMarkdownNeverBreaksFrontmatter(t *testing.T) {
	d := doc(media.YouTube)
	d.Video.Title = "a\n---\ninjected: true"
	got := string(RenderMarkdown(d))
	if want := `title: "a\n---\ninjected: true"`; !strings.Contains(got, want) {
		t.Errorf("title not escaped:\n%s", got)
	}
}

func TestDescriptionFallsBackToSpeechAndTruncates(t *testing.T) {
	d := doc(media.YouTube)
	d.Video.Description = ""
	if got := description(d); got != "Olá. Tchau." {
		t.Errorf("fallback = %q", got)
	}
	d.Video.Description = strings.Repeat("palavra ", 40)
	if got := description(d); len([]rune(got)) > 161 || !strings.HasSuffix(got, "…") {
		t.Errorf("truncated = %q (%d runes)", got, len([]rune(got)))
	}
}

func TestRenderSRT(t *testing.T) {
	want := "1\n00:00:00,000 --> 00:00:02,000\nOlá.\n\n2\n00:00:34,600 --> 00:00:36,000\nTchau.\n"
	if got := string(RenderSRT(doc(media.YouTube).Segments)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderText(t *testing.T) {
	if got := string(RenderText(doc(media.YouTube).Segments)); got != "Olá.\n\nTchau.\n" {
		t.Errorf("got %q", got)
	}
}

func TestRenderMarkdownEscapesLinksAndImages(t *testing.T) { // audit finding 3
	d := doc(media.X)
	d.Segments = []Segment{seg(0, 2, "veja ![x](https://evil.example/p.png) e [[Nota]]")}
	got := string(RenderMarkdown(d))
	if want := `[00:00:00] veja !\[x\](https://evil.example/p.png) e \[\[Nota\]\]`; !strings.Contains(got, want) {
		t.Errorf("body not escaped:\n%s", got)
	}
}

func TestRenderMarkdownNeutralizesActiveContent(t *testing.T) { // shield 2026-09-26, finding 1
	vtt := "WEBVTT\n\n00:00:00.000 --> 00:00:02.000\n" +
		"hi &lt;img src=\"https://evil.example/p.png\"&gt; &lt;%* await app.vault.adapter.write('pwn.md','x') %&gt; `$= dv.el('b','x')` #injected\n"
	segs, err := ParseVTT([]byte(vtt))
	if err != nil {
		t.Fatal(err)
	}
	d := doc(media.X)
	d.Segments = segs
	got := string(RenderMarkdown(d))
	body := got[strings.LastIndex(got, "---\n"):]
	for _, bad := range []string{"<img", "<%", " `$=", " #injected"} {
		if strings.Contains(body, bad) {
			t.Errorf("body keeps %q:\n%s", bad, body)
		}
	}
	if want := `hi &lt;img src="https://evil.example/p.png"&gt; &lt;%* await`; !strings.Contains(body, want) {
		t.Errorf("body missing %q:\n%s", want, body)
	}
	if want := "\\`$= dv.el('b','x')\\` \\#injected"; !strings.Contains(body, want) {
		t.Errorf("body missing %q:\n%s", want, body)
	}
}

func TestRenderMarkdownFrontmatterHasNoLinks(t *testing.T) { // shield 2026-09-26, finding 3
	d := doc(media.YouTube)
	d.Video.Author = "Chan]] [[Evil|x"
	d.Video.Title = "see [[Secret Note]]"
	d.Video.Description = "and [[Other]]"
	got := string(RenderMarkdown(d))
	for _, want := range []string{`- "[[Chan Evil x]]"`, `title: "see \\[\\[Secret Note\\]\\]"`, `description: "and \\[\\[Other\\]\\]"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in:\n%s", want, got)
		}
	}
}
