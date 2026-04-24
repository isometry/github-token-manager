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
	"cmp"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	gtmv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/tokenmanager"
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
	testToken3        = "token-3"
	testClusterToken1 = "cluster-token-1"
	testClusterToken2 = "cluster-token-2"
	testApp           = "test-app"

	// Secret names
	testSecret1 = "secret-1"
	testSecret2 = "secret-2"
	testSecret3 = "secret-3"
	testSecret4 = "secret-4"
	testSecret5 = "secret-5"

	testAppKeySecret = "test-app-key"
)

// gtmConfig holds the GitHub App credentials captured from the Helm install
// step so that later specs (notably the App CR test) can reuse them without
// re-parsing env vars or values files.
type gtmConfig struct {
	appID          int64
	installationID int64
	provider       string
	key            string
}

// Permission strings (var to allow taking address)
var readPermission = "read"

// e2ePreserveState reports whether E2E_PRESERVE is set in the environment.
// When true, the suite skips the uninstall spec and the namespace teardown in
// AfterAll so that a developer can inspect cluster state after a run.
func e2ePreserveState() bool {
	return os.Getenv("E2E_PRESERVE") != ""
}

var _ = Describe("GitHub Token Manager", Ordered, func() {
	var kubeContext, testImage, testRepo, testTag string
	var hasAppCredentials bool
	var capturedConfig *gtmConfig
	var k8sClient client.Client
	ctx := context.Background()
	var clientCtx *clientContext
	checkToken := newTokenValidator(testRepository)

	BeforeAll(func() {
		scheme := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(gtmv1.AddToScheme(scheme)).To(Succeed())

		// Get Kind cluster name from environment
		kubeContext = os.Getenv("KUBE_CONTEXT")
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

		By("creating token namespace")
		Expect(clientCtx.createNamespace(targetNamespace)).To(Succeed())
	})

	Context("Container Image", func() {
		It("builds successfully", func() {
			By("building the manager image")
			cmd := exec.Command("ko", "build", "--local", "./cmd/manager")
			output, err := runCommand(cmd)
			Expect(err).NotTo(HaveOccurred())
			testImage = strings.TrimSpace(string(output))
			imageParts := strings.SplitN(testImage, ":", 2)
			testRepo = imageParts[0]
			testTag = imageParts[1]

			if strings.HasPrefix(kubeContext, "kind-") {
				By("loading the manager image to Kind")
				Expect(loadImageToKindCluster(testImage, strings.TrimPrefix(kubeContext, "kind-"))).To(Succeed())
			}
		})
	})

	Context("Helm Chart", func() {
		It("installs cleanly", func() {
			ctx = context.Background()

			By("checking for valid GitHub App credentials")
			projectDir, err := getProjectDir()
			Expect(err).NotTo(HaveOccurred())
			valuesPath := filepath.Join(projectDir, "test", "e2e", "values.yaml")

			chartPath := filepath.Join(projectDir, "deploy", "charts", "github-token-manager")
			valuesArgs := []string{
				"helm", "upgrade", "--install", "github-token-manager", chartPath,
				"--namespace", operatorNamespace,
				"--create-namespace",
				"--timeout=60s",
				"--wait=watcher",
				fmt.Sprintf("--set=manager.repository=%s", testRepo),
				fmt.Sprintf("--set=manager.tag=%s", testTag),
			}

			gtmAppId := os.Getenv("GTM_APP_ID")
			gtmInstallationId := os.Getenv("GTM_INSTALLATION_ID")
			gtmProvider := cmp.Or(os.Getenv("GTM_PROVIDER"), "file")
			gtmKey := os.Getenv("GTM_KEY")

			// Check for credentials in the following priority order:
			// 1. Local values.yaml file
			// 2. Environment variables (for CI/GitHub Actions)
			if _, err := os.Stat(valuesPath); err == nil {
				hasAppCredentials = true
				GinkgoWriter.Printf("Using GitHub App configuration values from %q\n", valuesPath)
				cfg, err := parseHelmValuesFile(valuesPath)
				Expect(err).NotTo(HaveOccurred())
				capturedConfig = cfg
				valuesArgs = append(valuesArgs,
					fmt.Sprintf("--values=%s", valuesPath),
				)
			} else if gtmAppId != "" && gtmInstallationId != "" && gtmKey != "" {
				hasAppCredentials = true
				GinkgoWriter.Println("Using GitHub App credentials from the environment")

				appID, err := strconv.ParseInt(gtmAppId, 10, 64)
				Expect(err).NotTo(HaveOccurred(), "parsing GTM_APP_ID")
				installationID, err := strconv.ParseInt(gtmInstallationId, 10, 64)
				Expect(err).NotTo(HaveOccurred(), "parsing GTM_INSTALLATION_ID")
				capturedConfig = &gtmConfig{
					appID:          appID,
					installationID: installationID,
					provider:       gtmProvider,
					key:            gtmKey,
				}

				// Create temporary values file from environment variables
				envValuesPath := filepath.Join(projectDir, "test", "e2e", "values.env.yaml")

				// Structure matching the Helm values schema
				values := map[string]any{
					"config": map[string]string{
						"app_id":          gtmAppId,
						"installation_id": gtmInstallationId,
						"provider":        gtmProvider,
						"key":             gtmKey,
					},
				}

				valuesYAML, err := yaml.Marshal(values)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(envValuesPath, valuesYAML, 0600)).To(Succeed())
				defer func() {
					_ = os.Remove(envValuesPath)
				}()

				valuesArgs = append(valuesArgs,
					fmt.Sprintf("--values=%s", envValuesPath),
				)
			} else {
				GinkgoWriter.Printf("No real config found at %q or in the environment, generaging dumbie config\n", valuesPath)
				// Use test config values
				privateKey, err := generateTestKey()
				Expect(err).NotTo(HaveOccurred())
				valuesArgs = append(valuesArgs,
					"--set=config.app_id=123456",
					"--set=config.installation_id=789012",
					"--set=config.provider=file",
					fmt.Sprintf("--set-string=config.key=%s", privateKey),
				)
			}

			By("installing operator using Helm chart")
			cmd := exec.Command(valuesArgs[0], valuesArgs[1:]...)
			_, err = runCommand(cmd)
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
		It("manages Secrets of type github.as-code.io/token", func() {
			if !hasAppCredentials {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a Token resource with basicAuth=false")
			Expect(clientCtx.createToken(testToken1, targetNamespace, testSecret1, false, tokenRefreshInterval)).To(Succeed())

			By("waiting for Token reconciliation")
			clientCtx.waitForTokenReconciliation(testToken1, targetNamespace)

			By("checking managed Secret is created correctly")
			initialSecretToken := clientCtx.checkManagedSecret(testSecret1, targetNamespace, tokenmanager.SecretTypeToken)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialSecretToken)).To(Succeed())

			By("checking managed Secret token values are rotated")
			rotatedSecretToken := clientCtx.checkManagedSecretRotation(
				testSecret1,
				targetNamespace,
				tokenmanager.SecretTypeToken,
				initialSecretToken,
			)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(rotatedSecretToken)).To(Succeed())

			By("deleting the Token resource")
			Expect(clientCtx.deleteToken(testToken1, targetNamespace)).To(Succeed())
		})

		It("manages Secrets of type github.as-code.io/basic-auth", func() {
			if !hasAppCredentials {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a Token resource with basicAuth=true")
			Expect(clientCtx.createToken(testToken2, targetNamespace, testSecret2, true, tokenRefreshInterval)).To(Succeed())

			By("waiting for Token reconciliation")
			clientCtx.waitForTokenReconciliation(testToken2, targetNamespace)

			By("checking managed Secret is created correctly")
			initialSecretToken := clientCtx.checkManagedSecret(testSecret2, targetNamespace, tokenmanager.SecretTypeBasicAuth)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialSecretToken)).To(Succeed())

			By("checking managed Secret token values are rotated")
			rotatedSecretToken := clientCtx.checkManagedSecretRotation(
				testSecret2,
				targetNamespace,
				tokenmanager.SecretTypeBasicAuth,
				initialSecretToken,
			)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(rotatedSecretToken)).To(Succeed())

			By("deleting the Token resource")
			Expect(clientCtx.deleteToken(testToken2, targetNamespace)).To(Succeed())
		})
	})

	Context("ClusterToken CR", func() {
		It("manages Secrets of type github.as-code.io/token", func() {
			if !hasAppCredentials {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a ClusterToken resource with basicAuth=false")
			Expect(clientCtx.createClusterToken(
				testClusterToken1,
				testSecret3,
				targetNamespace,
				false,
				tokenRefreshInterval,
			)).To(Succeed())

			By("waiting for ClusterToken reconciliation")
			clientCtx.waitForClusterTokenReconciliation(testClusterToken1)

			By("checking managed Secret is created correctly")
			initialToken := clientCtx.checkManagedSecret(testSecret3, targetNamespace, tokenmanager.SecretTypeToken)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialToken)).To(Succeed())

			By("checking managed Secret token values are rotated")
			refreshedTokenValue := clientCtx.checkManagedSecretRotation(
				testSecret3,
				targetNamespace,
				tokenmanager.SecretTypeToken,
				initialToken,
			)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(refreshedTokenValue)).To(Succeed())

			By("deleting the ClusterToken resource")
			Expect(clientCtx.deleteClusterToken(testClusterToken1)).To(Succeed())
		})

		It("manages Secrets of type github.as-code.io/basic-auth", func() {
			if !hasAppCredentials {
				Skip("skipping tests - no valid GitHub App configuration provided")
			}

			By("creating a ClusterToken resource with basicAuth=true")
			Expect(clientCtx.createClusterToken(
				testClusterToken2,
				testSecret4,
				targetNamespace,
				true,
				tokenRefreshInterval,
			)).To(Succeed())

			By("waiting for ClusterToken reconciliation")
			clientCtx.waitForClusterTokenReconciliation(testClusterToken2)

			By("checking managed Secret is created correctly")
			initialTokenValue := clientCtx.checkManagedSecret(testSecret4, targetNamespace, tokenmanager.SecretTypeBasicAuth)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialTokenValue)).To(Succeed())

			By("checking managed Secret token values are rotated")
			updatedTokenValue := clientCtx.checkManagedSecretRotation(
				testSecret4,
				targetNamespace,
				tokenmanager.SecretTypeBasicAuth,
				initialTokenValue,
			)

			By("checking that the rotated Secret token value is valid")
			Expect(checkToken(updatedTokenValue)).To(Succeed())

			By("deleting the ClusterToken resource")
			Expect(clientCtx.deleteClusterToken(testClusterToken2)).To(Succeed())
		})
	})

	Context("App CR", Ordered, func() {
		AfterAll(func() {
			if !hasAppCredentials {
				return
			}
			_ = clientCtx.deleteApp(testApp, targetNamespace)
			_ = clientCtx.deleteSecret(testAppKeySecret, targetNamespace)
		})

		It("reconciles an App resource to Ready=True", func() {
			if !hasAppCredentials {
				Skip("skipping App test - no valid GitHub App configuration provided")
			}

			By("creating a Secret holding the GitHub App private key")
			Expect(clientCtx.createOpaqueSecret(testAppKeySecret, targetNamespace, map[string][]byte{
				"private-key.pem": []byte(capturedConfig.key),
			})).To(Succeed())

			By("creating a Secret-backed App resource referencing it")
			Expect(clientCtx.createApp(testApp, targetNamespace, gtmv1.AppSpec{
				AppID:          capturedConfig.appID,
				InstallationID: capturedConfig.installationID,
				Provider:       "secret",
				KeyRef:         &gtmv1.KeySecretReference{Name: testAppKeySecret},
			})).To(Succeed())

			By("waiting for App reconciliation")
			clientCtx.waitForAppReconciliation(testApp, targetNamespace)
		})

		It("manages a Token that references the App via spec.appRef", func() {
			if !hasAppCredentials {
				Skip("skipping App test - no valid GitHub App configuration provided")
			}

			By("creating a Token with spec.appRef pointing at the App")
			Expect(clientCtx.createTokenWithAppRef(
				testToken3, targetNamespace, testSecret5, testApp,
				false, tokenRefreshInterval,
			)).To(Succeed())

			By("waiting for Token reconciliation")
			clientCtx.waitForTokenReconciliation(testToken3, targetNamespace)

			By("checking managed Secret is created correctly")
			initialSecretToken := clientCtx.checkManagedSecret(testSecret5, targetNamespace, tokenmanager.SecretTypeToken)

			By("checking that the managed Secret token value is valid")
			Expect(checkToken(initialSecretToken)).To(Succeed())

			By("deleting the Token resource")
			Expect(clientCtx.deleteToken(testToken3, targetNamespace)).To(Succeed())
		})
	})

	Context("Metrics", func() {
		var metricsBody string

		BeforeAll(func() {
			By("finding the manager pod")
			podName := clientCtx.getManagerPodName(operatorNamespace, map[string]string{
				"app.kubernetes.io/name":     "github-token-manager",
				"app.kubernetes.io/instance": "github-token-manager",
			})

			By("scraping /metrics via port-forward")
			metricsBody = scrapeMetrics(operatorNamespace, podName, 8080)
		})

		It("serves /metrics with OTEL-identified, scope-pruned payload", func() {
			By("endpoint-up smoke check")
			Expect(metricsBody).To(ContainSubstring("go_goroutines"))

			By("target_info carries OTEL Resource identity")
			Expect(metricsBody).To(
				MatchRegexp(`target_info\{[^}]*service_name="github-token-manager"[^}]*\}`),
				"target_info missing service_name=github-token-manager",
			)
			Expect(metricsBody).To(
				MatchRegexp(`target_info\{[^}]*service_version="[^"]+"[^}]*\}`),
				"target_info missing non-empty service_version",
			)
			Expect(metricsBody).To(
				MatchRegexp(`target_info\{[^}]*service_instance_id="[^"]+"[^}]*\}`),
				"target_info missing non-empty service_instance_id",
			)

			By("otel_scope_* labels have been stripped")
			Expect(metricsBody).NotTo(ContainSubstring("otel_scope_"))

			By("no metric family starts with the retired gtm_ prefix")
			Expect(metricsBody).NotTo(
				MatchRegexp(`(?m)^gtm_`),
				"found residual gtm_-prefixed metric family",
			)

			By("controller-runtime defaults carry the expected controller labels")
			Expect(metricsBody).To(
				MatchRegexp(`controller_runtime_reconcile_total\{[^}]*controller="github-token"[^}]*\}`),
			)
			Expect(metricsBody).To(
				MatchRegexp(`controller_runtime_reconcile_total\{[^}]*controller="github-clustertoken"[^}]*\}`),
			)
			Expect(metricsBody).To(
				MatchRegexp(`workqueue_adds_total\{[^}]*controller="github-token"[^}]*\}`),
			)
			Expect(metricsBody).To(
				MatchRegexp(`workqueue_adds_total\{[^}]*controller="github-clustertoken"[^}]*\}`),
			)
		})

		It("exposes custom Prometheus metric TYPE lines after reconciliation", func() {
			if !hasAppCredentials {
				Skip("skipping custom metric checks - no valid GitHub App configuration provided")
			}

			// Only the happy-path instruments are asserted. The OTEL Prometheus
			// exporter only emits a `# TYPE` line once an instrument has recorded
			// at least one data point, so error-only counters (token_reconcile_errors_total,
			// config_errors_total) are intentionally excluded from this list.
			expectedTypes := []string{
				"token_refresh_total",
				"token_refresh_duration_seconds",
				"github_api_call_duration_seconds",
				"github_api_requests_total",
				"token_expiry_timestamp_seconds",
				"tokens_active",
				"kubernetes_secret_operations_total",
			}
			for _, metric := range expectedTypes {
				Expect(metricsBody).To(
					ContainSubstring(fmt.Sprintf("# TYPE %s ", metric)),
					"missing TYPE line for %s", metric,
				)
			}
		})

		It("reports non-zero success counters after reconciliation", func() {
			if !hasAppCredentials {
				Skip("skipping metrics counter checks - no valid GitHub App configuration provided")
			}

			By("checking token_refresh_total{controller=github-token,result=success} >= 1")
			Expect(metricsBody).To(
				MatchRegexp(`token_refresh_total\{[^}]*controller="github-token"[^}]*result="success"[^}]*\}\s+[1-9]`),
			)

			By("checking kubernetes_secret_operations_total{controller=github-token,operation=create,result=success} >= 1")
			Expect(metricsBody).To(
				MatchRegexp(`kubernetes_secret_operations_total\{[^}]*controller="github-token"[^}]*operation="create"[^}]*result="success"[^}]*\}\s+[1-9]`),
			)

			By("checking github_api_call_duration_seconds_count{controller=github-token,result=success} >= 1")
			Expect(metricsBody).To(
				MatchRegexp(`github_api_call_duration_seconds_count\{[^}]*controller="github-token"[^}]*result="success"[^}]*\}\s+[1-9]`),
			)

			By("checking github_api_requests_total{controller=github-token,result=success} >= 1")
			Expect(metricsBody).To(
				MatchRegexp(`github_api_requests_total\{[^}]*controller="github-token"[^}]*result="success"[^}]*\}\s+[1-9]`),
			)

			By("checking tokens_active{controller=github-token} >= 1")
			Expect(metricsBody).To(
				MatchRegexp(`tokens_active\{[^}]*controller="github-token"[^}]*\}\s+[1-9]`),
			)
		})
	})

	Context("Helm Chart", func() {
		It("should uninstall without error", func() {
			if e2ePreserveState() {
				Skip("E2E_PRESERVE is set; leaving Helm release in place for inspection")
			}
			By("uninstalling Helm chart")
			cmd := exec.Command("helm", "uninstall", "github-token-manager", "--namespace", operatorNamespace)
			_, err := runCommand(cmd)
			Expect(err).NotTo(HaveOccurred())

		})
	})

	AfterAll(func() {
		if e2ePreserveState() {
			fmt.Fprintln(GinkgoWriter, "E2E_PRESERVE is set; skipping namespace teardown")
			return
		}
		Expect(clientCtx.deleteNamespace(operatorNamespace)).To(Succeed())
		Expect(clientCtx.deleteNamespace(targetNamespace)).To(Succeed())
	})
})
