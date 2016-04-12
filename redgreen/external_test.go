package redgreen_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rhcarvalho/redgreen/redgreen"
)

func TestRunDoneBlockedOnIn(t *testing.T) {
	done := make(chan struct{})
	in := make(chan redgreen.RunSpec)
	out := redgreen.Run(done, in)

	// Test the execution of a single command.
	in <- redgreen.RunSpec{Command: []string{"true"}}
	if res := <-out; res.Error != nil {
		t.Fatalf("got %#v, want nil", res.Error)
	}

	// There is no more input, Run's goroutine should be blocked on
	// receiving from in.

	// Signal that we are done.
	close(done)

	// Ensure that out is eventually closed.
	mustBeClosedTimeout(out, time.Second, t)
}

func TestRunDoneBlockedOnOut(t *testing.T) {
	var outWasOpen, outWasClosed bool
	test := func() {
		done := make(chan struct{})
		in := make(chan redgreen.RunSpec)
		out := redgreen.Run(done, in)

		// Send a single command but don't receive from out, so that
		// Run's goroutine will be blocked on sending to out.
		in <- redgreen.RunSpec{Command: []string{"true"}}

		// Signal that we are done.
		close(done)

		// Ensure that out is eventually closed.
		timeout := time.Second
		select {
		case _, isOpen := <-out:
			// Since we receive from out here, we can observe that
			// the receive operation either succeed or not with
			// equal probability.
			if isOpen {
				outWasOpen = true
			} else {
				outWasClosed = true
			}
		case <-time.After(timeout):
			t.Fatalf("receive from channel timed out")
		}
		mustBeClosedTimeout(out, timeout, t)
	}
	var runs int
	for !(outWasOpen && outWasClosed) {
		// 2 or 3 runs should be enough. The probability of needing more
		// than 16 runs to observe both outWasOpen and outWasClosed is
		// (1/2)^16 ≈ 0.001
		if runs > 16 {
			t.Fatalf("failed to observe both conditions: outWasOpen=%v, outWasClosed=%v", outWasOpen, outWasClosed)
		}
		test()
		runs++
	}
	t.Logf("#runs: %d", runs)
}

func TestRunClosedInput(t *testing.T) {
	done := make(chan struct{})
	in := make(chan redgreen.RunSpec)
	out := redgreen.Run(done, in)

	// Signal that there will be no more input.
	close(in)

	// Ensure that out is eventually closed.
	mustBeClosedTimeout(out, time.Second, t)
}

func mustBeClosedTimeout(ch <-chan redgreen.RunResult, timeout time.Duration, t *testing.T) {
	select {
	case _, isOpen := <-ch:
		if isOpen {
			t.Fatalf("channel is open, want closed")
		}
	case <-time.After(timeout):
		t.Fatalf("receive from channel timed out")
	}
}

func TestWatchBadPath(t *testing.T) {
	done := make(chan struct{})
	path := "does/not/exist"
	_, err := redgreen.Watch(done, path)
	if err == nil {
		t.Fatalf("Watch(done, %q) = %v, want not nil", path, err)
	}
}

func TestWatchDoneBlockedOnIn(t *testing.T) {
	path, err := ioutil.TempDir("", "redgreen")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(path)

	done := make(chan struct{})
	out, err := redgreen.Watch(done, path)
	if err != nil {
		t.Fatalf("Watch(done, %q) = %v, want nil", path, err)
	}

	// No file system events should have been triggered, Watch's goroutine
	// is blocked on receiving events.

	// Signal that we are done.
	close(done)

	// Ensure that out is eventually closed.
	mustBeClosedTimeoutESC(out, time.Second, t)
}

func TestWatchDoneBlockedOnOut(t *testing.T) {
	t.Skip("bad test, fails on Go 1.6")
	var outWasOpen, outWasClosed bool
	test := func() {
		path, err := ioutil.TempDir("", "redgreen")
		if err != nil {
			t.Fatalf("create temp dir: %v", err)
		}
		defer os.RemoveAll(path)

		done := make(chan struct{})
		out, err := redgreen.Watch(done, path)
		if err != nil {
			t.Fatalf("Watch(done, %q) = %v, want nil", path, err)
		}

		// Create a file to trigger a watch event, but never receive from out,
		// so that Watch's goroutine is blocked on the send operation.
		_, err = os.Create(filepath.Join(path, "foo"))
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}

		// Signal that we are done.
		close(done)

		// Ensure that out is eventually closed.
		timeout := time.Second
		select {
		case _, isOpen := <-out:
			// Since we receive from out here, we can observe that
			// the receive operation either succeed or not with
			// equal probability.
			if isOpen {
				outWasOpen = true
			} else {
				outWasClosed = true
			}
		case <-time.After(timeout):
			t.Fatalf("receive from channel timed out")
		}
		mustBeClosedTimeoutESC(out, timeout, t)
	}
	var runs int
	for !(outWasOpen && outWasClosed) {
		// 2 or 3 runs should be enough. The probability of needing more
		// than 16 runs to observe both outWasOpen and outWasClosed is
		// (1/2)^16 ≈ 0.001
		if runs > 16 {
			t.Fatalf("failed to observe both conditions: outWasOpen=%v, outWasClosed=%v", outWasOpen, outWasClosed)
		}
		test()
		runs++
	}
	t.Logf("#runs: %d", runs)
}

func TestWatchNewFile(t *testing.T) {
	path, err := ioutil.TempDir("", "redgreen")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(path)

	done := make(chan struct{})
	defer close(done)
	out, err := redgreen.Watch(done, path)
	if err != nil {
		t.Fatalf("Watch(done, %q) = %v, want nil", path, err)
	}

	// Creating a file should trigger a watch event.
	_, err = os.Create(filepath.Join(path, "foo"))
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	select {
	case <-out:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for watch event")
	}
}

func TestWatchRemoveFile(t *testing.T) {
	path, err := ioutil.TempDir("", "redgreen")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(path)
	_, err = os.Create(filepath.Join(path, "foo"))
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	done := make(chan struct{})
	defer close(done)
	out, err := redgreen.Watch(done, path)
	if err != nil {
		t.Fatalf("Watch(done, %q) = %v, want nil", path, err)
	}

	// Removing a file should trigger a watch event.
	err = os.Remove(filepath.Join(path, "foo"))
	if err != nil {
		t.Fatalf("remove temp file: %v", err)
	}
	select {
	case <-out:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for watch event")
	}
}

func TestWatchModifyFile(t *testing.T) {
	path, err := ioutil.TempDir("", "redgreen")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(path)
	f, err := os.Create(filepath.Join(path, "foo"))
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	done := make(chan struct{})
	defer close(done)
	out, err := redgreen.Watch(done, path)
	if err != nil {
		t.Fatalf("Watch(done, %q) = %v, want nil", path, err)
	}

	// Modifying a file should trigger a watch event.
	f.WriteString("test")
	err = f.Sync()
	if err != nil {
		t.Fatalf("sync temp file: %v", err)
	}
	select {
	case <-out:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for watch event")
	}
}

// mustBeClosedTimeoutESC is like mustBeClosedTimeout but takes a channel of
// empty structs.
func mustBeClosedTimeoutESC(ch <-chan struct{}, timeout time.Duration, t *testing.T) {
	select {
	case _, isOpen := <-ch:
		if isOpen {
			t.Fatalf("channel is open, want closed")
		}
	case <-time.After(timeout):
		t.Fatalf("receive from channel timed out")
	}
}

func TestStateColor(t *testing.T) {
	var s redgreen.State
	checkColor(s, redgreen.ColorYellow, t)
	s.Results = append(s.Results, redgreen.RunResult{})
	checkColor(s, redgreen.ColorGreen, t)
	s.Results = append(s.Results, redgreen.RunResult{Error: errors.New("test")})
	checkColor(s, redgreen.ColorRed, t)
}

func checkColor(s redgreen.State, want redgreen.Color, t *testing.T) {
	if got := s.Color(); got != want {
		t.Errorf("s.Color() = %v, want %v", got, want)
	}
}

func TestRenderDone(t *testing.T) {
	done := make(chan struct{})
	in := make(chan redgreen.State)
	ok := make(chan struct{}, 1)
	go func() {
		redgreen.Render(done, in)
		ok <- struct{}{}
	}()
	in <- redgreen.State{}
	close(done)
	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting goroutine to return")
	}
}

func TestRenderClosedInput(t *testing.T) {
	done := make(chan struct{})
	in := make(chan redgreen.State)
	ok := make(chan struct{}, 1)
	go func() {
		redgreen.Render(done, in)
		ok <- struct{}{}
	}()
	in <- redgreen.State{}
	close(in)
	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting goroutine to return")
	}
}
