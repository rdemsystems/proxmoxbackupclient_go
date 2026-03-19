package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sys/windows/registry"
)

// EnableAutoStart enables the application to start at system boot
func (a *App) EnableAutoStart() error {
	writeDebugLog("EnableAutoStart called")

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if runtime.GOOS == "windows" {
		return enableAutoStartWindows(exePath)
	}
	return enableAutoStartLinux(exePath)
}

// DisableAutoStart disables the application from starting at system boot
func (a *App) DisableAutoStart() error {
	writeDebugLog("DisableAutoStart called")

	if runtime.GOOS == "windows" {
		return disableAutoStartWindows()
	}
	return disableAutoStartLinux()
}

// IsAutoStartEnabled checks if auto-start is currently enabled
func (a *App) IsAutoStartEnabled() bool {
	if runtime.GOOS == "windows" {
		return isAutoStartEnabledWindows()
	}
	return isAutoStartEnabledLinux()
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

// Linux: Use XDG autostart (.desktop file in ~/.config/autostart/)
func enableAutoStartLinux(exePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	autostartDir := filepath.Join(homeDir, ".config", "autostart")
	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return fmt.Errorf("failed to create autostart directory: %w", err)
	}

	desktopFilePath := filepath.Join(autostartDir, "nimbus-backup.desktop")

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Nimbus Backup
Exec=%s --minimized
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Comment=Nimbus Backup - Scheduled backups to Proxmox Backup Server
`, exePath)

	if err := os.WriteFile(desktopFilePath, []byte(desktopContent), 0644); err != nil {
		return fmt.Errorf("failed to write desktop file: %w", err)
	}

	// Make executable
	if err := os.Chmod(desktopFilePath, 0755); err != nil {
		return fmt.Errorf("failed to make desktop file executable: %w", err)
	}

	writeDebugLog("Auto-start enabled via .desktop file")
	return nil
}

func disableAutoStartLinux() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	desktopFilePath := filepath.Join(homeDir, ".config", "autostart", "nimbus-backup.desktop")

	if err := os.Remove(desktopFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove desktop file: %w", err)
	}

	writeDebugLog("Auto-start disabled (desktop file removed)")
	return nil
}

func isAutoStartEnabledLinux() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	desktopFilePath := filepath.Join(homeDir, ".config", "autostart", "nimbus-backup.desktop")
	_, err = os.Stat(desktopFilePath)
	return err == nil
}

// HandleStartupRun executes scheduled jobs that have runAtStartup enabled
func (a *App) HandleStartupRun() {
	writeDebugLog("HandleStartupRun called - checking for startup jobs")

	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading scheduled jobs: %v", err))
		return
	}

	for _, job := range jobs {
		if !job.Enabled || !job.RunAtStartup {
			continue
		}

		writeDebugLog(fmt.Sprintf("Executing startup job: %s", job.Name))
		go a.executeScheduledJob(job)
	}
}
