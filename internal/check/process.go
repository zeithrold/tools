package check

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const MaxProcessLogBytes = 1 << 20

const processLogTruncationMarker = "\n[zt: log truncated at 1 MiB]\n"

// ProcessRequest is one reviewed native command. Argv is executed directly; no
// shell is involved, so shell syntax in a name is inert.
type ProcessRequest struct {
	Argv    []string      // Argv[0] is resolved by exec.LookPath semantics
	Dir     string        // working directory
	LogPath string        // created with O_CREATE|O_EXCL; stdout and stderr both go here
	Env     []string      // complete child environment, not merged with os.Environ
	Timeout time.Duration // per command, covering the process group on Unix
	Output  io.Writer     // optional live combined stdout/stderr; independent of the log cap
}

// ProcessResult is what zt observed, never what the command claimed.
type ProcessResult struct {
	Status     string // "passed", "failed" or "blocked"
	ExitCode   int    // -1 when no process state exists
	DurationMS int64
	Detail     string
}

// RunProcess executes one native command without a shell and retains at most
// MaxProcessLogBytes of its combined stdout and stderr.
func RunProcess(ctx context.Context, request ProcessRequest) ProcessResult {
	result := ProcessResult{Status: "blocked", ExitCode: -1}
	switch {
	case len(request.Argv) == 0:
		result.Detail = "command argv must contain an executable"
		return result
	case request.Argv[0] == "":
		result.Detail = "command executable must not be empty"
		return result
	case request.Timeout <= 0:
		result.Detail = "command timeout must be positive"
		return result
	case request.LogPath == "":
		result.Detail = "command log path must not be empty"
		return result
	}

	log, err := os.OpenFile(request.LogPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer log.Close()

	childContext, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	command := exec.CommandContext(childContext, request.Argv[0], request.Argv[1:]...)
	command.Dir = request.Dir
	command.Env = request.Env
	output := &limitedProcessLog{file: log}
	var stream io.Writer = output
	if request.Output != nil {
		stream = io.MultiWriter(output, request.Output)
	}
	command.Stdout, command.Stderr = stream, stream
	command.WaitDelay = 2 * time.Second
	configureProcess(command)

	start := time.Now()
	err = command.Run()
	result.DurationMS = time.Since(start).Milliseconds()
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	logErr := output.finish()

	switch {
	case childContext.Err() != nil:
		result.Status = "failed"
		result.Detail = childContext.Err().Error()
	case err == nil && logErr == nil:
		result.Status = "passed"
	case command.ProcessState != nil:
		result.Status = "failed"
		if err != nil {
			result.Detail = err.Error()
		} else {
			result.Detail = "could not finish command log: " + logErr.Error()
		}
	default:
		// No ProcessState means exec could not start the command. That is a
		// blocked check rather than a failed native command.
		result.Status = "blocked"
		result.Detail = err.Error()
	}
	return result
}

type limitedProcessLog struct {
	mu        sync.Mutex
	file      *os.File
	written   int64
	truncated bool
}

func (w *limitedProcessLog) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	remaining := MaxProcessLogBytes - w.written
	if remaining <= 0 {
		if len(data) > 0 {
			w.truncated = true
		}
		return len(data), nil
	}

	toWrite := len(data)
	if int64(toWrite) > remaining {
		toWrite = int(remaining)
	}
	written, err := w.file.Write(data[:toWrite])
	w.written += int64(written)
	if err != nil {
		return written, err
	}
	if written != toWrite {
		return written, io.ErrShortWrite
	}
	if toWrite < len(data) {
		w.truncated = true
	}
	// Bytes past the cap are intentionally discarded, not an output error.
	return len(data), nil
}

func (w *limitedProcessLog) finish() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.truncated {
		return nil
	}
	written, err := io.WriteString(w.file, processLogTruncationMarker)
	if err == nil && written != len(processLogTruncationMarker) {
		return io.ErrShortWrite
	}
	return err
}
