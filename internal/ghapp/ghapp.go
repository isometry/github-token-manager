package ghapp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v60/github"
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

	return NewGHApp(cfg.AppID, cfg.PrivateKey, cfg.InstallationID)
}

func (g *GHApp) NewToken(ctx context.Context, installationId int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	if installationId == 0 {
		if g.InstallationID == 0 {
			return nil, fmt.Errorf("installation ID not provided")
		}
		installationId = g.InstallationID
	}

	token, _, err := g.Client.Apps.CreateInstallationToken(ctx, installationId, options)
	if err != nil {
		return nil, err
	}

	return token, nil
}
