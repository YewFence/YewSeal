//go:build !windows && !plan9

package main

import (
	"os/signal"
	"syscall"
)

// Keep broken diagnostic pipes from terminating active file operations.
func handleBrokenPipes() { signal.Ignore(syscall.SIGPIPE) }
