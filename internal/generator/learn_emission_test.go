package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateLearnPackageEmitsAllFiles verifies the learn package
// emission lands every expected file at the right path under
// internal/learn/. Mirrors the share-emission test pattern.
func TestGenerateLearnPackageEmitsAllFiles(t *testing.T) {
	t.Parallel()

	apiSpec := minimalSpec("learn-emit")
	apiSpec.Learn.Enabled = true
	outputDir := filepath.Join(t.TempDir(), "learn-emit-pp-cli")
	gen := New(apiSpec, outputDir)
	gen.VisionSet = VisionTemplateSet{Store: true}
	require.NoError(t, gen.Generate())

	wantFiles := []string{
		"internal/learn/doc.go",
		"internal/learn/normalize.go",
		"internal/learn/normalize_test.go",
		"internal/learn/match.go",
		"internal/learn/match_test.go",
		"internal/learn/recall.go",
		"internal/learn/recall_test.go",
		"internal/learn/teach.go",
		"internal/learn/teach_test.go",
		"internal/learn/teach_log.go",
		"internal/learn/teach_log_test.go",
		"internal/learn/preseed.go",
		"internal/learn/preseed_test.go",
		"internal/learn/entities/config.go",
		"internal/learn/entities/config_test.go",
		"internal/learn/entities/extract.go",
		"internal/learn/entities/extract_test.go",
		"internal/learn/lookups/store.go",
		"internal/learn/lookups/store_test.go",
		"internal/learn/lookups/seeds.go",
		"internal/learn/lookups/seeds_test.go",
		"internal/learn/patterns/doc.go",
		"internal/learn/patterns/store.go",
		"internal/learn/patterns/store_test.go",
		"internal/learn/patterns/extract.go",
		"internal/learn/patterns/extract_test.go",
		"internal/learn/patterns/apply.go",
		"internal/learn/patterns/apply_test.go",
	}
	for _, rel := range wantFiles {
		_, err := os.Stat(filepath.Join(outputDir, rel))
		require.NoError(t, err, "expected emitted file %s", rel)
	}
}

// TestGenerateLearnPackageGatedOff verifies the learn package files do
// NOT emit when Learn.Enabled is false; pairs with the store-side gate
// already covered in learn_store_test.go.
func TestGenerateLearnPackageGatedOff(t *testing.T) {
	t.Parallel()

	apiSpec := minimalSpec("learn-gated")
	apiSpec.Learn.Enabled = false
	outputDir := filepath.Join(t.TempDir(), "learn-gated-pp-cli")
	gen := New(apiSpec, outputDir)
	gen.VisionSet = VisionTemplateSet{Store: true}
	require.NoError(t, gen.Generate())

	_, err := os.Stat(filepath.Join(outputDir, "internal", "learn"))
	require.True(t, os.IsNotExist(err), "internal/learn must not exist when Learn.Enabled=false")
}

// TestGenerateLearnPackageCompilesAndTests drives the emitted learn
// package through `go test ./internal/learn/...` to catch any template
// issue that produces shape-valid but uncompilable Go, plus any
// behavior regression in the ported logic.
func TestGenerateLearnPackageCompilesAndTests(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("compile-and-test of emitted learn package skipped in -short mode")
	}

	apiSpec := minimalSpec("learn-built")
	apiSpec.Learn.Enabled = true
	outputDir := filepath.Join(t.TempDir(), "learn-built-pp-cli")
	gen := New(apiSpec, outputDir)
	gen.VisionSet = VisionTemplateSet{Store: true}
	require.NoError(t, gen.Generate())

	runGoCommand(t, outputDir, "test", "./internal/learn/...")
}
