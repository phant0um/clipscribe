// Package media knows the supported platforms, their URLs and the
// metadata that yt-dlp reports for them.
package media

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Platform identifies a supported video source.
type Platform string

const (
	YouTube Platform = "youtube"
	X       Platform = "x"
)

// Target is a validated video URL.
type Target struct {
	Platform Platform
	ID       string // YouTube video ID or X post (status) ID
	URL      string // normalized URL passed to yt-dlp
}

// ErrUnsupportedURL is returned for any URL outside the allowlist.
var ErrUnsupportedURL = errors.New("unsupported URL")

var (
	youtubeHosts = map[string]bool{"youtube.com": true, "www.youtube.com": true, "m.youtube.com": true}
	xHosts       = map[string]bool{"x.com": true, "twitter.com": true, "mobile.twitter.com": true, "www.x.com": true, "www.twitter.com": true}
	youtubeID    = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	xStatusPath  = regexp.MustCompile(`^/([A-Za-z0-9_]{1,15})/status/([0-9]{1,20})(?:/video/[0-9]+)?/?$`)
)

// ParseURL validates raw against the allowlist and extracts the video ID.
// Hosts are compared exactly; only https without userinfo is accepted.
func ParseURL(raw string) (Target, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return Target{}, fmt.Errorf("%w: %q", ErrUnsupportedURL, raw)
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "youtu.be":
		return youtubeTarget(strings.Trim(u.Path, "/"), raw)
	case youtubeHosts[host]:
		if u.Path == "/watch" {
			return youtubeTarget(u.Query().Get("v"), raw)
		}
		if id, ok := strings.CutPrefix(u.Path, "/shorts/"); ok {
			return youtubeTarget(strings.Trim(id, "/"), raw)
		}
	case xHosts[host]:
		if m := xStatusPath.FindStringSubmatch(u.Path); m != nil {
			return Target{Platform: X, ID: m[2], URL: "https://x.com/" + m[1] + "/status/" + m[2]}, nil
		}
	}
	return Target{}, fmt.Errorf("%w: %q", ErrUnsupportedURL, raw)
}

func youtubeTarget(id, raw string) (Target, error) {
	if !youtubeID.MatchString(id) {
		return Target{}, fmt.Errorf("%w: %q", ErrUnsupportedURL, raw)
	}
	return Target{Platform: YouTube, ID: id, URL: "https://www.youtube.com/watch?v=" + id}, nil
}
