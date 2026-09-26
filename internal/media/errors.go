package media

import "errors"

// Errors reported by a Downloader, so callers can give a useful next step.
var (
	ErrAuthRequired = errors.New("video requires login")
	ErrNoVideo      = errors.New("post has no video")
)

// FetchOpts are per-run options for a downloader.
type FetchOpts struct {
	Cookies string // path of a dedicated Netscape cookie file, or ""
	// ExpectID pins a download to the video the probe returned. yt-dlp
	// skips the download if the ID differs or the video is live.
	ExpectID string
}
