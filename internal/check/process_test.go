package check

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessHelper(t *testing.T) {
	if os.Getenv("ZT_PROCESS_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "pass":
		_, _ = os.Stdout.WriteString("stdout evidence\n")
		_, _ = os.Stderr.WriteString("stderr evidence\n")
		os.Exit(0)
	case "fail":
		_, _ = os.Stdout.WriteString("failure evidence\n")
		os.Exit(7)
	case "large":
		chunk := bytes.Repeat([]byte("x"), 64<<10)
		for i := 0; i < MaxProcessLogBytes/(64<<10)+2; i++ {
			_, _ = os.Stdout.Write(chunk)
		}
		os.Exit(0)
	case "check-env":
		if os.Getenv("ZT_PROCESS_PARENT_SECRET") != "" {
			os.Exit(5)
		}
		os.Exit(0)
	case "stream":
		_, _ = os.Stdout.WriteString("stdout before exit\n")
		_, _ = os.Stderr.WriteString("stderr before exit\n")
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
			if _, err := os.Stat(os.Getenv("ZT_PROCESS_RELEASE")); err == nil {
				os.Exit(0)
			}
			time.Sleep(5 * time.Millisecond)
		}
		os.Exit(9)
	default:
		os.Exit(8)
	}
}

func processHelperRequest(t *testing.T, logPath, mode string) ProcessRequest {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return ProcessRequest{
		Argv:    []string{executable, "-test.run=^TestProcessHelper$", "--", mode},
		Dir:     t.TempDir(),
		LogPath: logPath,
		Env:     append(os.Environ(), "ZT_PROCESS_HELPER=1"),
		Timeout: 5 * time.Second,
	}
}

func TestRunProcessPassesAndCapturesCombinedOutput(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "process.log")
	result := RunProcess(t.Context(), processHelperRequest(t, logPath, "pass"))
	if result.Status != "passed" || result.ExitCode != 0 || result.Detail != "" {
		t.Fatalf("result = %+v, want successful process", result)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "stdout evidence") || !strings.Contains(string(data), "stderr evidence") {
		t.Fatalf("combined log = %q", data)
	}
}

type processOutputFunc func([]byte) (int, error)

func (f processOutputFunc) Write(data []byte) (int, error) { return f(data) }

func TestRunProcessStreamsBothChannelsBeforeExit(t *testing.T) {
	directory := t.TempDir()
	request := processHelperRequest(t, filepath.Join(directory, "stream.log"), "stream")
	release := filepath.Join(directory, "release")
	request.Env = append(request.Env, "ZT_PROCESS_RELEASE="+release)
	var live bytes.Buffer
	request.Output = processOutputFunc(func(data []byte) (int, error) {
		n, err := live.Write(data)
		if strings.Contains(live.String(), "stdout before exit") && strings.Contains(live.String(), "stderr before exit") {
			err = os.WriteFile(release, nil, 0600)
		}
		return n, err
	})
	result := RunProcess(t.Context(), request)
	if result.Status != "passed" {
		t.Fatalf("child could not observe live output before exiting: %+v", result)
	}
	log, err := os.ReadFile(request.LogPath)
	if err != nil || string(log) != live.String() {
		t.Fatalf("log %q differs from live output %q: %v", log, live.String(), err)
	}
}

func TestRunProcessNonZeroExitIsFailed(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "process.log")
	result := RunProcess(t.Context(), processHelperRequest(t, logPath, "fail"))
	if result.Status != "failed" || result.ExitCode != 7 || result.Detail == "" {
		t.Fatalf("result = %+v, want failed process with exit code 7", result)
	}
}

func TestRunProcessMissingExecutableIsBlocked(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "missing.log")
	result := RunProcess(t.Context(), ProcessRequest{
		Argv:    []string{"zt-process-command-that-does-not-exist"},
		LogPath: logPath,
		Env:     os.Environ(),
		Timeout: time.Second,
	})
	if result.Status != "blocked" || result.ExitCode != -1 || result.Detail == "" {
		t.Fatalf("result = %+v, want actionable blocked result without a process state", result)
	}
}

func TestRunProcessTimeoutIsFailed(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is unavailable")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep is unavailable")
	}
	result := RunProcess(t.Context(), ProcessRequest{
		Argv:    []string{shell, "-c", "exec sleep 5"},
		Dir:     t.TempDir(),
		LogPath: filepath.Join(t.TempDir(), "timeout.log"),
		Env:     os.Environ(),
		Timeout: 100 * time.Millisecond,
	})
	if result.Status != "failed" || !strings.Contains(result.Detail, "deadline") {
		t.Fatalf("timeout result = %+v, want failed with context deadline detail", result)
	}
}

func TestRunProcessBoundsAndMarksTruncatedLog(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "large.log")
	request := processHelperRequest(t, logPath, "large")
	var live bytes.Buffer
	request.Output = &live
	result := RunProcess(t.Context(), request)
	if result.Status != "passed" {
		t.Fatalf("result = %+v, want passed", result)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), processLogTruncationMarker) {
		t.Fatalf("log does not end with truncation marker: final bytes %q", data[len(data)-min(len(data), len(processLogTruncationMarker)):])
	}
	if len(data) > MaxProcessLogBytes+len(processLogTruncationMarker) {
		t.Fatalf("log size = %d, exceeds bound %d", len(data), MaxProcessLogBytes+len(processLogTruncationMarker))
	}
	if len(data) != MaxProcessLogBytes+len(processLogTruncationMarker) {
		t.Fatalf("log size = %d, want the full limit and marker (%d)", len(data), MaxProcessLogBytes+len(processLogTruncationMarker))
	}
	if live.Len() != MaxProcessLogBytes+2*(64<<10) || bytes.Contains(live.Bytes(), []byte(processLogTruncationMarker)) {
		t.Fatalf("live output was truncated or contains the retained-log marker: %d bytes", live.Len())
	}
}

func TestRunProcessEnvironmentReplacesParent(t *testing.T) {
	t.Setenv("ZT_PROCESS_PARENT_SECRET", "parent-only")
	request := processHelperRequest(t, filepath.Join(t.TempDir(), "env.log"), "check-env")
	request.Env = []string{"ZT_PROCESS_HELPER=1"}
	result := RunProcess(t.Context(), request)
	if result.Status != "passed" || result.ExitCode != 0 {
		t.Fatalf("result = %+v, child inherited the parent's environment", result)
	}
}

func TestRunProcessDoesNotInterpretCommandNameAsShell(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, "shell-injected")
	commandName := fmt.Sprintf("printf; touch %q", marker)
	result := RunProcess(t.Context(), ProcessRequest{
		Argv:    []string{commandName},
		Dir:     directory,
		LogPath: filepath.Join(directory, "inert.log"),
		Env:     os.Environ(),
		Timeout: time.Second,
	})
	if result.Status != "blocked" || result.ExitCode != -1 || result.Detail == "" {
		t.Fatalf("result = %+v, command name should be unresolved rather than run by a shell", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("shell syntax created marker: %v", err)
	}
}

func TestRunProcessValidationDoesNotCreateLog(t *testing.T) {
	for _, test := range []struct {
		name    string
		request func(string) ProcessRequest
	}{
		{"empty argv", func(log string) ProcessRequest {
			return ProcessRequest{LogPath: log, Timeout: time.Second}
		}},
		{"empty executable", func(log string) ProcessRequest {
			return ProcessRequest{Argv: []string{""}, LogPath: log, Timeout: time.Second}
		}},
		{"non-positive timeout", func(log string) ProcessRequest {
			return ProcessRequest{Argv: []string{"true"}, LogPath: log}
		}},
		{"empty log path", func(string) ProcessRequest {
			return ProcessRequest{Argv: []string{"true"}, Timeout: time.Second}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			logPath := filepath.Join(directory, "invalid.log")
			result := RunProcess(t.Context(), test.request(logPath))
			if result.Status != "blocked" || result.Detail == "" {
				t.Fatalf("result = %+v, want actionable blocked validation result", result)
			}
			if _, err := os.Stat(logPath); !os.IsNotExist(err) {
				t.Fatalf("invalid request created a log: %v", err)
			}
		})
	}
}
