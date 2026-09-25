package app

import (
	"bytes"
	"context"
	"testing"
)

func TestRunWithoutArgsIsUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(context.Background(), nil, Deps{Stdout: &out, Stderr: &errb})
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if errb.Len() == 0 {
		t.Fatal("expected usage message on stderr")
	}
}
