// +build !windows

package main

import "fmt"

// EnableAutoStart is not supported on non-Windows platforms yet
func (a *App) EnableAutoStart() error {
	return fmt.Errorf("auto-start is only supported on Windows")
}

// DisableAutoStart is not supported on non-Windows platforms yet
func (a *App) DisableAutoStart() error {
	return fmt.Errorf("auto-start is only supported on Windows")
}

// IsAutoStartEnabled always returns false on non-Windows platforms
func (a *App) IsAutoStartEnabled() bool {
	return false
}
