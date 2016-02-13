package main

import (
	"flag"
	"log"
	"os/exec"
	"time"

	"github.com/go-fsnotify/fsnotify"
	"github.com/nsf/termbox-go"
)

var (
	debug   = flag.Bool("debug", false, "enable debug mode, disable termbox")
	timeout = flag.Duration("timeout", 5*time.Second, "maximum time to wait for command to finish")
)

var command []string

var updateScreen = func(err error) {
	log.Printf("command returned: %v", err)
}

func run() error {
	cmd := exec.Command(command[0], command[1:]...)
	if *debug {
		out, err := cmd.CombinedOutput()
		log.Printf("command: %#v", cmd.Args)
		log.Printf("output: %s", out)
		return err
	}
	time.AfterFunc(*timeout, func() {
		cmd.Process.Kill()
	})
	return cmd.Run()
}

func main() {
	flag.Parse()
	if flag.NArg() > 0 {
		command = flag.Args()
	} else {
		command = []string{"go", "test"}
	}

	if !*debug {
		if err := termbox.Init(); err != nil {
			log.Fatal(err)
		}
		defer termbox.Close()
		termbox.HideCursor()
		termbox.SetOutputMode(termbox.Output256)

		updateScreen = termboxUpdateScreen
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}

	updateScreen(run())
	go func() {
		for {
			event := <-watcher.Events
			if event.Op&fsnotify.Write == fsnotify.Write {
				updateScreen(run())
			}
		}
	}()

	for {
		e := termbox.PollEvent()
		if e.Type == termbox.EventKey && e.Key == termbox.KeyEsc {
			return
		}
	}
}

func termboxUpdateScreen(err error) {
	color := termbox.ColorGreen
	if err != nil {
		color = termbox.ColorRed
	}
	termboxSetBg(color)
}

func termboxSetBg(color termbox.Attribute) {
	buf := termbox.CellBuffer()
	for i := range buf {
		buf[i].Bg = color
	}
	termbox.Flush()
}
