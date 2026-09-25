package app

import (
	"context"
	"fmt"
)

// runDoctor is implemented in phase 5 (T060-T061).
func runDoctor(ctx context.Context, args []string, d Deps) int {
	fmt.Fprintln(d.Stderr, "clipscribe: doctor not implemented yet")
	return ExitRuntime
}
