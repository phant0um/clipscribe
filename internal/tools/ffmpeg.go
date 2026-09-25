package tools

import "context"

// FFmpeg converts audio with ffmpeg.
type FFmpeg struct {
	Bin string
	run runner
}

// NewFFmpeg returns an adapter that runs the real binary.
func NewFFmpeg(bin string) FFmpeg { return FFmpeg{Bin: bin, run: execRun} }

// ToWAV16k converts in to 16 kHz mono PCM WAV, the input whisper expects.
func (f FFmpeg) ToWAV16k(ctx context.Context, in, out string) error {
	_, err := f.run(ctx, f.Bin, "-nostdin", "-hide_banner", "-loglevel", "error",
		"-i", in, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-y", out)
	return err
}
