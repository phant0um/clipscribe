package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/phant0um/clipscribe/internal/model"
)

// runDoctor reports missing dependencies and optionally installs the model.
func runDoctor(ctx context.Context, args []string, d Deps) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	install := fs.Bool("install-model", false, "download and verify the whisper model")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		fmt.Fprint(d.Stderr, usage)
		return ExitUsage
	}
	cfg, err := LoadConfig(d.ConfigPath)
	if err != nil {
		fmt.Fprintf(d.Stderr, "clipscribe: %v\n", err)
		return ExitUsage
	}
	if *install {
		if code := installModel(ctx, d, cfg.ModelPath); code != ExitOK {
			return code
		}
	}

	code := ExitOK
	for _, bin := range requiredBinaries {
		if p, err := d.LookPath(bin); err == nil {
			fmt.Fprintf(d.Stdout, "ok       %-12s %s\n", bin, p)
		} else {
			fmt.Fprintf(d.Stdout, "missing  %-12s fix: brew install %s\n", bin, brewPackage(bin))
			code = ExitMissing
		}
	}
	if _, err := os.Stat(cfg.ModelPath); err == nil {
		fmt.Fprintf(d.Stdout, "ok       %-12s %s\n", "model", cfg.ModelPath)
	} else {
		fmt.Fprintf(d.Stdout, "missing  %-12s fix: clipscribe doctor --install-model\n", "model")
		code = ExitMissing
	}
	if cfg.OutDir == "" {
		fmt.Fprintf(d.Stdout, "warning  %-12s no out_dir in %s; md output needs --out\n", "config", d.ConfigPath)
	}
	return code
}

func installModel(ctx context.Context, d Deps, dest string) int {
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(d.Stderr, "model already installed at %s\n", dest)
		return ExitOK
	}
	spec := d.Model
	if spec == (model.Spec{}) {
		spec = model.LargeV3Turbo
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	fmt.Fprintf(d.Stderr, "downloading model (%d MB) and verifying SHA-256\n", spec.Size>>20)
	if err := model.Install(ctx, client, spec, dest); err != nil {
		return fail(d, err)
	}
	fmt.Fprintf(d.Stderr, "model installed at %s\n", dest)
	return ExitOK
}
