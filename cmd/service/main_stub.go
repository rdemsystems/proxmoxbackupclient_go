//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Nimbus Backup Service is only supported on Windows")
	os.Exit(1)
}
