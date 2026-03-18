package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"security"
)

type Config struct {
	// PBS Connection
	BaseURL         string `json:"baseurl"`
	CertFingerprint string `json:"certfingerprint"`
	AuthID          string `json:"authid"`
	Secret          string `json:"secret"`
	Datastore       string `json:"datastore"`
	Namespace       string `json:"namespace"`

	// Backup Settings
	BackupDir      string   `json:"backupdir"`
	BackupID       string   `json:"backup-id"`
	UseVSS         bool     `json:"usevss"`
	LastBackupDirs []string `json:"last_backup_dirs,omitempty"` // Remember last used directories

	// Email Notifications (optional)
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     string `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	EmailFrom    string `json:"email_from,omitempty"`
	EmailTo      string `json:"email_to,omitempty"`
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".proxmox-backup-guardian")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func LoadConfig() *Config {
	config := &Config{
		UseVSS: true, // Default to VSS enabled on Windows
	}

	configPath, err := getConfigPath()
	if err != nil {
		return config
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist yet, return empty config
		return config
	}

	if err := json.Unmarshal(data, config); err != nil {
		return config
	}

	return config
}

func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func (c *Config) Validate() error {
	// Validate BaseURL
	if c.BaseURL == "" {
		return fmt.Errorf("URL du serveur PBS requis")
	}
	if err := security.ValidateURL(c.BaseURL); err != nil {
		return fmt.Errorf("URL invalide: %w", err)
	}

	// Validate AuthID
	if c.AuthID == "" {
		return fmt.Errorf("Authentication ID requis")
	}
	if err := security.ValidateAuthID(c.AuthID); err != nil {
		return fmt.Errorf("Authentication ID invalide: %w", err)
	}

	// Validate Secret (non-empty check)
	if c.Secret == "" {
		return fmt.Errorf("Secret requis")
	}

	// Validate Datastore
	if c.Datastore == "" {
		return fmt.Errorf("Datastore requis")
	}
	if err := security.ValidateDatastore(c.Datastore); err != nil {
		return fmt.Errorf("Datastore invalide: %w", err)
	}

	// Validate BackupID if present
	if c.BackupID != "" {
		if err := security.ValidateBackupID(c.BackupID); err != nil {
			return fmt.Errorf("Backup ID invalide: %w", err)
		}
	}

	// Validate Certificate Fingerprint if present
	if c.CertFingerprint != "" {
		if err := security.ValidateFingerprint(c.CertFingerprint); err != nil {
			return fmt.Errorf("Empreinte certificat invalide: %w", err)
		}
	}

	// Validate BackupDir if present
	if c.BackupDir != "" {
		if err := security.ValidatePath(c.BackupDir); err != nil {
			return fmt.Errorf("Chemin de backup invalide: %w", err)
		}
	}

	return nil
}
