// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"

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

// Windows: Use Task Scheduler with highest privileges for admin rights
func enableAutoStartWindows(exePath string) error {
	// Use schtasks to create a task that runs at logon with highest privileges
	// This ensures the app starts with admin rights for VSS to work
	taskName := "NimbusBackup"

	// Delete existing task if any
	disableAutoStartWindows()

	// Also clean up old registry entry if exists (migration from old version)
	cleanupLegacyRegistryEntry()

	// Create task with highest privileges
	// /SC ONLOGON = trigger at logon
	// /RL HIGHEST = run with highest privileges (admin)
	// /F = force create (overwrite if exists)
	cmd := fmt.Sprintf(`schtasks /Create /TN "%s" /TR "\"%s\" --minimized" /SC ONLOGON /RL HIGHEST /F`,
		taskName, exePath)

	writeDebugLog(fmt.Sprintf("Creating scheduled task: %s", cmd))

	// Execute schtasks command
	if err := executeCommand("cmd", "/C", cmd); err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	writeDebugLog("Auto-start enabled via Task Scheduler with admin privileges")
	return nil
}

func disableAutoStartWindows() error {
	taskName := "NimbusBackup"

	// Delete scheduled task (ignore error if doesn't exist)
	cmd := fmt.Sprintf(`schtasks /Delete /TN "%s" /F`, taskName)
	executeCommand("cmd", "/C", cmd)

	writeDebugLog("Auto-start disabled (task scheduler)")
	return nil
}

func isAutoStartEnabledWindows() bool {
	taskName := "NimbusBackup"

	// Query if task exists
	cmd := fmt.Sprintf(`schtasks /Query /TN "%s"`, taskName)
	err := executeCommand("cmd", "/C", cmd)
	return err == nil
}

// executeCommand executes a Windows command
func executeCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Command failed: %s - %s", err, string(output)))
		return err
	}
	writeDebugLog(fmt.Sprintf("Command success: %s", string(output)))
	return nil
}

// cleanupLegacyRegistryEntry removes old registry-based auto-start
func cleanupLegacyRegistryEntry() {
	// Import here to avoid issues on non-Windows
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	key.DeleteValue("NimbusBackup")
	writeDebugLog("Cleaned up legacy registry entry")
}

