package teststdio

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// mu serializes os.Stdout/os.Stderr swaps across test packages that run in parallel.
var mu sync.Mutex

// Capture runs fn while capturing process stdout and stderr.
func Capture(fn func()) (stdout, stderr string) {
	mu.Lock()
	defer mu.Unlock()

	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	os.Stdout = outW
	os.Stderr = errW

	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(&outBuf, outR)
	}()
	go func() {
		defer wg.Done()
		io.Copy(&errBuf, errR)
	}()

	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

	fn()
	outW.Close()
	errW.Close()
	wg.Wait()
	outR.Close()
	errR.Close()
	return outBuf.String(), errBuf.String()
}

// CaptureStdout runs fn while capturing process stdout.
func CaptureStdout(fn func()) string {
	stdout, _ := Capture(fn)
	return stdout
}
