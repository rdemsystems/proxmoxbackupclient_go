package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	BackupDir string `json:"backupdir"`
	BackupID  string `json:"backup-id"`
	UseVSS    bool   `json:"usevss"`

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
	if c.BaseURL == "" {
		return fmt.Errorf("URL du serveur PBS requis")
	}
	if c.AuthID == "" {
		return fmt.Errorf("Authentication ID requis")
	}
	if c.Secret == "" {
		return fmt.Errorf("Secret requis")
	}
	if c.Datastore == "" {
		return fmt.Errorf("Datastore requis")
	}
	return nil
}
