package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoBuildCacheDirIsShared(t *testing.T) {
	t.Setenv("GOCACHE", "")

	// Two different project directories should get the same cache dir.
	// This is critical for CI performance — shared cache avoids each
	// parallel test recompiling the Go standard library from scratch.
	dir1, err := goBuildCacheDir("/tmp/project-a")
	require.NoError(t, err)

	dir2, err := goBuildCacheDir("/tmp/project-b")
	require.NoError(t, err)

	assert.Equal(t, dir1, dir2, "different projects should share the same build cache")
}

func TestGoBuildCacheDirPath(t *testing.T) {
	t.Setenv("GOCACHE", "")

	dir, err := goBuildCacheDir("/tmp/any-project")
	require.NoError(t, err)

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	expected := filepath.Join(home, ".cache", "printing-press", "go-build")
	assert.Equal(t, expected, dir)
}

func TestGoBuildCacheDirHonorsExplicitGOCACHE(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "go-build")
	t.Setenv("GOCACHE", cacheDir)

	dir, err := goBuildCacheDir("/tmp/any-project")
	require.NoError(t, err)

	assert.Equal(t, cacheDir, dir)
	assert.DirExists(t, cacheDir)
}
