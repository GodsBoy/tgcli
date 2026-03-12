package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DefaultStoreDir returns the default storage directory (~/.tgcli).
func DefaultStoreDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".tgcli")
	}
	return filepath.Join(home, ".tgcli")
}

// Config holds application-level configuration.
type Config struct {
	AppID   int    `json:"app_id"`
	AppHash string `json:"app_hash"`
	Phone   string `json:"phone"`
}

// Load reads config from path. Returns zero Config if the file doesn't exist.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes config to path.
func Save(path string, c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
