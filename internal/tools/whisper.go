package tools

import (
	"context"
	"os"
	"path/filepath"

	"github.com/phant0um/clipscribe/internal/transcript"
)

// Whisper transcribes audio with whisper-cli (whisper.cpp).
type Whisper struct {
	Bin   string
	Model string
	run   runner
}

// NewWhisper returns an adapter that runs the real binary.
func NewWhisper(bin, model string) Whisper { return Whisper{Bin: bin, Model: model, run: execRun} }

// Transcribe runs whisper-cli on wav. lang is "auto", "pt" or "en"; it is
// always passed explicitly because whisper-cli defaults to English.
// prompt, when not empty, biases the spelling of names in the audio.
func (w Whisper) Transcribe(ctx context.Context, wav, lang, prompt string) ([]transcript.Segment, string, error) {
	base := filepath.Join(filepath.Dir(wav), "whisper")
	args := []string{"-m", w.Model, "-f", wav, "-l", lang, "-oj", "-of", base, "-np"}
	if prompt != "" {
		args = append(args, "--prompt", prompt)
	}
	if _, err := w.run(ctx, w.Bin, args...); err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(base + ".json")
	if err != nil {
		return nil, "", err
	}
	return transcript.ParseWhisperJSON(b)
}
