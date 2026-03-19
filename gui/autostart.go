// +build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// EnableAutoStart enables the application to start at system boot (Windows)
func (a *App) EnableAutoStart() error {
	writeDebugLog("EnableAutoStart called")

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	return enableAutoStartWindows(exePath)
}

// DisableAutoStart disables the application from starting at system boot (Windows)
func (a *App) DisableAutoStart() error {
	writeDebugLog("DisableAutoStart called")
	return disableAutoStartWindows()
}

// IsAutoStartEnabled checks if auto-start is currently enabled (Windows)
func (a *App) IsAutoStartEnabled() bool {
	return isAutoStartEnabledWindows()
}

// Windows: Use Registry (HKCU\Software\Microsoft\Windows\CurrentVersion\Run)
func enableAutoStartWindows(exePath string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// Set registry value to start minimized to tray
	err = key.SetStringValue("NimbusBackup", fmt.Sprintf(`"%s" --minimized`, exePath))
	if err != nil {
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	writeDebugLog("Auto-start enabled in Windows registry")
	return nil
}

func disableAutoStartWindows() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	err = key.DeleteValue("NimbusBackup")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to delete registry value: %w", err)
	}

	writeDebugLog("Auto-start disabled in Windows registry")
	return nil
}

func isAutoStartEnabledWindows() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue("NimbusBackup")
	return err == nil
}

