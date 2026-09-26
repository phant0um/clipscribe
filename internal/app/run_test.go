package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phant0um/clipscribe/internal/media"
	"github.com/phant0um/clipscribe/internal/transcript"
)

const (
	tedURL = "https://www.youtube.com/watch?v=qD0_yWgifDM"
	ptURL  = "https://youtu.be/MKU9suCNJQo"
	xURL   = "https://x.com/poteto/status/2102050467505430555/video/1"
	// sha256("fake wav"), computed with shasum.
	fakeWavSHA = "007e1f0495cedd59cc57aad6d160baafaa905366d148905f73d084fbc0c5093c"
)

type fakes struct {
	video     media.Video
	probeErr  error
	transErr  error
	onTrans   func()
	probes    int
	captions  []string
	audio     int
	transLang string
	prompt    string
	cookies   []string
	expectIDs []string
}

func (f *fakes) Probe(ctx context.Context, url string, o FetchOpts) (media.Video, error) {
	f.probes++
	f.cookies = append(f.cookies, o.Cookies)
	return f.video, f.probeErr
}

func (f *fakes) FetchCaptions(ctx context.Context, url, lang, dir string, o FetchOpts) (string, error) {
	f.captions = append(f.captions, lang)
	f.expectIDs = append(f.expectIDs, o.ExpectID)
	b, err := os.ReadFile("../../testdata/captions/qD0_yWgifDM.pt-BR.vtt")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "subs.vtt")
	return p, os.WriteFile(p, b, 0o600)
}

func (f *fakes) FetchAudio(ctx context.Context, url, dir string, o FetchOpts) (string, error) {
	f.audio++
	f.expectIDs = append(f.expectIDs, o.ExpectID)
	p := filepath.Join(dir, "audio.webm")
	return p, os.WriteFile(p, []byte("webm"), 0o600)
}

func (f *fakes) ToWAV16k(ctx context.Context, in, out string) error {
	return os.WriteFile(out, []byte("fake wav"), 0o600)
}

func (f *fakes) Transcribe(ctx context.Context, wav, lang, prompt string) ([]transcript.Segment, string, error) {
	f.transLang = lang
	f.prompt = prompt
	if f.onTrans != nil {
		f.onTrans()
	}
	if f.transErr != nil {
		return nil, "", f.transErr
	}
	return []transcript.Segment{{Start: 0, End: 2 * time.Second, Text: "Olá mundo."}}, "pt", nil
}

type env struct {
	f              *fakes
	d              Deps
	out, err       *bytes.Buffer
	outDir, tmpDir string
}

func newEnv(t *testing.T, fixture string) *env {
	t.Helper()
	b, err := os.ReadFile("../../testdata/ytdlp/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	v, err := media.ParseVideo(b)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	e := &env{f: &fakes{video: v}, out: &bytes.Buffer{}, err: &bytes.Buffer{},
		outDir: filepath.Join(root, "clippings"), tmpDir: filepath.Join(root, "tmp")}
	for _, d := range []string{e.outDir, e.tmpDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	model := filepath.Join(root, "model.bin")
	os.WriteFile(model, []byte("m"), 0o600)
	cfg := filepath.Join(root, "config.json")
	os.WriteFile(cfg, []byte(`{"out_dir":"`+e.outDir+`","model_path":"`+model+`"}`), 0o600)
	e.d = Deps{
		Downloader: e.f, Converter: e.f, Transcriber: e.f,
		LookPath: func(name string) (string, error) { return "/opt/homebrew/bin/" + name, nil },
		Now:      func() time.Time { return time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC) },
		Stdout:   e.out, Stderr: e.err,
		ConfigPath: cfg, TempDir: e.tmpDir,
	}
	return e
}

func (e *env) run(args ...string) int { return Run(context.Background(), args, e.d) }

func (e *env) only(t *testing.T) (string, string) {
	t.Helper()
	entries, _ := os.ReadDir(e.outDir)
	if len(entries) != 1 {
		t.Fatalf("want 1 clipping, got %v", entries)
	}
	p := filepath.Join(e.outDir, entries[0].Name())
	b, _ := os.ReadFile(p)
	return p, string(b)
}

func (e *env) tmpEmpty(t *testing.T) {
	t.Helper()
	if entries, _ := os.ReadDir(e.tmpDir); len(entries) != 0 {
		t.Errorf("temporary files left: %v", entries)
	}
}

func TestRunWithoutArgsIsUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run(context.Background(), nil, Deps{Stdout: &out, Stderr: &errb}); code != ExitUsage || errb.Len() == 0 {
		t.Fatalf("exit = %d stderr = %q", code, errb.String())
	}
}

func TestManualCaptionInSpokenLanguageSkipsWhisper(t *testing.T) { // AC1.5
	e := newEnv(t, "youtube-ted-ed.json") // spoken en, manual en
	if code := e.run(tedURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if e.f.audio != 0 || len(e.f.captions) != 1 || e.f.captions[0] != "en" {
		t.Errorf("audio=%d captions=%v", e.f.audio, e.f.captions)
	}
	p, body := e.only(t)
	if strings.TrimSpace(e.out.String()) != p {
		t.Errorf("stdout = %q, want %q", e.out, p)
	}
	for _, want := range []string{`transcriber: "captions-manual"`, `lang: "en"`, `video_id: "qD0_yWgifDM"`, "[00:00:00](https://youtu.be/qD0_yWgifDM?t=0)"} {
		if !strings.Contains(body, want) {
			t.Errorf("clipping missing %q", want)
		}
	}
	if strings.Contains(body, "audio_sha256") {
		t.Error("caption clipping must not have audio_sha256")
	}
	if filepath.Base(p) != "The science of spiciness - Rose Eveleth.md" {
		t.Errorf("name = %q", filepath.Base(p))
	}
	e.tmpEmpty(t)
}

func TestTranslatedCaptionIsIgnored(t *testing.T) { // C4
	e := newEnv(t, "youtube-ted-ed.json")
	e.f.video.ManualSubs = []string{"pt-BR"}
	if code := e.run(tedURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if len(e.f.captions) != 0 || e.f.audio != 1 {
		t.Errorf("captions=%v audio=%d", e.f.captions, e.f.audio)
	}
}

func TestWhisperPath(t *testing.T) { // AC1.6
	e := newEnv(t, "youtube-pt-nosubs.json")
	if code := e.run(ptURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if e.f.transLang != "auto" {
		t.Errorf("whisper lang = %q, want auto", e.f.transLang)
	}
	_, body := e.only(t)
	for _, want := range []string{`transcriber: "whisper-large-v3-turbo"`, `audio_sha256: "` + fakeWavSHA + `"`, `lang: "pt"`, "Olá mundo."} {
		if !strings.Contains(body, want) {
			t.Errorf("clipping missing %q:\n%s", want, body)
		}
	}
	e.tmpEmpty(t)
}

func TestLangFlagIsPassedToWhisper(t *testing.T) {
	e := newEnv(t, "youtube-pt-nosubs.json")
	if code := e.run(ptURL, "--lang", "pt"); code != ExitOK || e.f.transLang != "pt" {
		t.Fatalf("exit %d lang %q", code, e.f.transLang)
	}
}

func TestXAlwaysUsesWhisper(t *testing.T) { // AC2.1, C5
	e := newEnv(t, "x-poteto.json") // has "en" subs but unknown spoken language
	if code := e.run(xURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if len(e.f.captions) != 0 || e.f.audio != 1 {
		t.Errorf("captions=%v audio=%d", e.f.captions, e.f.audio)
	}
	p, body := e.only(t)
	if !strings.HasPrefix(filepath.Base(p), "@poteto — ") || !strings.Contains(body, `"[[@poteto]]"`) {
		t.Errorf("name %q body:\n%s", p, body)
	}
}

func TestXPostWithoutVideo(t *testing.T) { // AC2.3
	e := newEnv(t, "x-poteto.json")
	e.f.probeErr = media.ErrNoVideo
	if code := e.run(xURL); code != ExitRuntime {
		t.Fatalf("exit %d", code)
	}
	if entries, _ := os.ReadDir(e.outDir); len(entries) != 0 {
		t.Errorf("unexpected files %v", entries)
	}
}

func TestInvalidInputNeverCallsSubprocess(t *testing.T) { // AC4.1, AC4.2
	for _, args := range [][]string{
		{"https://vimeo.com/1"},
		{"http://www.youtube.com/watch?v=qD0_yWgifDM"},
		{tedURL, "--format", "pdf"},
		{tedURL, "--lang", "fr"},
		{tedURL, "--bogus"},
		{tedURL, ptURL},
	} {
		e := newEnv(t, "youtube-ted-ed.json")
		if code := e.run(args...); code != ExitUsage || e.f.probes != 0 {
			t.Errorf("%v: exit %d probes %d", args, code, e.f.probes)
		}
	}
}

func TestCookieFile(t *testing.T) { // AC3.1, AC3.2, AC3.4
	e := newEnv(t, "youtube-pt-nosubs.json")
	secret := filepath.Join(t.TempDir(), "secret-cookies.txt")
	os.WriteFile(secret, []byte("# Netscape HTTP Cookie File\n"), 0o644)
	if code := e.run(ptURL, "--cookies", secret); code != ExitUsage || e.f.probes != 0 {
		t.Fatalf("open cookie file: exit %d probes %d", code, e.f.probes)
	}
	if strings.Contains(e.err.String(), "secret-cookies") {
		t.Error("stderr leaks the cookie path")
	}
	os.Chmod(secret, 0o600)
	if code := e.run(ptURL, "--cookies", secret); code != ExitOK || e.f.cookies[0] != secret {
		t.Fatalf("exit %d cookies %v: %s", code, e.f.cookies, e.err)
	}
	_, body := e.only(t)
	if strings.Contains(body+e.err.String()+e.out.String(), "secret-cookies") {
		t.Error("cookie path leaked")
	}
}

func TestAuthRequiredSuggestsCookiesWithoutRetry(t *testing.T) { // AC3.3
	e := newEnv(t, "x-poteto.json")
	e.f.probeErr = media.ErrAuthRequired
	if code := e.run(xURL); code != ExitRuntime || e.f.probes != 1 || !strings.Contains(e.err.String(), "--cookies") {
		t.Fatalf("exit %d probes %d stderr %q", code, e.f.probes, e.err)
	}
}

func TestSameVideoTwiceIsSkipped(t *testing.T) { // AC5.1
	e := newEnv(t, "youtube-pt-nosubs.json")
	e.run(ptURL)
	first, _ := e.only(t)
	e.out.Reset()
	e.f.audio = 0
	if code := e.run(ptURL); code != ExitOK || e.f.audio != 0 {
		t.Fatalf("exit %d audio %d", code, e.f.audio)
	}
	if strings.TrimSpace(e.out.String()) != first || !strings.Contains(e.err.String(), "already") {
		t.Errorf("stdout %q stderr %q", e.out, e.err)
	}
	e.only(t)
}

func TestForceReplacesExistingClipping(t *testing.T) { // AC5.2
	e := newEnv(t, "youtube-pt-nosubs.json")
	e.run(ptURL)
	p, _ := e.only(t)
	os.WriteFile(p, []byte("---\nplatform: \"youtube\"\nvideo_id: \"MKU9suCNJQo\"\n---\nstale\n"), 0o644)
	if code := e.run(ptURL, "--force"); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	p2, body := e.only(t)
	if p2 != p || strings.Contains(body, "stale") {
		t.Errorf("not replaced: %q\n%s", p2, body)
	}
}

func TestCancelCleansUp(t *testing.T) { // AC5.3
	e := newEnv(t, "youtube-pt-nosubs.json")
	ctx, cancel := context.WithCancel(context.Background())
	e.f.onTrans = cancel
	e.f.transErr = context.Canceled
	if code := Run(ctx, []string{ptURL}, e.d); code != ExitRuntime {
		t.Fatalf("exit %d", code)
	}
	if entries, _ := os.ReadDir(e.outDir); len(entries) != 0 {
		t.Errorf("partial clipping: %v", entries)
	}
	e.tmpEmpty(t)
}

func TestKeepAudio(t *testing.T) { // AC5.4
	e := newEnv(t, "youtube-pt-nosubs.json")
	if code := e.run(ptURL, "--keep-audio"); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var wav string
	for _, line := range strings.Split(e.err.String(), "\n") {
		if i := strings.Index(line, "audio kept: "); i >= 0 {
			wav = strings.TrimSpace(line[i+len("audio kept: "):])
		}
	}
	if b, err := os.ReadFile(wav); err != nil || string(b) != "fake wav" {
		t.Errorf("wav %q: %v", wav, err)
	}
}

func TestOtherFormatsGoToOutDirNotClippings(t *testing.T) { // AC6.1-AC6.3
	for _, format := range []string{"srt", "txt"} {
		e := newEnv(t, "youtube-pt-nosubs.json")
		dir := t.TempDir()
		if code := e.run(ptURL, "--format", format, "--out", dir); code != ExitOK {
			t.Fatalf("%s: exit %d %s", format, code, e.err)
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 1 || filepath.Ext(entries[0].Name()) != "."+format {
			t.Errorf("%s: %v", format, entries)
		}
		if c, _ := os.ReadDir(e.outDir); len(c) != 0 {
			t.Errorf("%s written to clippings", format)
		}
	}
}

func TestMissingDependency(t *testing.T) { // AC7.2
	e := newEnv(t, "youtube-pt-nosubs.json")
	e.d.LookPath = func(name string) (string, error) {
		if name == "yt-dlp" {
			return "", errors.New("not found")
		}
		return "/bin/" + name, nil
	}
	if code := e.run(ptURL); code != ExitMissing || !strings.Contains(e.err.String(), "brew install yt-dlp") || e.f.probes != 0 {
		t.Fatalf("exit %d probes %d stderr %q", code, e.f.probes, e.err)
	}
}

func TestMissingModel(t *testing.T) {
	e := newEnv(t, "youtube-pt-nosubs.json")
	os.WriteFile(e.d.ConfigPath, []byte(`{"out_dir":"`+e.outDir+`","model_path":"/nope/model.bin"}`), 0o600)
	if code := e.run(ptURL); code != ExitMissing || !strings.Contains(e.err.String(), "doctor --install-model") {
		t.Fatalf("exit %d stderr %q", code, e.err)
	}
}

func TestForceNeverRewritesArchive(t *testing.T) { // audit finding 2
	e := newEnv(t, "youtube-pt-nosubs.json")
	archive := filepath.Join(filepath.Dir(e.outDir), "archive")
	old := filepath.Join(archive, "2026-09-01", "old.md")
	os.MkdirAll(filepath.Dir(old), 0o755)
	stale := "---\nplatform: \"youtube\"\nvideo_id: \"MKU9suCNJQo\"\n---\narchived\n"
	os.WriteFile(old, []byte(stale), 0o644)
	cfg := `{"out_dir":"` + e.outDir + `","dedup_dirs":["` + e.outDir + `","` + archive + `"],"model_path":"` + filepath.Join(filepath.Dir(e.outDir), "model.bin") + `"}`
	os.WriteFile(e.d.ConfigPath, []byte(cfg), 0o600)

	if code := e.run(ptURL); code != ExitOK || strings.TrimSpace(e.out.String()) != old {
		t.Fatalf("without --force: exit %d stdout %q", code, e.out)
	}
	e.out.Reset()
	if code := e.run(ptURL, "--force"); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if b, _ := os.ReadFile(old); string(b) != stale {
		t.Error("archived clipping was modified")
	}
	p, _ := e.only(t)
	if strings.TrimSpace(e.out.String()) != p {
		t.Errorf("stdout %q, want new clipping %q", e.out, p)
	}
}

func TestProbedVideoMustMatchURL(t *testing.T) { // shield 2026-09-26, finding 2
	e := newEnv(t, "youtube-ted-ed.json")
	e.f.video.ID = "AAAAAAAAAAA"
	if code := e.run(tedURL); code != ExitRuntime {
		t.Fatalf("exit %d", code)
	}
	if entries, _ := os.ReadDir(e.outDir); len(entries) != 0 {
		t.Errorf("unexpected files %v", entries)
	}
}

func TestSourceIsTheValidatedURL(t *testing.T) { // shield 2026-09-26, finding 2
	e := newEnv(t, "youtube-ted-ed.json")
	e.f.video.URL = "https://evil.example/watch"
	if code := e.run(tedURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if _, body := e.only(t); !strings.Contains(body, `source: "https://www.youtube.com/watch?v=qD0_yWgifDM"`) {
		t.Errorf("source not from validated URL:\n%s", body[:300])
	}
}

func TestLiveStreamIsRefused(t *testing.T) { // v1.1, shield S5
	e := newEnv(t, "youtube-pt-nosubs.json")
	e.f.video.Live = true
	if code := e.run(ptURL); code != ExitRuntime || e.f.audio != 0 {
		t.Fatalf("exit %d audio %d", code, e.f.audio)
	}
	if !strings.Contains(e.err.String(), "live") {
		t.Errorf("stderr = %q", e.err)
	}
}

func TestTooLongVideoIsRefused(t *testing.T) { // v1.1, shield S5
	e := newEnv(t, "youtube-pt-nosubs.json")
	e.f.video.Duration = 5 * time.Hour
	if code := e.run(ptURL); code != ExitRuntime || e.f.audio != 0 {
		t.Fatalf("exit %d audio %d", code, e.f.audio)
	}
}

func TestDownloadsArePinnedToProbedVideo(t *testing.T) { // v1.1, shield re-review
	e := newEnv(t, "youtube-pt-nosubs.json")
	if code := e.run(ptURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	e2 := newEnv(t, "youtube-ted-ed.json")
	if code := e2.run(tedURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e2.err)
	}
	if got := append(e.f.expectIDs, e2.f.expectIDs...); len(got) != 2 || got[0] != "MKU9suCNJQo" || got[1] != "qD0_yWgifDM" {
		t.Errorf("expected IDs = %q", got)
	}
}

func TestWhisperPromptHasNames(t *testing.T) { // v1.1
	e := newEnv(t, "x-poteto.json")
	if code := e.run(xURL); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	if e.f.prompt != "@poteto" {
		t.Errorf("X prompt = %q", e.f.prompt)
	}
	e2 := newEnv(t, "youtube-pt-nosubs.json")
	e2.run(ptURL)
	if want := "MISTÉRIOS do UNIVERSO - MELHORES MOMENTOS - SÉRGIO SACANI. Cortes do Inteligência [OFICIAL]."; e2.f.prompt != want {
		t.Errorf("YouTube prompt = %q, want %q", e2.f.prompt, want)
	}
}

func TestWhisperPromptDropsControlCharacters(t *testing.T) {
	e := newEnv(t, "x-poteto.json")
	e.f.video.Author = "po\nte\x1bto"
	e.run(xURL)
	if e.f.prompt != "@po te to" {
		t.Errorf("prompt = %q", e.f.prompt)
	}
}
