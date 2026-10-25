package runner

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRun_Lines(t *testing.T) {

	var lines []string
	err := Run(context.Background(), func(l string) { lines = append(lines, l) },
		"sh", "-c", `printf 'a\nb\rc\n'; echo d >&2; printf e`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slices.Sort(lines) // stdout and stderr can interleave
	if want := []string{"a", "b", "c", "d", "e"}; !slices.Equal(lines, want) {
		t.Errorf("lines = %v, want %v", lines, want)
	}
}

func TestRun_ErrorHasStderrTail(t *testing.T) {

	err := Run(context.Background(), nil, "sh", "-c", `echo ok; echo "ERROR: bad link" >&2; exit 3`)

	var runErr *Error
	if !errors.As(err, &runErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if !slices.Equal(runErr.Output, []string{"ERROR: bad link"}) {
		t.Errorf("Output = %v", runErr.Output)
	}
	if !strings.Contains(err.Error(), "ERROR: bad link") {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestRun_CancelKillsChildren(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	pidCh := make(chan int, 1)

	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := Run(ctx, func(l string) {
		if pid, err := strconv.Atoi(l); err == nil {
			pidCh <- pid
		}
	}, "sh", "-c", `sleep 30 & echo $!; wait`)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("Run took %v after cancel", d)
	}

	child := <-pidCh
	time.Sleep(100 * time.Millisecond)
	if syscall.Kill(child, 0) == nil {
		syscall.Kill(child, syscall.SIGKILL)
		t.Errorf("child process %d is still running", child)
	}
}
