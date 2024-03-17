package ghapp

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

const (
	ConfigPath         = "/config"
	AppIdFile          = "app_id"
	InstallationIdFile = "installation_id"
	PrivateKeyFile     = "private_key.pem"
)

type Config struct {
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
}

func LoadConfig() (*Config, error) {
	appIdPath := path.Join(ConfigPath, AppIdFile)
	rawAppID, err := os.ReadFile(appIdPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", appIdPath, err)
	}

	appID, err := strconv.ParseInt(string(rawAppID), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s as int64: %w", appIdPath, err)
	}

	privateKeyPath := path.Join(ConfigPath, PrivateKeyFile)
	rawPrivateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", privateKeyPath, err)
	}

	config := &Config{
		AppID:      appID,
		PrivateKey: rawPrivateKey,
	}

	// Load installation ID if it exists.
	installationIdPath := path.Join(ConfigPath, InstallationIdFile)
	if _, err := os.Stat(installationIdPath); err == nil {
		rawInstallationId, err := os.ReadFile(installationIdPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read %s: %w", installationIdPath, err)
		}
		installationId, err := strconv.ParseInt(string(rawInstallationId), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse %s as int64: %w", installationIdPath, err)
		}
		config.InstallationID = installationId
	}

	return config, nil
}
