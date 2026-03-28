package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunVerify_CleansTransientArtifactsButKeepsCache(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-cli")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "sample-cli"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal", "cli"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "library", "sample-cli"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".cache", "go-build"), 0o755))

	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/sample-cli\n\ngo 1.26.1\n")
	writeTestFile(t, filepath.Join(dir, "cmd", "sample-cli", "main.go"), `package main
func main() {}
`)
	writeTestFile(t, filepath.Join(dir, "internal", "cli", "root.go"), `package cli
func initRoot() {
	rootCmd.AddCommand(newUsersListCmd())
}
`)
	writeTestFile(t, filepath.Join(dir, "cmd", "library", "sample-cli", "main.go"), "package recursive\n")
	writeTestFile(t, filepath.Join(dir, ".DS_Store"), "finder")
	writeTestFile(t, filepath.Join(dir, ".cache", "go-build", "index"), "cache")

	report, err := RunVerify(VerifyConfig{Dir: dir})
	require.NoError(t, err)
	assert.Equal(t, "PASS", report.Verdict)
	assert.FileExists(t, report.Binary)

	assert.FileExists(t, filepath.Join(dir, "sample-cli"))
	assert.NoDirExists(t, filepath.Join(dir, "cmd", "library"))
	assert.NoFileExists(t, filepath.Join(dir, ".DS_Store"))
	assert.DirExists(t, filepath.Join(dir, ".cache"))
}

func TestRunVerify_KeepsExistingBinaryWhenRebuildFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-cli")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "sample-cli"), 0o755))

	existingBinary := filepath.Join(dir, "sample-cli")
	require.NoError(t, os.WriteFile(existingBinary, []byte("previous-build"), 0o755))
	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/sample-cli\n\ngo 1.26.1\n")
	writeTestFile(t, filepath.Join(dir, "cmd", "sample-cli", "main.go"), `package main
func main() {
	this will not compile
}
`)

	report, err := RunVerify(VerifyConfig{Dir: dir})
	require.Error(t, err)
	assert.Nil(t, report)
	assert.FileExists(t, existingBinary)
}

func TestBuildCLI_UsesCanonicalCommandDirForClaimedOutput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-pp-cli-2")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "sample-pp-cli"), 0o755))

	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/sample-pp-cli\n\ngo 1.26.1\n")
	writeTestFile(t, filepath.Join(dir, "cmd", "sample-pp-cli", "main.go"), `package main
func main() {}
`)

	binaryPath, err := buildCLI(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "sample-pp-cli-2"), binaryPath)
	assert.FileExists(t, binaryPath)
}
