package ghapp

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

var (
	ConfigPath = "/config"
	ConfigName = "github-token-manager"
	// AppIdFile          = "app_id"
	// InstallationIdFile = "installation_id"
	// PrivateKeyFile     = "private_key"
)

type Config struct {
	AppID          int64  `mapstructure:"appId"`
	PrivateKey     []byte `mapstructure:"privateKey"`
	InstallationID int64  `mapstructure:"installationId"`
}

// TokenValidity is the duration for which a token is valid. Always exactly 1 hour.
const TokenValidity = time.Hour

func LoadConfig() (*Config, error) {
	// appIdPath := path.Join(ConfigPath, AppIdFile)
	// rawAppID, err := os.ReadFile(appIdPath)
	// if err != nil {
	// 	return nil, fmt.Errorf("unable to read %s: %w", appIdPath, err)
	// }

	// appID, err := strconv.ParseInt(string(rawAppID), 10, 64)
	// if err != nil {
	// 	return nil, fmt.Errorf("unable to parse %s as int64: %w", appIdPath, err)
	// }

	// privateKeyPath := path.Join(ConfigPath, PrivateKeyFile)
	// rawPrivateKey, err := os.ReadFile(privateKeyPath)
	// if err != nil {
	// 	return nil, fmt.Errorf("unable to read %s: %w", privateKeyPath, err)
	// }

	// config := &Config{
	// 	AppID:      appID,
	// 	PrivateKey: rawPrivateKey,
	// }

	// // Load installation ID if it exists.
	// installationIdPath := path.Join(ConfigPath, InstallationIdFile)
	// if _, err := os.Stat(installationIdPath); err == nil {
	// 	rawInstallationId, err := os.ReadFile(installationIdPath)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("unable to read %s: %w", installationIdPath, err)
	// 	}
	// 	installationId, err := strconv.ParseInt(string(rawInstallationId), 10, 64)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("unable to parse %s as int64: %w", installationIdPath, err)
	// 	}
	// 	config.InstallationID = installationId
	// }

	viper.AutomaticEnv()

	viper.AddConfigPath(ConfigPath)
	viper.SetConfigName(ConfigName)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read config: %w", err)
	}

	config := &Config{}

	if err := viper.Unmarshal(config); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("invalid configuration file: %w", err)
		}
	}

	if config.AppID == 0 {
		return nil, fmt.Errorf("configuration: app_id is required")
	}

	// Check that the PrivateKey is a valid PEM-encoded RSA private key.
	if err := config.CheckPrivateKey(); err != nil {
		return nil, fmt.Errorf("configuration: invalid private_key: %w", err)
	}

	return config, nil
}

func (c *Config) CheckPrivateKey() error {
	if c.PrivateKey == nil {
		return errors.New("unset")
	}

	block, _ := pem.Decode(c.PrivateKey)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return errors.New("failed to decode PEM")
	}

	if _, err := x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
		return fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	return nil
}
