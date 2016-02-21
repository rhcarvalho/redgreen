package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
	"github.com/rhcarvalho/redgreen/redgreen"
)

// Command-line flags and arguments.
var (
	testCommand = []string{"go", "test"}
	timeout     time.Duration
	debug       bool
)

func init() {
	flag.BoolVar(&debug, "debug", false, "enable debug mode, disable termbox")
	flag.DurationVar(&timeout, "timeout", 5*time.Second, "maximum time to wait for command to finish")
}

func main() {
	flag.Parse()

	// Customize testCommand if passed as arguments.
	args := flag.Args()
	if len(args) > 0 {
		testCommand = args
	}

	if err := do(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func do() error {
	done := make(chan struct{})
	defer close(done)

	// Initialize and defer termination of termbox.
	if !debug {
		if err := termbox.Init(); err != nil {
			return err
		}
		defer termbox.Close()
		termbox.HideCursor()
		termbox.SetOutputMode(termbox.Output256)
	}

	w, err := redgreen.Watch(done, ".")
	if err != nil {
		return err
	}

	runSpec := redgreen.RunSpec{Command: testCommand, Timeout: timeout}

	run := make(chan redgreen.RunSpec, 1)
	// Trigger an initial run of the test command.
	run <- runSpec
	// Run tests every time a file is created/removed/modified.
	go func() {
		for range w {
			run <- runSpec
		}
	}()

	res := redgreen.Run(done, run)

	state := make(chan redgreen.State)
	go redgreen.Render(done, state)

	s := redgreen.State{Debug: debug}
	var mu sync.RWMutex // synchronizes access to s.

	// Render initial state.
	state <- s
	// Render after every test command result.
	go func() {
		for r := range res {
			mu.Lock()
			s.Results = append(s.Results, r)
			mu.Unlock()
			mu.RLock()
			state <- s
			mu.RUnlock()
		}
	}()

	if debug {
		// Wait for Ctrl-C.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
	} else {
		// Block until Esc is pressed.
		for {
			e := termbox.PollEvent()
			if e.Type == termbox.EventKey && e.Key == termbox.KeyEsc {
				break
			}
			if e.Type == termbox.EventResize {
				mu.RLock()
				state <- s
				mu.RUnlock()
			}
		}
	}
	return nil
}
