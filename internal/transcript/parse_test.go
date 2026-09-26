package transcript

import (
	"os"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../testdata/" + path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseVTT(t *testing.T) {
	segs, err := ParseVTT(fixture(t, "captions/qD0_yWgifDM.pt-BR.vtt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) < 50 {
		t.Fatalf("got %d segments", len(segs))
	}
	s := segs[1]
	if s.Start != 6951*time.Millisecond || s.End != 8951*time.Millisecond || s.Text != "exemplo simples com frases curtas para" {
		t.Errorf("segs[1] = %+v", s)
	}
	if segs[2].Text != "cada trecho da aula o texto?" {
		t.Errorf("multi-line cue = %q", segs[2].Text)
	}
}

func TestParseVTTStripsTagsAndEntities(t *testing.T) {
	in := "WEBVTT\n\nNOTE comment\n\ncue-1\n00:01:02.500 --> 00:01:04.000 align:start\n<v Ana><i>Olá</i> &amp; tchau</v>\n"
	segs, err := ParseVTT([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 || segs[0].Start != 62500*time.Millisecond || segs[0].Text != "Olá & tchau" {
		t.Fatalf("segs = %+v", segs)
	}
}

func TestParseVTTRejectsNonVTT(t *testing.T) {
	if _, err := ParseVTT([]byte("<html>")); err == nil {
		t.Fatal("want error")
	}
}

func TestParseWhisperJSON(t *testing.T) {
	segs, lang, err := ParseWhisperJSON(fixture(t, "whisper/en-ted-ed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if lang != "en" || len(segs) != 61 {
		t.Fatalf("lang=%q n=%d", lang, len(segs))
	}
	if segs[1].Start != 11080*time.Millisecond || segs[1].End != 12820*time.Millisecond || segs[1].Text != "of the lesson this synthetic test text?" {
		t.Errorf("segs[1] = %+v", segs[1])
	}
	for _, s := range segs {
		if s.Text != strings.TrimSpace(s.Text) {
			t.Fatalf("untrimmed text %q", s.Text)
		}
	}
}

func TestParseWhisperJSONPortuguese(t *testing.T) {
	segs, lang, err := ParseWhisperJSON(fixture(t, "whisper/pt-sacani.json"))
	if err != nil || lang != "pt" || segs[1].Text != "da aula." {
		t.Fatalf("lang=%q seg=%q err=%v", lang, segs[1].Text, err)
	}
}
