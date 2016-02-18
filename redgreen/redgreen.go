package redgreen

import (
	"errors"
	"os/exec"
	"time"
)

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
