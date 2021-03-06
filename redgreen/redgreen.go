package redgreen

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nsf/termbox-go"
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
				// FIXME: expose the debug flag properly.
				debugFlag := flag.Lookup("debug")
				debug := debugFlag != nil && debugFlag.Value.String() == "true"
				r := RunResult{run(spec.Command, spec.Timeout, debug)}
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
func run(command []string, timeout time.Duration, debug bool) (err error) {
	if len(command) == 0 {
		return errors.New("command must not be empty")
	}
	cmd := exec.Command(command[0], command[1:]...)
	if debug {
		log.Printf("running: %s", strings.Join(cmd.Args, " "))
		var b bytes.Buffer
		cmd.Stdout = &b
		cmd.Stderr = &b
		defer func() {
			log.Printf("output:\n%s", b.String())
			if err != nil {
				log.Println("error:", err)
			}
		}()
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if timeout > 0 {
		defer time.AfterFunc(timeout, func() { cmd.Process.Kill() }).Stop()
	}
	return cmd.Wait()
}

// Watch returns a channel that will be sent to after file system events in path
// (non-recursively). Send operations happen after a certain delay, debouncing
// multiple events within the delay. This is useful to group together multiple
// related events, such as what happens when a file is saved and immediately
// automatically gofmt'ed. Closing done interrupts the file system watcher and
// closes the output channel, freeing all allocated resources.
func Watch(done <-chan struct{}, path string, delay time.Duration) (<-chan struct{}, error) {
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
		// timer holds a scheduled send to out.
		var timer *time.Timer
		// stopTimer stops the timer if non-nil.
		stopTimer := func() {
			if timer != nil {
				timer.Stop()
			}
		}
		// Abort any previously scheduled send before returning.
		defer stopTimer()
		for {
			select {
			case <-watcher.Events:
				// Abort any previously scheduled send.
				stopTimer()
				// Schedule send to out.
				timer = time.AfterFunc(delay, func() {
					select {
					case out <- struct{}{}:
					case <-done:
						return
					}
				})
			case err := <-watcher.Errors:
				log.Println("ERROR:", err)
			case <-done:
				return
			}
		}
	}()
	return out, nil
}

// State represents the program state that can be rendered to the screen. If
// Debug is false, termbox must have been initialized. If Debug is true, termbox
// is not used.
type State struct {
	Results []RunResult
	Debug   bool
}

// Color returns the color that represents the state. There are three possible
// colors: ColorRed means the last test command failed, ColorGreen means the
// last test command succeeded, and ColorYellow means the state is unknown.
func (s State) Color() Color {
	var color Color
	if len(s.Results) == 0 {
		color = ColorYellow
	} else {
		if s.Results[len(s.Results)-1].Error == nil {
			color = ColorGreen
		} else {
			color = ColorRed
		}
	}
	return color
}

// A Color represents the state of the program.
type Color termbox.Attribute

// All possible colors.
const (
	ColorRed    = Color(termbox.ColorRed)
	ColorGreen  = Color(termbox.ColorGreen)
	ColorYellow = Color(termbox.ColorYellow)
)

func (c Color) String() string {
	switch c {
	case ColorRed:
		return "red"
	case ColorGreen:
		return "green"
	case ColorYellow:
		return "yellow"
	default:
		return "unknown"
	}
}

// Render receives updates to the program state from in, and updates the screen
// accordingly. Render blocks until either done or in is closed.
func Render(done <-chan struct{}, in <-chan State) {
	for {
		select {
		case s, ok := <-in:
			if !ok {
				return
			}
			render(s)
		case <-done:
			return
		}
	}
}

// render updates the screen according to s.
func render(s State) {
	color := s.Color()
	if s.Debug {
		log.Printf("render: %v", color)
	} else {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		buf := termbox.CellBuffer()
		w, _ := termbox.Size()
		for i := range buf[:w] {
			k := len(s.Results) - i - 1
			if k < 0 {
				break
			}
			if err := s.Results[k].Error; err == nil {
				buf[i].Fg = termbox.ColorGreen
				buf[i].Ch = '✔'
			} else {
				buf[i].Fg = termbox.ColorRed
				buf[i].Ch = '✘'
			}
		}
		for i := range buf[w:] {
			buf[w+i].Bg = termbox.Attribute(color)
		}
		termbox.Flush()
	}
}
