package metrics

import (
	"context"
	"testing"
)

// TestSetup guards the resource.Merge path in Setup: a schema-URL conflict between the
// custom resource and resource.Default() previously made this return an error at
// startup. Setup registers the exporter against the global crmetrics.Registry, so it
// must only be invoked once per test binary.
func TestSetup(t *testing.T) {
	rec, err := Setup("v-test")
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if rec == nil {
		t.Fatal("Setup returned nil recorder")
	}
	// t.Context() is already cancelled when cleanups run; derive an
	// uncancelled context for the final flush.
	t.Cleanup(func() { _ = rec.Shutdown(context.WithoutCancel(t.Context())) })
}
