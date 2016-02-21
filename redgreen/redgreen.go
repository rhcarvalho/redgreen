package redgreen

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/go-fsnotify/fsnotify"
)

// RunSpec holds the specification of a command to be run.
type RunSpec struct {
	Command []string
	Timeout time.Duration
}

// RunResult holds information about a command execution.
type RunResult struct {
	Error error
	// CombinedOutput []byte
}

// Run runs commands coming from the in channel in a new goroutine and returns a
// channel of results of each execution. Input and output is synchronized, a new
// command will be executed only after the error returned by the previous
// execution is consumed downstream. Closing either done or in signals that no
// more commands are to be run, and, consequently, the output channel will be
// closed.
func Run(done <-chan struct{}, in <-chan RunSpec) <-chan RunResult {
	out := make(chan RunResult)
	go func() {
		defer close(out)
		for {
			select {
			case spec, ok := <-in:
				if !ok {
					return
				}
				r := RunResult{run(spec.Command, spec.Timeout)}
				select {
				case out <- r:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()
	return out
}

// run runs command and waits for it to terminate for at most timeout. Zero or
// negative timeout means no timeout.
func run(command []string, timeout time.Duration) error {
	if len(command) == 0 {
		return errors.New("command must not be empty")
	}
	cmd := exec.Command(command[0], command[1:]...)
	if err := cmd.Start(); err != nil {
		return err
	}
	if timeout > 0 {
		defer time.AfterFunc(timeout, func() { cmd.Process.Kill() }).Stop()
	}
	return cmd.Wait()
}

// Watch returns a channel that will be sent to once for every file system event
// in path (non-recursively). Closing done interrupts the file system watcher
// and closes the output channel, freeing all allocated resources.
func Watch(done <-chan struct{}, path string) (<-chan struct{}, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create file system watcher: %v", err)
	}
	err = watcher.Add(path)
	if err != nil {
		return nil, fmt.Errorf("add path %q to file system watcher: %v", path, err)
	}
	out := make(chan struct{})
	go func() {
		defer close(out)
		defer watcher.Close()
		for {
			select {
			case <-watcher.Events:
				select {
				case out <- struct{}{}:
				case <-done:
					return
				}
			case err := <-watcher.Errors:
				log.Println("ERROR:", err)
			case <-done:
				return
			}
		}
	}()
	return out, nil
}
