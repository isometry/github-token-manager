package ghapp

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/isometry/ghait/v84"
	_ "github.com/isometry/ghait/v84/provider/aws"   // AWS KMS provider
	_ "github.com/isometry/ghait/v84/provider/azure" // Azure Key Vault provider
	_ "github.com/isometry/ghait/v84/provider/gcp"   // GCP KMS provider
	_ "github.com/isometry/ghait/v84/provider/vault" // HashiCorp Vault provider
)

func NewGHApp(ctx context.Context) (ghait.GHAIT, error) {
	logger := log.FromContext(ctx)

	cfg, err := LoadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("configuration: %w", err)
	}

	logger.Info("loaded configuration", "config", cfg)

	ghapp, err := ghait.NewGHAIT(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("ghait: %w", err)
	}

	return ghapp, nil
}
