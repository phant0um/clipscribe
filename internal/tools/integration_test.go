//go:build integration

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/phant0um/clipscribe/internal/media"
)

// Run with: go test -tags integration ./internal/tools
// Needs network and yt-dlp. Checks that the real JSON still matches the parser.
func TestRealProbe(t *testing.T) {
	cases := []struct {
		url      string
		platform media.Platform
		id       string
	}{
		{"https://www.youtube.com/watch?v=qD0_yWgifDM", media.YouTube, "qD0_yWgifDM"},
		{"https://x.com/poteto/status/2102050467505430555", media.X, "2102050467505430555"},
	}
	y := NewYtDlp("yt-dlp")
	for _, c := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		v, err := y.Probe(ctx, c.url, media.FetchOpts{})
		cancel()
		if err != nil {
			t.Errorf("%s: %v", c.url, err)
			continue
		}
		if v.Platform != c.platform || v.ID != c.id || v.Title == "" || v.Author == "" || v.Duration == 0 {
			t.Errorf("%s: %+v", c.url, v)
		}
	}
}
