package redgreen

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// helperCommand creates a simulated external command for tests.
func helperCommand(name string, arg ...string) *exec.Cmd {
	arg = append([]string{"-test.run=TestHelperProcess", "--", name}, arg...)
	cmd := exec.Command(os.Args[0], arg...)
	cmd.Env = []string{"WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess isn't a real test. It's used by other tests to simulate
// running external commands.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "sleep":
		d, _ := time.ParseDuration(args[0])
		time.Sleep(d)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func TestRunWithTimeout(t *testing.T) {
	tests := []struct {
		sleep   time.Duration
		timeout time.Duration
		want    error
	}{
		{time.Second, time.Millisecond, TimeoutError(time.Millisecond)},
		{time.Millisecond, 20 * time.Millisecond, nil},
	}
	for _, tt := range tests {
		// Slow down when running with the race detector enabled.
		if RaceEnabled {
			const factor = 100
			tt.sleep *= factor
			tt.timeout *= factor
			if _, ok := tt.want.(TimeoutError); ok {
				tt.want = TimeoutError(tt.timeout)
			}
		}
		cmd := helperCommand("sleep", tt.sleep.String())
		if err := RunWithTimeout(cmd, tt.timeout); err != tt.want {
			t.Errorf("RunWithTimeout(\"sleep %v\", %v) = %v, want %v", tt.sleep, tt.timeout, err, tt.want)
		}
	}
}
