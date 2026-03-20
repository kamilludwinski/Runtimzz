package spinner

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var frames = []string{"|", "/", "-", "\\"}

// interactive returns false in CI or when the terminal is dumb (e.g. no TTY).
// When false, we skip the spinner to avoid writing to stderr, which can break
// scripts (e.g. PowerShell treats native stderr as errors).
func interactive() bool {
	if os.Getenv("CI") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return true
}

func Run(msg string, fn func() error) error {
	if !interactive() {
		return fn()
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", len(frames[0])+1+len(msg))+"\r")
				return
			case <-ticker.C:
				frame := frames[i%len(frames)]
				fmt.Fprint(os.Stderr, "\r"+frame+" "+msg)
				i++
			}
		}
	}()

	err := fn()
	close(done)
	wg.Wait()

	return err
}
