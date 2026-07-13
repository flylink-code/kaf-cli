//go:build windows && wailsgui

package main

import "syscall"

func hideWindowAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
