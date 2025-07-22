//go:build e2e
// +build e2e

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

package e2e_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v73/github"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega" //nolint:staticcheck
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gtmv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/tokenmanager"
)

type clientContext struct {
	client  client.Client
	context context.Context
}

func newClientContext(ctx context.Context, k8sClient client.Client) *clientContext {
	return &clientContext{
		client:  k8sClient,
		context: ctx,
	}
}

// waitForPod waits for a pod to be created and enter the running state
func (c *clientContext) waitForPod(inNamespace string, matchingLabels map[string]string) {
	Eventually(func(g Gomega) {
		podList := &corev1.PodList{}
		g.Expect(
			c.client.List(c.context, podList,
				client.InNamespace(inNamespace),
				client.MatchingLabels(matchingLabels),
			),
		).NotTo(HaveOccurred())
		g.Expect(podList.Items).To(HaveLen(1))
		g.Expect(podList.Items[0].Status).ToNot(BeNil())
		g.Expect(podList.Items[0].Status.Phase).To(Equal(corev1.PodRunning))
	}).Within(podReadyTimeout).Should(Succeed())
}

// waitForTokenReconciliation waits for a Token resource to reach status Ready=True
func (c *clientContext) waitForTokenReconciliation(name, namespace string) {
	Eventually(func(g Gomega) {
		tokenObj := &gtmv1.Token{}
		g.Expect(
			c.client.Get(c.context, client.ObjectKey{
				Name:      name,
				Namespace: namespace,
			}, tokenObj),
		).NotTo(HaveOccurred())
		g.Expect(tokenObj.Status.Conditions).To(HaveLen(1))
		g.Expect(tokenObj.Status.Conditions[0].Type).To(Equal(gtmv1.ConditionTypeReady))
		g.Expect(tokenObj.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
	}).Within(reconciliationTimeout).Should(Succeed())
}

// waitForClusterTokenReconciliation waits for a ClusterToken resource to reach status Ready=True
func (c *clientContext) waitForClusterTokenReconciliation(name string) {
	Eventually(func(g Gomega) {
		clusterTokenObj := &gtmv1.ClusterToken{}
		g.Expect(
			c.client.Get(c.context, client.ObjectKey{
				Name: name,
			}, clusterTokenObj),
		).NotTo(HaveOccurred())
		g.Expect(clusterTokenObj.Status.Conditions).To(HaveLen(1))
		g.Expect(clusterTokenObj.Status.Conditions[0].Type).To(Equal(gtmv1.ConditionTypeReady))
		g.Expect(clusterTokenObj.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
	}).Within(reconciliationTimeout).Should(Succeed())
}

// checkManagedSecret waits for a secret to be created and returns its initial token value
func (c *clientContext) checkManagedSecret(
	name, namespace string, //nolint:unparam
	secretType corev1.SecretType,
) (secretValue string) {
	secret := &corev1.Secret{}
	Expect(
		c.client.Get(c.context, client.ObjectKey{
			Name:      name,
			Namespace: namespace,
		}, secret),
	).NotTo(HaveOccurred())

	Expect(secret.Labels).To(HaveKey("app.kubernetes.io/created-by"))
	Expect(secret.Labels["app.kubernetes.io/created-by"]).To(Equal("github-token-manager"))

	switch secretType {
	case tokenmanager.SecretTypeBasicAuth:
		Expect(secret.Type).To(Equal(tokenmanager.SecretTypeBasicAuth))
		Expect(secret.Data).To(HaveKey("username"))
		Expect(secret.Data["username"]).ToNot(BeEmpty())
		Expect(secret.Data).To(HaveKey("password"))
		Expect(secret.Data["password"]).ToNot(BeEmpty())
		secretValue = string(secret.Data["password"])
	case tokenmanager.SecretTypeToken:
		Expect(secret.Type).To(Equal(tokenmanager.SecretTypeToken))
		Expect(secret.Data).To(HaveKey("token"))
		Expect(secret.Data["token"]).ToNot(BeEmpty())
		secretValue = string(secret.Data["token"])
	}

	return secretValue
}

// checkManagedSecretRotation waits for a token to be refreshed and returns the refreshed token
func (c *clientContext) checkManagedSecretRotation(
	secretName, namespace string, //nolint:unparam
	secretType corev1.SecretType,
	oldSecretValue string,
) (newSecretValue string) {
	Eventually(func(g Gomega) {
		secret := &corev1.Secret{}
		g.Expect(
			c.client.Get(c.context, client.ObjectKey{
				Name:      secretName,
				Namespace: namespace,
			}, secret),
		).NotTo(HaveOccurred())

		switch secretType {
		case tokenmanager.SecretTypeBasicAuth:
			g.Expect(secret.Data).To(HaveKey("password"))
			g.Expect(secret.Data["password"]).ToNot(BeEmpty())
			newSecretValue = string(secret.Data["password"])
		case tokenmanager.SecretTypeToken:
			g.Expect(secret.Data).To(HaveKey("token"))
			g.Expect(secret.Data["token"]).ToNot(BeEmpty())
			newSecretValue = string(secret.Data["token"])
		}

		g.Expect(newSecretValue).To(Not(Equal(oldSecretValue)))
	}).Within(secretCheckTimeout).ProbeEvery(secretCheckInterval).Should(Succeed())

	return newSecretValue
}

// createNamespace creates a namespace for testing
func (c *clientContext) createNamespace(name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return c.client.Create(c.context, ns)
}

// deleteNamespace deletes a namespace
func (c *clientContext) deleteNamespace(name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return c.client.Delete(c.context, ns)
}

// createToken creates a standard Token resource for testing
func (c *clientContext) createToken(
	name, namespace, secretName string,
	isBasicAuth bool,
	refreshInterval time.Duration,
) error {
	token := &gtmv1.Token{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "github.as-code.io/v1",
			Kind:       "Token",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gtmv1.TokenSpec{
			RefreshInterval: metav1.Duration{Duration: refreshInterval},
			Secret: gtmv1.TokenSecretSpec{
				Name:      secretName,
				BasicAuth: isBasicAuth,
			},
			Repositories: []string{
				testRepositoryName,
			},
			Permissions: &gtmv1.Permissions{
				Contents: &readPermission,
				Metadata: &readPermission,
			},
		},
	}

	return c.client.Create(c.context, token)
}

// deleteToken deletes a Token resource
func (c *clientContext) deleteToken(name, namespace string) error {
	token := &gtmv1.Token{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return c.client.Delete(c.context, token)
}

// createClusterToken creates a standard ClusterToken resource for testing
func (c *clientContext) createClusterToken(
	name, secretName, targetNamespace string,
	isBasicAuth bool,
	refreshInterval time.Duration,
) error {
	clusterToken := &gtmv1.ClusterToken{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "github.as-code.io/v1",
			Kind:       "ClusterToken",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: gtmv1.ClusterTokenSpec{
			RefreshInterval: metav1.Duration{Duration: refreshInterval},
			Secret: gtmv1.ClusterTokenSecretSpec{
				Name:      secretName,
				Namespace: targetNamespace,
				BasicAuth: isBasicAuth,
			},
			Repositories: []string{
				testRepositoryName,
			},
			Permissions: &gtmv1.Permissions{
				Contents: &readPermission,
			},
		},
	}
	return c.client.Create(c.context, clusterToken)
}

// deleteClusterToken deletes a ClusterToken resource
func (c *clientContext) deleteClusterToken(name string) error {
	clusterToken := &gtmv1.ClusterToken{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return c.client.Delete(c.context, clusterToken)
}

// runCommand executes the provided command within this context
func runCommand(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := getProjectDir()
	cmd.Dir = dir

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "running: %s\n", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return stdout.Bytes(), fmt.Errorf("%s failed with error: (%v) stderr: %s", command, err, stderrStr)
		}
		return stdout.Bytes(), fmt.Errorf("%s failed with error: (%v)", command, err)
	}

	return stdout.Bytes(), nil
}

// loadImageToKindCluster loads a local docker image to the kind cluster
func loadImageToKindCluster(name, cluster string) error {
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := runCommand(cmd)
	return err
}

// getProjectDir will return the directory where the project is
func getProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// generateTestKey generates a new RSA private key and returns it as a PEM-encoded string
func generateTestKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return string(privateKeyPEM), nil
}

// checkToken validates that a GitHub token can read content from the specified repository
// by making a simple API call to get the repository's README.
// Returns nil if the token is valid, error otherwise.
func checkToken(repository, token string) error {
	ctx := context.Background()

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create GitHub client
	client := github.NewClient(tc)

	// Parse repository string (format: "owner/repo")
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format, expected 'owner/repo', got '%s'", repository)
	}
	owner, repo := parts[0], parts[1]

	// Test the /repos/OWNER/REPO/readme endpoint to validate content read permissions
	_, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err != nil {
		return fmt.Errorf("failed to validate token for repository %s (readme endpoint): %w", repository, err)
	}

	return nil
}

// newTokenValidator creates a new GitHub token validator for the specified repository.
// Returns a function that can be used to validate tokens.
func newTokenValidator(repository string) func(string) error {
	return func(token string) error {
		return checkToken(repository, token)
	}
}
