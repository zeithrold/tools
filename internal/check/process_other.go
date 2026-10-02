//go:build !unix

package check

import "os/exec"

// CommandContext kills the immediate process on these platforms. Projects with
// long-lived child servers should use their native runner's cleanup facility.
func configureProcess(command *exec.Cmd) {}
