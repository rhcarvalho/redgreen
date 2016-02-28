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
	if flag.NArg() > 0 {
		testCommand = flag.Args()
	}

	if err := do(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func do() error {
	// Initialize and defer termination of termbox.
	if !debug {
		if err := termbox.Init(); err != nil {
			return err
		}
		defer termbox.Close()
		termbox.HideCursor()
		termbox.SetOutputMode(termbox.Output256)
	}

	// wg waits for all goroutines started by this function to return.
	var wg sync.WaitGroup
	defer wg.Wait()

	// Closing done signals all goroutines to terminate.
	done := make(chan struct{})
	defer close(done)

	w, err := redgreen.Watch(done, ".", 200*time.Millisecond)
	if err != nil {
		return err
	}

	runSpec := redgreen.RunSpec{Command: testCommand, Timeout: timeout}
	run := make(chan redgreen.RunSpec, 1)
	res := redgreen.Run(done, run)

	// Trigger an initial run of the test command.
	run <- runSpec
	// Run tests every time a file is created/removed/modified.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range w {
			run <- runSpec
		}
	}()

	state := make(chan redgreen.State)
	wg.Add(1)
	go func() {
		defer wg.Done()
		redgreen.Render(done, state)
	}()

	s := redgreen.State{Debug: debug}
	var mu sync.RWMutex // synchronizes access to s.

	// Render initial state.
	state <- s
	// Render after every test command result.
	wg.Add(1)
	go func() {
		defer wg.Done()
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
