package app

import (
	"os"
	"os/exec"
	"time"

	"github.com/phant0um/clipscribe/internal/tools"
)

// DefaultDeps wires the real binaries. The model path comes from config;
// a config error is reported later by Run.
func DefaultDeps() Deps {
	cfgPath := DefaultConfigPath()
	cfg, _ := LoadConfig(cfgPath)
	return Deps{
		Downloader:  tools.NewYtDlp("yt-dlp"),
		Converter:   tools.NewFFmpeg("ffmpeg"),
		Transcriber: tools.NewWhisper("whisper-cli", cfg.ModelPath),
		LookPath:    exec.LookPath,
		Now:         time.Now,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		ConfigPath:  cfgPath,
	}
}
