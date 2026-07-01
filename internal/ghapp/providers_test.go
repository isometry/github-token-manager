package ghapp

import (
	"slices"
	"testing"

	"github.com/isometry/ghait/v84/provider"
)

// TestDefaultBuildRegistersAllProviders guards against the default (tagless)
// build silently dropping cloud KMS providers. Published images are built
// with no Go build tags, so every provider that should work out of the box
// must self-register via an init() pulled in by an underscore import
// somewhere in this package's dependency graph.
func TestDefaultBuildRegistersAllProviders(t *testing.T) {
	want := []string{"aws", "azure", "gcp", "vault", "file"}

	registered := provider.Registered()

	for _, name := range want {
		if !slices.Contains(registered, name) {
			t.Errorf("provider %q not registered in default build; registered: %v", name, registered)
		}
	}
}
