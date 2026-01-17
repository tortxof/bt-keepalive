package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

const (
	sampleRate   = 44100
	channelCount = 2
	bitDepth     = 2 // 16-bit = 2 bytes
)

type SineWaveReader struct {
	freq   float64
	volume float64
	pos    int64
}

func (s *SineWaveReader) Read(buf []byte) (int, error) {
	for i := 0; i < len(buf)/4; i++ {
		sample := s.volume * math.Sin(2*math.Pi*s.freq*float64(s.pos)/sampleRate)
		s.pos++

		// Convert to 16-bit signed integer
		val := int16(sample * math.MaxInt16)

		// Write left channel
		buf[i*4] = byte(val)
		buf[i*4+1] = byte(val >> 8)
		// Write right channel
		buf[i*4+2] = byte(val)
		buf[i*4+3] = byte(val >> 8)
	}
	return len(buf), nil
}

func main() {
	var duration time.Duration
	var freq float64
	var volume float64

	flag.DurationVar(&duration, "d", 0, "Duration to run (e.g., 1h, 30m). 0 = indefinite")
	flag.DurationVar(&duration, "duration", 0, "Duration to run (e.g., 1h, 30m). 0 = indefinite")
	flag.Float64Var(&freq, "freq", 20, "Sine wave frequency in Hz")
	flag.Float64Var(&volume, "volume", 0.001, "Volume level (0.0-1.0)")
	flag.Parse()

	if volume < 0 || volume > 1 {
		fmt.Fprintln(os.Stderr, "Error: volume must be between 0.0 and 1.0")
		os.Exit(1)
	}

	if freq < 1 || freq > 20000 {
		fmt.Fprintln(os.Stderr, "Error: frequency must be between 1 and 20000 Hz")
		os.Exit(1)
	}

	ctx, ready, err := oto.NewContext(sampleRate, channelCount, bitDepth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing audio: %v\n", err)
		os.Exit(1)
	}
	<-ready

	sineWave := &SineWaveReader{freq: freq, volume: volume}
	player := ctx.NewPlayer(sineWave)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up timeout if duration specified
	var timer *time.Timer
	if duration > 0 {
		timer = time.NewTimer(duration)
	}

	fmt.Printf("Playing %.0f Hz sine wave at %.1f%% volume\n", freq, volume*100)
	if duration > 0 {
		fmt.Printf("Will stop after %v\n", duration)
	} else {
		fmt.Println("Press Ctrl+C to stop")
	}

	player.Play()

	// Wait for signal or timeout
	select {
	case <-sigChan:
		fmt.Println("\nShutting down...")
	case <-func() <-chan time.Time {
		if timer != nil {
			return timer.C
		}
		return make(chan time.Time)
	}():
		fmt.Println("\nDuration reached, shutting down...")
	}

	player.Close()
}
