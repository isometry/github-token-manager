/*
Copyright 2024 Robin Breathe.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/ghapp"
)

// AppKeyRefIndex is the field-indexer key used to watch Secrets and map them
// back to the Apps that reference them via spec.keyRef.name.
const AppKeyRefIndex = ".spec.keyRef.name"

// defaultKeyRefDataKey matches the kubebuilder default on
// AppSpec.KeyRef.Key; mirrored here so the in-process Get path agrees with
// any object that bypassed defaulting (e.g. tests using the typed client).
const defaultKeyRefDataKey = "private-key.pem"

// resolveAppConfig returns an [*ghapp.OperatorConfig] for the given App
// together with a version string capturing every input that affects client
// identity.
//
// Cloud-KMS configs pass through unchanged with the version set to the App's
// spec generation. provider:"secret" configs translate to ghait's file
// provider with the literal PEM bytes in Key (its os.Stat fallback handles
// literal PEM bytes), and the version composes the spec generation with the
// referenced Secret's ResourceVersion so cached clients invalidate on key
// rotation.
//
// reason is one of the v1.Reason* constants when err is non-nil, suitable
// for the caller to write into a status condition.
func resolveAppConfig(ctx context.Context, c client.Reader, app *githubv1.App) (cfg *ghapp.OperatorConfig, version, reason string, err error) {
	cfg = &ghapp.OperatorConfig{
		AppID:          app.Spec.AppID,
		InstallationID: app.Spec.InstallationID,
		Provider:       app.Spec.Provider,
		Key:            app.Spec.Key,
		ValidateKey:    app.Spec.ValidateKey,
	}

	if app.Spec.Provider != "secret" {
		return cfg, strconv.FormatInt(app.Generation, 10), "", nil
	}

	dataKey := app.Spec.KeyRef.Key
	if dataKey == "" {
		dataKey = defaultKeyRefDataKey
	}

	var secret corev1.Secret
	nn := types.NamespacedName{Namespace: app.Namespace, Name: app.Spec.KeyRef.Name}
	if err := c.Get(ctx, nn, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, "", githubv1.ReasonSecretNotFound, fmt.Errorf("Secret %s not found: %w", nn, err)
		}
		return nil, "", githubv1.ReasonSetupFailed, fmt.Errorf("fetch Secret %s: %w", nn, err)
	}

	pemBytes, ok := secret.Data[dataKey]
	if !ok || len(pemBytes) == 0 {
		return nil, "", githubv1.ReasonInvalidKey, fmt.Errorf("Secret %s has no data under key %q", nn, dataKey)
	}

	cfg.Provider = "file"
	cfg.Key = string(pemBytes)
	return cfg, fmt.Sprintf("%d:%s", app.Generation, secret.ResourceVersion), "", nil
}
