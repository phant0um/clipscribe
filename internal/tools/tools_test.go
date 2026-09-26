package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/phant0um/clipscribe/internal/media"
)

type call struct {
	bin  string
	args []string
}

type fakeRunner struct {
	calls  []call
	stdout []byte
	err    error
	effect func(args []string) // simulates files written by the tool
}

func (f *fakeRunner) run(ctx context.Context, bin string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, call{bin, args})
	if f.effect != nil {
		f.effect(args)
	}
	return f.stdout, f.err
}

func argAfter(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestYtDlpProbeArgs(t *testing.T) {
	b, _ := os.ReadFile("../../testdata/ytdlp/youtube-ted-ed.json")
	r := &fakeRunner{stdout: b}
	y := YtDlp{Bin: "yt-dlp", run: r.run}
	v, err := y.Probe(context.Background(), "https://www.youtube.com/watch?v=qD0_yWgifDM", media.FetchOpts{})
	if err != nil || v.ID != "qD0_yWgifDM" {
		t.Fatalf("v=%+v err=%v", v, err)
	}
	want := []string{"--ignore-config", "--no-playlist", "--no-progress", "--use-extractors", "youtube,twitter", "--dump-single-json", "--", "https://www.youtube.com/watch?v=qD0_yWgifDM"}
	if !reflect.DeepEqual(r.calls[0].args, want) {
		t.Errorf("args = %q\nwant  %q", r.calls[0].args, want)
	}
}

func TestYtDlpCookiesOnlyWhenGiven(t *testing.T) {
	b, _ := os.ReadFile("../../testdata/ytdlp/youtube-ted-ed.json")
	r := &fakeRunner{stdout: b}
	y := YtDlp{Bin: "yt-dlp", run: r.run}
	y.Probe(context.Background(), "u", media.FetchOpts{Cookies: "/k/c.txt"})
	args := r.calls[0].args
	if argAfter(args, "--cookies") != "/k/c.txt" {
		t.Errorf("cookies not passed: %q", args)
	}
	if args[len(args)-2] != "--" {
		t.Errorf("URL must follow --: %q", args)
	}
	for _, a := range args {
		if strings.Contains(a, "exec") || strings.Contains(a, "cookies-from-browser") || strings.Contains(a, "netrc") {
			t.Errorf("forbidden flag %q", a)
		}
	}
}

func TestYtDlpFetchAudio(t *testing.T) {
	dir := t.TempDir()
	r := &fakeRunner{effect: func(args []string) {
		out := strings.Replace(argAfter(args, "-o"), "%(ext)s", "webm", 1)
		os.WriteFile(out, []byte("a"), 0o600)
		os.WriteFile(out+".part", []byte("p"), 0o600)
	}}
	y := YtDlp{Bin: "yt-dlp", run: r.run}
	p, err := y.FetchAudio(context.Background(), "u", dir, media.FetchOpts{})
	if err != nil || p != filepath.Join(dir, "audio.webm") {
		t.Fatalf("p=%q err=%v", p, err)
	}
	if argAfter(r.calls[0].args, "-f") != "bestaudio/best" {
		t.Errorf("args %q", r.calls[0].args)
	}
}

func TestYtDlpFetchCaptions(t *testing.T) {
	dir := t.TempDir()
	r := &fakeRunner{effect: func(args []string) {
		out := strings.NewReplacer("%(id)s", "qD0_yWgifDM", "%(ext)s", "en.vtt").Replace(argAfter(args, "-o"))
		os.WriteFile(out, []byte("WEBVTT\n"), 0o600)
	}}
	y := YtDlp{Bin: "yt-dlp", run: r.run}
	p, err := y.FetchCaptions(context.Background(), "u", "en", dir, media.FetchOpts{})
	if err != nil || filepath.Ext(p) != ".vtt" {
		t.Fatalf("p=%q err=%v", p, err)
	}
	args := r.calls[0].args
	for _, want := range []string{"--skip-download", "--write-subs", "--no-write-auto-subs"} {
		if !contains(args, want) {
			t.Errorf("missing %s in %q", want, args)
		}
	}
	if argAfter(args, "--sub-langs") != "en" || argAfter(args, "--sub-format") != "vtt" {
		t.Errorf("args %q", args)
	}
}

func TestYtDlpErrorMapping(t *testing.T) {
	cases := []struct {
		stderr string
		want   error
	}{
		{"ERROR: [youtube] x: Sign in to confirm you're not a bot", media.ErrAuthRequired},
		{"ERROR: [youtube] x: Sign in to confirm your age", media.ErrAuthRequired},
		{"ERROR: [twitter] 1: NSFW tweet requires authentication", media.ErrAuthRequired},
		{"ERROR: [twitter] 1: No video could be found in this tweet", media.ErrNoVideo},
	}
	for _, c := range cases {
		r := &fakeRunner{err: &ExitError{Stderr: c.stderr}}
		y := YtDlp{Bin: "yt-dlp", run: r.run}
		if _, err := y.Probe(context.Background(), "u", media.FetchOpts{}); !errors.Is(err, c.want) {
			t.Errorf("%q: err = %v, want %v", c.stderr, err, c.want)
		}
	}
}

func TestYtDlpErrorsNeverShowCookiePath(t *testing.T) {
	r := &fakeRunner{err: &ExitError{Stderr: "ERROR: could not load cookies from /Users/me/secret-cookies.txt"}}
	y := YtDlp{Bin: "yt-dlp", run: r.run}
	_, err := y.Probe(context.Background(), "u", media.FetchOpts{Cookies: "/Users/me/secret-cookies.txt"})
	if err == nil || strings.Contains(err.Error(), "secret-cookies") {
		t.Fatalf("err = %v", err)
	}
}

func TestFFmpegArgs(t *testing.T) {
	r := &fakeRunner{}
	f := FFmpeg{Bin: "ffmpeg", run: r.run}
	if err := f.ToWAV16k(context.Background(), "/t/in.webm", "/t/out.wav"); err != nil {
		t.Fatal(err)
	}
	want := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", "/t/in.webm", "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-y", "/t/out.wav"}
	if !reflect.DeepEqual(r.calls[0].args, want) {
		t.Errorf("args = %q", r.calls[0].args)
	}
}

func TestWhisperArgsAndOutput(t *testing.T) {
	fixture, _ := os.ReadFile("../../testdata/whisper/pt-sacani.json")
	r := &fakeRunner{effect: func(args []string) {
		os.WriteFile(argAfter(args, "-of")+".json", fixture, 0o600)
	}}
	w := Whisper{Bin: "whisper-cli", Model: "/m/model.bin", run: r.run}
	wav := filepath.Join(t.TempDir(), "audio.wav")
	segs, lang, err := w.Transcribe(context.Background(), wav, "auto")
	if err != nil || lang != "pt" || len(segs) != 40 || segs[1].Text != "da aula." {
		t.Fatalf("lang=%q n=%d err=%v", lang, len(segs), err)
	}
	args := r.calls[0].args
	if argAfter(args, "-m") != "/m/model.bin" || argAfter(args, "-f") != wav || argAfter(args, "-l") != "auto" || !contains(args, "-oj") || !contains(args, "-np") {
		t.Errorf("args = %q", args)
	}
	if filepath.Dir(argAfter(args, "-of")) != filepath.Dir(wav) {
		t.Errorf("output must stay in the temp dir: %q", args)
	}
}

func contains(s []string, x string) bool {
	for _, a := range s {
		if a == x {
			return true
		}
	}
	return false
}
