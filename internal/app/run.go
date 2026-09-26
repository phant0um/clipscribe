// Package app wires the clipscribe command line to its dependencies.
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phant0um/clipscribe/internal/media"
	"github.com/phant0um/clipscribe/internal/model"
	"github.com/phant0um/clipscribe/internal/transcript"
	"github.com/phant0um/clipscribe/internal/vault"
)

// Exit codes, as defined in the constitution.
const (
	ExitOK      = 0
	ExitRuntime = 1
	ExitUsage   = 2
	ExitMissing = 3
)

const (
	transcriberCaptions = "captions-manual"
	transcriberWhisper  = "whisper-large-v3-turbo"
)

const usage = `usage: clipscribe <url> [--lang auto|pt|en] [--format md|srt|txt] [--out dir] [--cookies file] [--force] [--keep-audio]
       clipscribe doctor [--install-model]
`

// FetchOpts are per-run options for the downloader.
type FetchOpts = media.FetchOpts

// Downloader wraps yt-dlp.
type Downloader interface {
	Probe(ctx context.Context, url string, o FetchOpts) (media.Video, error)
	FetchCaptions(ctx context.Context, url, lang, dir string, o FetchOpts) (string, error)
	FetchAudio(ctx context.Context, url, dir string, o FetchOpts) (string, error)
}

// AudioConverter wraps ffmpeg.
type AudioConverter interface {
	ToWAV16k(ctx context.Context, in, out string) error
}

// Transcriber wraps whisper-cli.
type Transcriber interface {
	Transcribe(ctx context.Context, wav, lang string) ([]transcript.Segment, string, error)
}

// Deps holds everything Run needs from the outside world.
type Deps struct {
	Downloader  Downloader
	Converter   AudioConverter
	Transcriber Transcriber
	LookPath    func(string) (string, error)
	Now         func() time.Time
	Stdout      io.Writer
	Stderr      io.Writer
	ConfigPath  string
	TempDir     string       // parent of the per-run temp dir; "" = system default
	HTTPClient  *http.Client // used by doctor --install-model
	Model       model.Spec   // zero value means model.LargeV3Turbo
}

var requiredBinaries = []string{"yt-dlp", "ffmpeg", "whisper-cli"}

type options struct {
	url, lang, format, out, cookies string
	force, keepAudio                bool
}

// Run executes clipscribe and returns the process exit code.
func Run(ctx context.Context, args []string, d Deps) int {
	if len(args) == 0 {
		fmt.Fprint(d.Stderr, usage)
		return ExitUsage
	}
	if args[0] == "doctor" {
		return runDoctor(ctx, args[1:], d)
	}
	opt, err := parseArgs(args, d.Stderr)
	if err != nil {
		fmt.Fprintf(d.Stderr, "clipscribe: %v\n%s", err, usage)
		return ExitUsage
	}
	target, err := media.ParseURL(opt.url)
	if err != nil {
		fmt.Fprintln(d.Stderr, "clipscribe: only https URLs of youtube.com, youtu.be, x.com and twitter.com videos are supported")
		return ExitUsage
	}
	if opt.cookies != "" {
		if err := checkCookieFile(opt.cookies); err != nil {
			fmt.Fprintf(d.Stderr, "clipscribe: cookie file rejected: %v\n", err)
			return ExitUsage
		}
	}
	cfg, err := LoadConfig(d.ConfigPath)
	if err != nil {
		fmt.Fprintf(d.Stderr, "clipscribe: %v\n", err)
		return ExitUsage
	}
	outDir := cfg.OutDir
	if opt.out != "" {
		outDir = opt.out
	} else if opt.format != "md" {
		outDir = "."
	}
	if outDir == "" {
		fmt.Fprintf(d.Stderr, "clipscribe: no output directory; set \"out_dir\" in %s or pass --out\n", d.ConfigPath)
		return ExitUsage
	}
	if code := checkDeps(d, cfg); code != ExitOK {
		return code
	}
	return transcribe(ctx, d, cfg, opt, target, outDir)
}

func transcribe(ctx context.Context, d Deps, cfg Config, opt options, target media.Target, outDir string) int {
	fo := FetchOpts{Cookies: opt.cookies}
	fmt.Fprintf(d.Stderr, "reading metadata for %s %s\n", target.Platform, target.ID)
	v, err := d.Downloader.Probe(ctx, target.URL, fo)
	if err != nil {
		return fail(d, err)
	}

	existing := ""
	if opt.format == "md" {
		p, found, err := vault.Find(cfg.DedupDirs, v.Platform, v.ID)
		if err != nil {
			return fail(d, fmt.Errorf("search existing clippings: %w", err))
		}
		if found && !opt.force {
			fmt.Fprintf(d.Stderr, "already transcribed; use --force to replace\n")
			fmt.Fprintln(d.Stdout, p)
			return ExitOK
		}
		if found && within(p, outDir) {
			existing = p // replace in place only inside out_dir, never in the archive
		}
	}

	tmp, err := os.MkdirTemp(d.TempDir, "clipscribe-")
	if err != nil {
		return fail(d, err)
	}
	if opt.keepAudio {
		defer fmt.Fprintf(d.Stderr, "audio kept: %s\n", filepath.Join(tmp, "audio.wav"))
	} else {
		defer os.RemoveAll(tmp)
	}

	doc := transcript.Doc{Video: v, Created: d.Now()}
	spoken := v.Language
	if opt.lang != "auto" {
		spoken = opt.lang
	}
	if code := v.CaptionFor(spoken); v.Platform == media.YouTube && code != "" {
		fmt.Fprintf(d.Stderr, "using manual captions (%s)\n", code)
		doc.Segments, err = captions(ctx, d, target.URL, code, tmp, fo)
		doc.Lang, doc.Transcriber = spoken, transcriberCaptions
	} else {
		doc.Segments, doc.Lang, doc.AudioSHA256, err = whisper(ctx, d, target.URL, opt.lang, tmp, fo)
		doc.Transcriber = transcriberWhisper
	}
	if err != nil {
		return fail(d, err)
	}
	if len(doc.Segments) == 0 {
		return fail(d, errors.New("no speech found"))
	}

	path, err := write(doc, opt.format, outDir, existing)
	if err != nil {
		return fail(d, err)
	}
	fmt.Fprintln(d.Stdout, path)
	return ExitOK
}

func captions(ctx context.Context, d Deps, url, code, dir string, fo FetchOpts) ([]transcript.Segment, error) {
	p, err := d.Downloader.FetchCaptions(ctx, url, code, dir, fo)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return transcript.ParseVTT(b)
}

func whisper(ctx context.Context, d Deps, url, lang, dir string, fo FetchOpts) ([]transcript.Segment, string, string, error) {
	fmt.Fprintln(d.Stderr, "downloading audio")
	audio, err := d.Downloader.FetchAudio(ctx, url, dir, fo)
	if err != nil {
		return nil, "", "", err
	}
	wav := filepath.Join(dir, "audio.wav")
	if err := d.Converter.ToWAV16k(ctx, audio, wav); err != nil {
		return nil, "", "", err
	}
	sum, err := fileSHA256(wav)
	if err != nil {
		return nil, "", "", err
	}
	fmt.Fprintln(d.Stderr, "transcribing")
	segs, detected, err := d.Transcriber.Transcribe(ctx, wav, lang)
	return segs, detected, sum, err
}

func write(doc transcript.Doc, format, outDir, existing string) (string, error) {
	var data []byte
	switch format {
	case "srt":
		data = transcript.RenderSRT(doc.Segments)
	case "txt":
		data = transcript.RenderText(doc.Segments)
	default:
		data = transcript.RenderMarkdown(doc)
	}
	if existing != "" {
		return existing, vault.Replace(existing, data)
	}
	return vault.Save(outDir, vault.FileName(doc.Video), doc.Video.ID, "."+format, data)
}

func parseArgs(args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("clipscribe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.lang, "lang", "auto", "spoken language: auto, pt or en")
	fs.StringVar(&o.format, "format", "md", "output format: md, srt or txt")
	fs.StringVar(&o.out, "out", "", "output directory")
	fs.StringVar(&o.cookies, "cookies", "", "dedicated Netscape cookie file (mode 0600)")
	fs.BoolVar(&o.force, "force", false, "replace an existing clipping")
	fs.BoolVar(&o.keepAudio, "keep-audio", false, "keep the temporary WAV file")
	// Accept flags before and after the URL.
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return o, err
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(positional) != 1 {
		return o, fmt.Errorf("expected exactly one URL, got %d", len(positional))
	}
	o.url = positional[0]
	if o.lang != "auto" && o.lang != "pt" && o.lang != "en" {
		return o, fmt.Errorf("invalid --lang %q", o.lang)
	}
	if o.format != "md" && o.format != "srt" && o.format != "txt" {
		return o, fmt.Errorf("invalid --format %q", o.format)
	}
	return o, nil
}

// checkCookieFile never includes the path in its errors.
func checkCookieFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("cannot read the file")
	}
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("permissions are %#o; run chmod 600 on it", info.Mode().Perm())
	}
	return nil
}

func checkDeps(d Deps, cfg Config) int {
	for _, bin := range requiredBinaries {
		if _, err := d.LookPath(bin); err != nil {
			fmt.Fprintf(d.Stderr, "clipscribe: %s not found; install it with: brew install %s\n", bin, brewPackage(bin))
			return ExitMissing
		}
	}
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		fmt.Fprintln(d.Stderr, "clipscribe: whisper model not found; run: clipscribe doctor --install-model")
		return ExitMissing
	}
	return ExitOK
}

func brewPackage(bin string) string {
	if bin == "whisper-cli" {
		return "whisper-cpp"
	}
	return bin
}

func fail(d Deps, err error) int {
	switch {
	case errors.Is(err, media.ErrAuthRequired):
		fmt.Fprintln(d.Stderr, "clipscribe: this video requires login; retry with --cookies <dedicated cookie file>")
	case errors.Is(err, media.ErrNoVideo):
		fmt.Fprintln(d.Stderr, "clipscribe: this post has no video")
	case errors.Is(err, context.Canceled):
		fmt.Fprintln(d.Stderr, "clipscribe: interrupted; temporary files removed")
	default:
		fmt.Fprintf(d.Stderr, "clipscribe: %v\n", err)
	}
	return ExitRuntime
}

// within reports whether path is inside dir.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
