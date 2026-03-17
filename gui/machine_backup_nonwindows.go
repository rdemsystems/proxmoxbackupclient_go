//go:build !windows
// +build !windows

package main

import "fmt"

// RunMachineBackup is not supported on non-Windows platforms
func RunMachineBackup(opts BackupOptions) error {
	return fmt.Errorf("Machine backup is only supported on Windows")
}
