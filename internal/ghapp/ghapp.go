package ghapp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v61/github"
)

type GHApp struct {
	AppID          int64
	InstallationID int64
	Client         *github.Client
}

func NewGHApp(appID int64, privateKey []byte, installationID int64) (*GHApp, error) {
	transport, err := ghinstallation.NewAppsTransport(
		http.DefaultTransport,
		appID,
		privateKey,
	)
	if err != nil {
		return nil, err
	}

	return &GHApp{
		AppID:          appID,
		InstallationID: installationID,
		Client:         github.NewClient(&http.Client{Transport: transport}),
	}, nil
}

func NewGHAppFromConfig() (*GHApp, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return NewGHApp(cfg.AppID, []byte(cfg.PrivateKey), cfg.InstallationID)
}

func (g *GHApp) CreateInstallationToken(ctx context.Context, installationId int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	if installationId == 0 {
		if g.InstallationID == 0 {
			return nil, fmt.Errorf("no GitHub App Installation ID configured")
		}
		installationId = g.InstallationID
	}

	installationToken, _, err := g.Client.Apps.CreateInstallationToken(ctx, installationId, options)
	if err != nil {
		return nil, err
	}

	return installationToken, nil
}
