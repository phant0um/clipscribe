package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"clipscribe/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := app.Run(ctx, os.Args[1:], app.Deps{Stdout: os.Stdout, Stderr: os.Stderr})
	stop()
	os.Exit(code)
}
