//go:build !windows

package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	fmt.Fprintf(os.Stderr, "krayn-uninstall %s (%s) is only supported on Windows\n", version, commit)
	os.Exit(1)
}
