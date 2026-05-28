//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
)

const logMaxBytes = 512 << 10 // 512KB

type logBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (l *logBuffer) append(s string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf.WriteString(s)
	out := l.buf.String()
	if len(out) > logMaxBytes {
		out = out[len(out)-logMaxBytes:]
		l.buf.Reset()
		l.buf.WriteString(out)
	}
	return out
}

func (l *logBuffer) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf.Reset()
}

func runConvert(opts kafcli.GUIOptions, onLog func(string)) error {
	book := kafcli.NewBookGUI(opts)

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return err
	}

	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	pump := func(r io.Reader) {
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				onLog(string(buf[:n]))
			}
			if readErr != nil {
				if readErr != io.EOF {
					onLog(readErr.Error())
				}
				return
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pump(stdoutR)
	}()
	go func() {
		defer wg.Done()
		pump(stderrR)
	}()

	runErr := kafcli.Run(book, version, secret, measurement)
	stdoutW.Close()
	stderrW.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	wg.Wait()

	if runErr != nil {
		return fmt.Errorf("转换失败: %w", runErr)
	}
	return nil
}
