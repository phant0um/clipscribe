// Package tools runs the external binaries clipscribe orchestrates:
// yt-dlp, ffmpeg and whisper-cli. Arguments are always passed as a slice,
// never through a shell.
package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const maxStderr = 64 << 10

// runner executes bin with args and returns stdout.
type runner func(ctx context.Context, bin string, args ...string) ([]byte, error)

// ExitError is a failed run with the tail of its stderr.
type ExitError struct {
	Bin    string
	Stderr string
	Err    error
}

func (e *ExitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if i := strings.LastIndex(msg, "\n"); i >= 0 {
		msg = msg[i+1:] // last line carries the actual error
	}
	return fmt.Sprintf("%s failed: %s", e.Bin, msg)
}

func (e *ExitError) Unwrap() error { return e.Err }

// limitedBuffer keeps only the last maxStderr bytes.
type limitedBuffer struct{ b []byte }

func (l *limitedBuffer) Write(p []byte) (int, error) {
	l.b = append(l.b, p...)
	if len(l.b) > maxStderr {
		l.b = l.b[len(l.b)-maxStderr:]
	}
	return len(p), nil
}

func execRun(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.WaitDelay = 5 * time.Second // do not hang on orphaned pipes after cancel
	var stdout bytes.Buffer
	var stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &ExitError{Bin: bin, Stderr: string(stderr.b), Err: err}
	}
	return stdout.Bytes(), nil
}
