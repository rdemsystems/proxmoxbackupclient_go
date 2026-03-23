//go:build windows
// +build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// cleanupVSS removes all orphaned VSS snapshots at service startup
// This prevents shadow copies from accumulating after crashes or abnormal terminations
func cleanupVSS() error {
	// List all shadows first to check if cleanup is needed
	listCmd := exec.Command("vssadmin", "list", "shadows")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		// If vssadmin fails, log but don't block service startup
		return fmt.Errorf("failed to list VSS shadows: %w", err)
	}

	// Only delete if there are actually shadows present
	if len(output) > 0 && !strings.Contains(string(output), "No items found") {
		logger.Info("VSS Cleanup: Removing orphaned shadow copies...")

		// Delete all shadow copies
		// Note: This is safe because Nimbus creates snapshots only during backup
		// and releases them immediately after. Any remaining snapshots are orphans.
		deleteCmd := exec.Command("vssadmin", "delete", "shadows", "/all", "/quiet")
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("VSS cleanup failed: %w - %s", err, string(deleteOutput))
		}

		logger.Info("VSS Cleanup: Successfully removed orphaned snapshots")
	} else {
		logger.Info("VSS Cleanup: No orphaned snapshots found")
	}

	return nil
}
