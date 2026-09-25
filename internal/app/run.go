// Package app wires the clipscribe command line to its dependencies.
package app

import (
	"context"
	"fmt"
	"io"
)

// Exit codes, as defined in the constitution.
const (
	ExitOK      = 0
	ExitRuntime = 1
	ExitUsage   = 2
	ExitMissing = 3
)

const usage = `usage: clipscribe <url> [--lang auto|pt|en] [--format md|srt|txt] [--out dir] [--cookies file] [--force] [--keep-audio]
       clipscribe doctor [--install-model]
`

// Deps holds everything Run needs from the outside world.
type Deps struct {
	Stdout, Stderr io.Writer
}

// Run executes clipscribe and returns the process exit code.
func Run(ctx context.Context, args []string, d Deps) int {
	if len(args) == 0 {
		fmt.Fprint(d.Stderr, usage)
		return ExitUsage
	}
	return ExitOK
}
