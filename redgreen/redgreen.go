package redgreen

import (
	"errors"
	"os/exec"
	"time"
)

// RunSpec holds the specification of a command to be run.
type RunSpec struct {
	Command []string
	Timeout time.Duration
}

// Run runs commands coming from the in channel in a new goroutine and returns a
// channel of errors of each execution. Input and output is synchronized, a new
// command will be executed only after the error returned by the previous
// execution is consumed downstream. Closing either done or in signals that no
// more commands are to be run, and, consequently, the output channel will be
// closed.
func Run(done <-chan struct{}, in <-chan RunSpec) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)
		for {
			select {
			case spec, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- run(spec.Command, spec.Timeout):
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
