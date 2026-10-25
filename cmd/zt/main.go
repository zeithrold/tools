package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
)

func main() {
	// One signal context owns cancellation for every command, so Ctrl+C stops
	// running checks and pending device-flow polling through cmd.Context().
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := execute(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "zt:", err)
		os.Exit(1)
	}
}

// execute builds the CLI against explicit streams and a fresh process
// environment. It never reads os.Args, so tests drive the CLI through
// executeApp with injected dependencies instead of mutating global state.
func execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	return executeApp(newApp(stdout, stderr), ctx, args)
}
