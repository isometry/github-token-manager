package ghapp

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ConfigPath = "/config"
	ConfigName = "gtm"
)

type OperatorConfig struct {
	AppID          int64  `mapstructure:"app_id"`
	InstallationID int64  `mapstructure:"installation_id"`
	Provider       string `mapstructure:"provider"`
	Key            string `mapstructure:"key"`
}

func (c *OperatorConfig) GetAppID() int64 {
	return c.AppID
}

func (c *OperatorConfig) GetInstallationID() int64 {
	return c.InstallationID
}

func (c *OperatorConfig) GetProvider() string {
	return c.Provider
}

func (c *OperatorConfig) GetKey() string {
	return c.Key
}

// TokenValidity is the duration for which a token is valid. Always exactly 1 hour.
const TokenValidity = time.Hour

func LoadConfig(ctx context.Context) (*OperatorConfig, error) {
	log := log.FromContext(ctx)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("GTM")

	_ = viper.BindEnv("app_id", "GTM_APP_ID", "GITHUB_APP_ID")
	_ = viper.BindEnv("installation_id", "GTM_INSTALLATION_ID", "GITHUB_INSTALLATION_ID")
	_ = viper.BindEnv("provider", "GTM_PROVIDER", "KMS_PROVIDER")
	_ = viper.BindEnv("key", "GTM_KEY", "KMS_KEY", "GITHUB_PRIVATE_KEY")

	viper.SetDefault("provider", "file")

	viper.AddConfigPath(ConfigPath)
	viper.SetConfigName(ConfigName)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading configuration file: %w", err)
		} else {
			log.Info("no configuration file found, continuing with environment variables only")
		}
	}

	config := &OperatorConfig{}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}
