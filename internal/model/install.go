// Package model downloads the whisper model and verifies it.
package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Spec pins one model file by URL, size and SHA-256.
type Spec struct {
	URL    string
	SHA256 string
	Size   int64
}

// LargeV3Turbo is the model chosen in ADR-0002. Hash and size come from
// the Hugging Face API (see spec 001 research.md).
var LargeV3Turbo = Spec{
	URL:    "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin",
	SHA256: "1fc70f774d38eb169993ac391eea357ef47c88757ef72ee5943879b7e8e2bc69",
	Size:   1624555275,
}

// Install downloads spec to dest. The file only appears at dest after its
// size and SHA-256 match; otherwise the partial download is removed.
func Install(ctx context.Context, client *http.Client, spec Spec, dest string) error {
	if !strings.HasPrefix(spec.URL, "https://") {
		return fmt.Errorf("refusing non-https model URL")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download model: HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".model-*.part")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	h := sha256.New()
	// Read one byte past the expected size to detect oversized bodies.
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(resp.Body, spec.Size+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("download model: %w", err)
	}
	if n != spec.Size {
		return fmt.Errorf("model size is %d bytes, want %d", n, spec.Size)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != spec.SHA256 {
		return fmt.Errorf("model SHA-256 mismatch: got %s, want %s", got, spec.SHA256)
	}
	return os.Rename(tmp.Name(), dest)
}
