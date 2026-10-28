package main

import (
	"context"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/zeithrold/tools/internal/githubauth"
)

// app carries the process dependencies a command may use. Building it performs
// no I/O: it does not resolve the home directory, read credentials, or open a
// network connection. Commands opt into those effects explicitly, which keeps
// inspect, plan and sync --plan read-only and lets tests inject fakes.
type app struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader

	// noInput and noColor mirror the --no-input and --no-color global flags.
	// Human output is currently plain text, so --no-color has nothing to strip;
	// it is accepted so scripts can set it unconditionally.
	noInput bool
	noColor bool

	// homeDir resolves the user-global tree for skills sync. Only that command
	// calls it, so read-only commands never touch the filesystem outside the
	// project root.
	homeDir func() (string, error)

	// getenv reads the public client id override. It never reads tokens.
	getenv func(string) string

	// isTTY and openBrowser decide whether login may launch a browser. Tests
	// keep isTTY false or pass --no-browser, so no test spawns a process.
	isTTY       func() bool
	openBrowser func(string) error

	// newAuth builds the device-flow service for a host and opens the OS
	// keystore lazily. Only auth subcommands call it, so inspect, plan, check
	// and skills sync never touch the keystore.
	newAuth func(host string) (authService, error)

	// openAPI opens an authenticated, target-bound GitHub API endpoint for the
	// selected host and repository. Only `evolve submit` and `evolve status
	// --remote` call it, so no other command reaches the keystore or the network.
	openAPI func(ctx context.Context, host, clientID, profile, repository string) (githubauth.API, error)

	// openGit is used only by the explicit evolve upstream fetch command.
	// Offline cache discovery and consumption never construct a credential.
	openGit func(ctx context.Context, host, clientID, profile, repository string) (githubauth.GitEndpoint, error)
}

func newApp(stdout, stderr io.Writer) *app {
	a := &app{
		stdout:      stdout,
		stderr:      stderr,
		stdin:       os.Stdin,
		homeDir:     os.UserHomeDir,
		getenv:      os.Getenv,
		openBrowser: openInBrowser,
	}
	a.isTTY = func() bool { return isTerminal(a.stdout) || isTerminal(a.stderr) }
	a.newAuth = newAuthService
	a.openAPI = newOpenAPI
	a.openGit = newOpenGit
	return a
}

// isTerminal reports whether w is a real terminal. ModeCharDevice is not
// enough: /dev/null and NUL are character devices but not terminals. term keeps
// headless runs from launching a browser.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
