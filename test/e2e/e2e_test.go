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

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gtmv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/tokenmanager"
	"github.com/isometry/github-token-manager/test/utils"
)

const (
	// Operator namespace and configuration
	operatorNamespace   = "github-token-manager"
	testRepositoryOwner = "isometry"
	testRepositoryName  = "github-token-manager"
	testRepository      = testRepositoryOwner + "/" + testRepositoryName

	// Test timeouts and intervals
	podReadyTimeout       = 2 * time.Minute
	reconciliationTimeout = 10 * time.Second
	tokenRefreshInterval  = 10 * time.Second
	secretCheckTimeout    = 2 * tokenRefreshInterval
	secretCheckInterval   = 1 * time.Second

	// Test namespace
	targetNamespace = "test-tokens"

	// Resource names
	testToken1        = "token-1"
	testToken2        = "token-2"
	testClusterToken1 = "cluster-token-1"
	testClusterToken2 = "cluster-token-2"

	// Secret names
	testSecret1 = "secret-1"
	testSecret2 = "secret-2"
	testSecret3 = "secret-3"
	testSecret4 = "secret-4"
)

// Permission strings (var to allow taking address)
var readPermission = "read"

var _ = Describe("GitHub Token Manager", Ordered, func() {
	var projectImage string
	var hasRealConfig bool
	var k8sClient client.Client
	ctx := context.Background()
	var clientCtx *clientContext
	checkToken := utils.NewGitHubTokenValidator(testRepository)

	BeforeAll(func() {
		scheme := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(gtmv1.AddToScheme(scheme)).To(Succeed())

		// Get Kind cluster name from environment
		kubeContext := os.Getenv("KUBE_CONTEXT")
		Expect(kubeContext).NotTo(BeEmpty())

		// Build kubeconfig for the specific Kind cluster context
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeconfig, err := clientcmd.LoadFromFile(loadingRules.GetDefaultFilename())
		Expect(err).NotTo(HaveOccurred())

		config, err := clientcmd.NewDefaultClientConfig(*kubeconfig, &clientcmd.ConfigOverrides{
			CurrentContext: kubeContext,
		}).ClientConfig()
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(config, client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred())

		clientCtx = newClientContext(ctx, k8sClient)

		cmd := exec.Command("ko", "build", "--local", "./cmd/manager")
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		projectImage = string(output)[:len(output)-1] // Remove newline

		if strings.HasPrefix(kubeContext, "kind-") {
			By("loading the manager image to Kind")
			err = utils.LoadImageToKindCluster(projectImage, strings.TrimPrefix(kubeContext, "kind-"))
			Expect(err).NotTo(HaveOccurred())
		}

		By("creating token namespace")
		err = clientCtx.createNamespace(targetNamespace)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Helm Chart", func() {
		It("should install cleanly", func() {
			ctx = context.Background()

			By("checking for valid GitHub App credentials")
			projectDir, err := utils.GetProjectDir()
			Expect(err).NotTo(HaveOccurred())
			valuesPath := filepath.Join(projectDir, "test", "e2e", "values.yaml")

			if _, err := os.Stat(valuesPath); err == nil {
				hasRealConfig = true
				GinkgoWriter.Printf("Found real config values at %s\n", valuesPath)
			} else {
				GinkgoWriter.Printf("No real config found at %s, using test config\n", valuesPath)
			}

			chartPath := filepath.Join(projectDir, "deploy", "charts", "github-token-manager")
			valuesArgs := []string{
				"helm", "upgrade", "--install", "github-token-manager", chartPath,
				"--namespace", operatorNamespace,
				"--create-namespace",
			}

			if hasRealConfig {
				valuesArgs = append(valuesArgs,
					"--values", valuesPath,
				)
			} else {
				// Use test config values
				privateKey, err := utils.GenerateRSAPrivateKey()
				Expect(err).NotTo(HaveOccurred())
				valuesArgs = append(valuesArgs,
					"--set", "config.app_id=123456",
					"--set", "config.installation_id=789012",
					"--set", "config.provider=file",
					"--set", fmt.Sprintf("config.key=%s", privateKey),
				)
			}

			// Always override the manager image settings regardless of config source
			valuesArgs = append(valuesArgs,
				"--set", fmt.Sprintf("manager.repository=%s", strings.Split(projectImage, ":")[0]),
				"--set", fmt.Sprintf("manager.tag=%s", strings.Split(projectImage, ":")[1]),
			)

			By("installing operator using Helm chart")
			cmd := exec.Command(valuesArgs[0], valuesArgs[1:]...)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for manager pod to be ready")
			clientCtx.waitForPod(
				operatorNamespace,
				map[string]string{
					"app.kubernetes.io/name":     "github-token-manager",
					"app.kubernetes.io/instance": "github-token-manager",
				},
			)
		})
	})

	Context("Token CR", func() {
		It("supports managing Secrets of type github.as-code.io/token", func() {
			if !hasRealConfig {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a Token resource with basicAuth=false")
			_, err := clientCtx.createToken(testToken1, targetNamespace, testSecret1, false, tokenRefreshInterval)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for Token reconciliation")
			clientCtx.waitForTokenReconciliation(testToken1, targetNamespace)

			By("checking managed Secret is created correctly")
			initialSecretToken := clientCtx.checkManagedSecret(testSecret1, targetNamespace, tokenmanager.SecretTypeToken)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialSecretToken)).To(Succeed())

			By("checking managed Secret token values are rotated")
			rotatedSecretToken := clientCtx.checkManagedSecretRotation(testSecret1, targetNamespace, tokenmanager.SecretTypeToken, initialSecretToken)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(rotatedSecretToken)).To(Succeed())

			By("deleting the Token resource")
			Expect(clientCtx.deleteToken(testToken1, targetNamespace)).To(Succeed())
		})

		It("supports managing Secrets of type github.as-code.io/basic-auth", func() {
			if !hasRealConfig {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a Token resource with basicAuth=true")
			_, err := clientCtx.createToken(testToken2, targetNamespace, testSecret2, true, tokenRefreshInterval)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for Token reconciliation")
			clientCtx.waitForTokenReconciliation(testToken2, targetNamespace)

			By("checking managed Secret is created correctly")
			initialSecretToken := clientCtx.checkManagedSecret(testSecret2, targetNamespace, tokenmanager.SecretTypeBasicAuth)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialSecretToken)).To(Succeed())

			By("checking managed Secret token values are rotated")
			rotatedSecretToken := clientCtx.checkManagedSecretRotation(testSecret2, targetNamespace, tokenmanager.SecretTypeBasicAuth, initialSecretToken)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(rotatedSecretToken)).To(Succeed())

			By("deleting the Token resource")
			Expect(clientCtx.deleteToken(testToken2, targetNamespace)).To(Succeed())
		})
	})

	Context("ClusterToken CR", func() {
		It("supports managing Secrets of type github.as-code.io/token", func() {
			if !hasRealConfig {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a ClusterToken resource with basicAuth=false")
			_, err := clientCtx.createTestClusterTokenResource(testClusterToken1, testSecret3, targetNamespace, false, tokenRefreshInterval)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for ClusterToken reconciliation")
			clientCtx.waitForClusterTokenReconciliation(testClusterToken1)

			By("checking managed Secret is created correctly")
			initialToken := clientCtx.checkManagedSecret(testSecret3, targetNamespace, tokenmanager.SecretTypeToken)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialToken)).To(Succeed())

			By("checking managed Secret token values are rotated")
			refreshedTokenValue := clientCtx.checkManagedSecretRotation(testSecret3, targetNamespace, tokenmanager.SecretTypeToken, initialToken)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(refreshedTokenValue)).To(Succeed())

			By("deleting the ClusterToken resource")
			Expect(clientCtx.deleteClusterToken(testClusterToken1)).To(Succeed())
		})

		It("supports managing Secrets of type github.as-code.io/basic-auth", func() {
			if !hasRealConfig {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a ClusterToken resource with basicAuth=true")
			_, err := clientCtx.createTestClusterTokenResource(testClusterToken2, testSecret4, targetNamespace, true, tokenRefreshInterval)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for ClusterToken reconciliation")
			clientCtx.waitForClusterTokenReconciliation(testClusterToken2)

			By("checking managed Secret is created correctly")
			initialTokenValue := clientCtx.checkManagedSecret(testSecret4, targetNamespace, tokenmanager.SecretTypeBasicAuth)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialTokenValue)).To(Succeed())

			By("checking managed Secret token values are rotated")
			updatedTokenValue := clientCtx.checkManagedSecretRotation(testSecret4, targetNamespace, tokenmanager.SecretTypeBasicAuth, initialTokenValue)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(updatedTokenValue)).To(Succeed())

			By("deleting the ClusterToken resource")
			err = clientCtx.deleteClusterToken(testClusterToken2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Helm Chart", func() {
		It("should uninstall without error", func() {
			By("uninstalling Helm chart")
			cmd := exec.Command("helm", "uninstall", "github-token-manager", "--namespace", operatorNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

		})
	})

	AfterAll(func() {
		Expect(clientCtx.deleteNamespace(operatorNamespace)).To(Succeed())
		Expect(clientCtx.deleteNamespace(targetNamespace)).To(Succeed())
	})
})
