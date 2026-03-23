//go:build !windows
// +build !windows

package main

// cleanupVSS is a no-op on non-Windows platforms
func cleanupVSS() error {
	// VSS is Windows-only, nothing to clean up on other platforms
	return nil
}
