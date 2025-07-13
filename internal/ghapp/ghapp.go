package ghapp

import (
	"context"
	"fmt"

	"github.com/isometry/ghait"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
