package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const tailLines = 6

// Error is returned when a program exits with an error.
// Output holds the last lines it wrote to stderr.
type Error struct {
	Program string
	Err     error
	Output  []string
}

func (e *Error) Error() string {
	if len(e.Output) == 0 {
		return fmt.Sprintf("%s: %v", e.Program, e.Err)
	}
	return fmt.Sprintf("%s: %v\n%s", e.Program, e.Err, strings.Join(e.Output, "\n"))
}

func (e *Error) Unwrap() error { return e.Err }

// Run starts a program and calls onLine for every line it prints (stdout and stderr).
// Canceling ctx kills the program and every child it started (like ffmpeg under yt-dlp).
func Run(ctx context.Context, onLine func(line string), name string, args ...string) error {

	if onLine == nil {
		onLine = func(string) {}
	}

	cmd := exec.CommandContext(ctx, name, args...)

	// own process group, so cancel can kill the children too
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 2 * time.Second

	var mu sync.Mutex
	var tail []string

	stdout := &lineWriter{mu: &mu, onLine: onLine}
	stderr := &lineWriter{mu: &mu, onLine: func(line string) {
		if len(line) > 300 {
			line = line[:300] + "…"
		}
		tail = append(tail, line)
		if len(tail) > tailLines {
			tail = tail[1:]
		}
		onLine(line)
	}}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	stdout.flush()
	stderr.flush()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return &Error{Program: name, Err: err, Output: tail}
	}
	return nil
}

// lineWriter splits output into lines. yt-dlp and ffmpeg also use \r.
type lineWriter struct {
	mu     *sync.Mutex
	onLine func(string)
	buf    []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexAny(w.buf, "\r\n")
		if i < 0 {
			break
		}
		w.emit(w.buf[:i])
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

func (w *lineWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.emit(w.buf)
	w.buf = nil
}

func (w *lineWriter) emit(b []byte) {
	if line := strings.TrimSpace(string(b)); line != "" {
		w.onLine(line)
	}
}
