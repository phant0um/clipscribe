package tools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phant0um/clipscribe/internal/media"
)

// YtDlp downloads metadata, captions and audio with yt-dlp.
type YtDlp struct {
	Bin string
	run runner
}

// NewYtDlp returns an adapter that runs the real binary.
func NewYtDlp(bin string) YtDlp { return YtDlp{Bin: bin, run: execRun} }

var authMarkers = []string{"sign in to confirm", "requires authentication", "login required", "log in", "private video", "protected"}

func (y YtDlp) call(ctx context.Context, url string, o media.FetchOpts, extra ...string) ([]byte, error) {
	args := []string{"--ignore-config", "--no-playlist", "--no-progress"}
	if o.Cookies != "" {
		args = append(args, "--cookies", o.Cookies)
	}
	args = append(args, extra...)
	args = append(args, "--", url)
	out, err := y.run(ctx, y.Bin, args...)
	if err == nil {
		return out, nil
	}
	var ee *ExitError
	if !errors.As(err, &ee) {
		return nil, err
	}
	low := strings.ToLower(ee.Stderr)
	switch {
	case strings.Contains(low, "no video could be found"):
		return nil, media.ErrNoVideo
	case containsAny(low, authMarkers):
		return nil, media.ErrAuthRequired
	}
	if o.Cookies != "" {
		ee = &ExitError{Bin: ee.Bin, Stderr: strings.ReplaceAll(ee.Stderr, o.Cookies, "<cookie file>"), Err: ee.Err}
	}
	return nil, ee
}

// Probe reads the video metadata.
func (y YtDlp) Probe(ctx context.Context, url string, o media.FetchOpts) (media.Video, error) {
	out, err := y.call(ctx, url, o, "--dump-single-json")
	if err != nil {
		return media.Video{}, err
	}
	return media.ParseVideo(out)
}

// FetchCaptions downloads the manual caption lang as WebVTT into dir.
func (y YtDlp) FetchCaptions(ctx context.Context, url, lang, dir string, o media.FetchOpts) (string, error) {
	_, err := y.call(ctx, url, o, "--skip-download", "--write-subs", "--no-write-auto-subs",
		"--sub-langs", lang, "--sub-format", "vtt", "-o", filepath.Join(dir, "%(id)s.%(ext)s"))
	if err != nil {
		return "", err
	}
	return single(filepath.Join(dir, "*.vtt"))
}

// FetchAudio downloads the best audio stream into dir.
func (y YtDlp) FetchAudio(ctx context.Context, url, dir string, o media.FetchOpts) (string, error) {
	if _, err := y.call(ctx, url, o, "-f", "bestaudio/best", "-o", filepath.Join(dir, "audio.%(ext)s")); err != nil {
		return "", err
	}
	return single(filepath.Join(dir, "audio.*"))
}

func single(pattern string) (string, error) {
	matches, _ := filepath.Glob(pattern)
	var found []string
	for _, m := range matches {
		if !strings.HasSuffix(m, ".part") && !strings.HasSuffix(m, ".ytdl") {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		return "", fmt.Errorf("yt-dlp produced %d files matching %s", len(found), filepath.Base(pattern))
	}
	return found[0], nil
}

func containsAny(s string, subs []string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}
