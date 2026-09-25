// Command clipscribe transcribes a YouTube or X video into an Obsidian clipping.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/phant0um/clipscribe/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := app.Run(ctx, os.Args[1:], app.DefaultDeps())
	stop()
	os.Exit(code)
}
