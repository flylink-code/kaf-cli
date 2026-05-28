//go:build windows && wailsgui

package main

import (
	"fmt"
	"io"
	"os"
	"sync"

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
)

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
	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	wg.Wait()

	if runErr != nil {
		return fmt.Errorf("转换失败: %w", runErr)
	}
	return nil
}
