package controller

import (
	"github.com/google/go-github/v61/github"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/isometry/github-token-manager/internal/ghapp"
	tm "github.com/isometry/github-token-manager/internal/token_manager"
)

// Definitions to manage status conditions
const (
	// conditionTypeAvailable represents the status of the Secret reconciliation
	conditionTypeAvailable = "Available"
)

var (
	app *ghapp.GHApp // cached GHApp instance
)

const SecretTypeGithubToken = "github.as-code.io/token"

// newSecretForToken returns a new Secret object containing the credentials for the Token
func newSecretForToken(token tm.TokenManager, scheme *runtime.Scheme, installationToken *github.InstallationToken) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      token.GetSecretName(),
			Namespace: token.GetSecretNamespace(),
			Labels:    labelsForToken(token.GetName()),
		},
		Type: SecretTypeGithubToken,
		Data: token.SecretData(installationToken),
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(token, secret, scheme); err != nil {
		return nil, err
	}
	return secret, nil
}
