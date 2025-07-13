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

package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v73/github"
	"github.com/onsi/ginkgo/v2"
	"golang.org/x/oauth2"
)

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "running: %s\n", command)
	output, err := cmd.Output()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func LoadImageToKindCluster(name, cluster string) error {
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// GenerateRSAPrivateKey generates a new RSA private key and returns it as a PEM-encoded string
func GenerateRSAPrivateKey() (string, error) {
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

// ValidateGitHubToken validates that a GitHub token can read content from the specified repository
// by making a simple API call to get the repository's README.
// Returns nil if the token is valid, error otherwise.
func ValidateGitHubToken(repository, token string) error {
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

// NewGitHubTokenValidator creates a new GitHub token validator for the specified repository.
// Returns a function that can be used to validate tokens.
func NewGitHubTokenValidator(repository string) func(string) error {
	return func(token string) error {
		return ValidateGitHubToken(repository, token)
	}
}
