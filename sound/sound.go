// Package sound provides functions to play sounds and some predefined sounds.
// This package is mostly a port of the original Python code in
// https://github.com/rhcarvalho/sound_alarm.
package sound

import (
	"bytes"
	"math"
	"os/exec"
	"time"
)

// Play plays the sound stored in the buffer buf.
func Play(buf *bytes.Buffer) {
	cmd := exec.Command("aplay")
	cmd.Stdin = buf
	cmd.Run()
}

// Say speaks aloud the string s using text-to-speech.
func Say(s string) {
	cmd := exec.Command("espeak")
	cmd.Stdin = bytes.NewBufferString(string(s))
	cmd.Run()
}

// makeBeep is a magical function to produce a mono sound signal of the given
// frequency and duration sampled at 8 kHz.
func makeBeep(frequency float64, duration time.Duration) []byte {
	const (
		sample    = 8000
		amplitude = 100
	)
	halfPeriod := int(sample / frequency / 2)
	return bytes.Repeat(
		append(bytes.Repeat([]byte{amplitude}, halfPeriod),
			bytes.Repeat([]byte{0}, halfPeriod)...),
		int(float64(duration)*frequency/float64(time.Second)))
}

// Some predefined sounds.
var (
	ExplosiveCounter bytes.Buffer
	SuperNintendo    bytes.Buffer
	PhoneRing        bytes.Buffer
	Fluid            bytes.Buffer
)

func init() {
	for i := 6; i < 85; i++ {
		ExplosiveCounter.Write(makeBeep(37*float64(i), 50*time.Millisecond))
		ExplosiveCounter.Write(makeBeep(1, 70*time.Nanosecond))
	}
	ExplosiveCounter.Write(makeBeep(37, 2000))

	for i := 1; i < 5; i++ {
		for j := 1; j < 5; j++ {
			SuperNintendo.Write(makeBeep(100*float64(i*j), 50*time.Millisecond))
			SuperNintendo.Write(makeBeep(1, 20*time.Nanosecond))
		}
	}

	for i := 1; i < 5; i++ {
		for j := 1; j < 5; j++ {
			PhoneRing.Write(makeBeep(700+40*float64(i), 200*time.Millisecond))
			PhoneRing.Write(makeBeep(1, 1*time.Second))
		}
	}

	for i := float64(0); i < math.Pi/2; i += 0.1 {
		Fluid.Write(makeBeep(200+100*math.Sin(i), 200*time.Millisecond))
	}
}
