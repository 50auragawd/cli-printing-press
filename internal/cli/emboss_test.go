package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mvanhorn/cli-printing-press/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteEmbossDeltaReportWritesToScopedProofsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PRINTING_PRESS_HOME", home)
	t.Setenv("PRINTING_PRESS_SCOPE", "test-scope")
	t.Setenv("PRINTING_PRESS_REPO_ROOT", filepath.Join(home, "repo"))

	dir := filepath.Join(t.TempDir(), "sample-pp-cli-2")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	state := pipeline.NewStateWithRun("sample", dir, "run-123", "test-scope")
	require.NoError(t, state.Save())
	before := EmbossSnapshot{ScorecardTotal: 60, ScorecardGrade: "B", VerifyPassRate: 80, VerifyPassed: 8, VerifyTotal: 10, CommandCount: 10}
	after := EmbossSnapshot{ScorecardTotal: 66, ScorecardGrade: "B", VerifyPassRate: 90, VerifyPassed: 9, VerifyTotal: 10, CommandCount: 11}
	delta := &EmbossDelta{ScorecardDelta: 6, VerifyDelta: 10, CommandDelta: 1}

	path, err := writeEmbossDeltaReport(dir, before, after, delta)
	require.NoError(t, err)

	assert.Contains(t, path, filepath.Join(home, ".runstate", "test-scope", "runs", "run-123", "proofs"))
	assert.Contains(t, filepath.Base(path), "sample-pp-cli-2")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "# Emboss Delta Report: sample-pp-cli-2"))
}

func TestResolveEmbossWorkspaceCreatesFreshRunForPublishedCLI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PRINTING_PRESS_HOME", home)
	t.Setenv("PRINTING_PRESS_SCOPE", "test-scope")
	t.Setenv("PRINTING_PRESS_REPO_ROOT", filepath.Join(home, "repo"))

	current := pipeline.NewStateWithRun("sample", filepath.Join(home, ".runstate", "test-scope", "runs", "run-current", "working", "sample-pp-cli"), "run-current", "test-scope")
	require.NoError(t, current.Save())

	publishedDir := filepath.Join(home, "library", "sample-pp-cli")
	require.NoError(t, os.MkdirAll(publishedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(publishedDir, "README.md"), []byte("published"), 0o644))

	existing := pipeline.NewStateWithRun("sample", filepath.Join(home, ".runstate", "test-scope", "runs", "run-old", "working", "sample-pp-cli"), "run-old", "test-scope")
	existing.PublishedDir = publishedDir
	require.NoError(t, os.MkdirAll(existing.EffectiveWorkingDir(), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(existing.EffectiveWorkingDir(), "README.md"), []byte("stale working copy"), 0o644))
	require.NoError(t, existing.SaveWithoutCurrentPointer())

	workingDir, baselinePath, state, err := resolveEmbossWorkspace(publishedDir)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.NotEqual(t, existing.RunID, state.RunID)
	assert.NotEqual(t, existing.EffectiveWorkingDir(), workingDir)
	assert.Equal(t, filepath.Join(state.ProofsDir(), ".emboss-baseline.json"), baselinePath)

	data, err := os.ReadFile(filepath.Join(workingDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "published", string(data))

	currentState, err := pipeline.LoadCurrentState("sample")
	require.NoError(t, err)
	assert.Equal(t, current.RunID, currentState.RunID)
}
