package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunScorecardLoadsResearchFromSiblingResearchDir(t *testing.T) {
	outputDir := t.TempDir()
	runRoot := t.TempDir()
	researchDir := filepath.Join(runRoot, "research")
	proofsDir := filepath.Join(runRoot, "proofs")

	require.NoError(t, os.MkdirAll(researchDir, 0o755))

	research := &ResearchResult{
		APIName: "sample",
		Alternatives: []Alternative{
			{Name: "competitor/sample-cli"},
		},
	}
	data, err := json.MarshalIndent(research, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(researchDir, "research.json"), data, 0o644))

	scorecard, err := RunScorecard(outputDir, proofsDir, "", nil)
	require.NoError(t, err)

	assert.Len(t, scorecard.CompetitorScores, 1)
	assert.FileExists(t, filepath.Join(proofsDir, "scorecard.md"))
}
