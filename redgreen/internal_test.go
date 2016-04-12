package redgreen

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func Test_run(t *testing.T) {
	tests := []struct {
		command []string
		timeout time.Duration
		check   checkFunc
	}{
		{
			command: []string{},
			check: func(err error) error {
				got := err.Error()
				want := "command must not be empty"
				if got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			command: []string{"invalid command"},
			check:   isExecError,
		},
		{
			command: []string{"true"},
			check:   isNil,
		},
		{
			command: []string{"false"},
			check:   isSignal(-1),
		},
		{
			command: []string{"sleep", "2"},
			timeout: 1 * time.Nanosecond,
			check:   isSignal(syscall.SIGKILL),
		},
	}
	for _, tt := range tests {
		err := run(tt.command, tt.timeout, false)
		if checkErr := tt.check(err); checkErr != nil {
			t.Errorf("run(%v, %v): %v", tt.command, tt.timeout, checkErr)
		}
	}
}

// checkFunc takes an error and returns another error if the given error does
// not satisfy a certain condition.
type checkFunc func(error) error

func isNil(err error) error {
	if err != nil {
		return fmt.Errorf("got %T (%[1]v), want nil", err)
	}
	return nil
}
func isExecError(err error) error {
	if _, ok := err.(*exec.Error); !ok {
		return fmt.Errorf("got %T (%[1]v), want *exec.Error", err)
	}
	return nil
}
func isSignal(want syscall.Signal) checkFunc {
	return func(err error) error {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return fmt.Errorf("got %T (%[1]v), want *exec.ExitError", err)
		}
		status, ok := exitErr.Sys().(syscall.WaitStatus)
		if !ok {
			return errors.New("cannot check proccess status")
		}
		got := status.Signal()
		if got != want {
			return fmt.Errorf("got %v, want %v", got, want)
		}
		return nil
	}
}
