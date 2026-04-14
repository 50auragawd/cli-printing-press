package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeMCPTools writes a generated tools.go with the provided RegisterTools
// body and returns the CLI dir. The body is inserted into a minimal file
// stub that matches what the MCP template produces.
func writeMCPTools(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	mcpDir := filepath.Join(dir, "internal", "mcp")
	require.NoError(t, os.MkdirAll(mcpDir, 0o755))
	content := `package mcp

import mcplib "github.com/mark3labs/mcp-go/mcp"

func RegisterTools(s *server.MCPServer) {
` + body + `
}
`
	require.NoError(t, os.WriteFile(filepath.Join(mcpDir, "tools.go"), []byte(content), 0o644))
	return dir
}

func TestEstimateMCPTokens_NoMCPSurface(t *testing.T) {
	dir := t.TempDir()
	est := estimateMCPTokens(dir)
	assert.Equal(t, 0, est.TotalChars)
	assert.Equal(t, 0, est.ToolCount)
}

func TestEstimateMCPTokens_StubWithoutRegisterTools(t *testing.T) {
	dir := t.TempDir()
	mcpDir := filepath.Join(dir, "internal", "mcp")
	require.NoError(t, os.MkdirAll(mcpDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mcpDir, "tools.go"),
		[]byte(`package mcp`), 0o644))

	est := estimateMCPTokens(dir)
	assert.Equal(t, 0, est.ToolCount, "stub without RegisterTools treated as no MCP surface")
}

func TestEstimateMCPTokens_SingleSmallTool(t *testing.T) {
	dir := writeMCPTools(t, `
	s.AddTool(
		mcplib.NewTool("get_user",
			mcplib.WithDescription("Retrieve a user by ID."),
			mcplib.WithString("id", mcplib.Description("User ID"))),
		handleGetUser)
`)
	est := estimateMCPTokens(dir)
	require.Equal(t, 1, est.ToolCount)
	assert.Equal(t, "get_user", est.PerTool[0].Name)
	assert.Greater(t, est.TotalChars, 0)
	assert.Greater(t, est.TotalTokens, 0)
	// small tool should be well under 80 tokens
	assert.Less(t, est.PerTool[0].Tokens, 80)
}

func TestEstimateMCPTokens_MultipleToolsTopHeaviest(t *testing.T) {
	// Three tools with increasing description length so TopHeaviest
	// ordering is deterministic.
	dir := writeMCPTools(t, `
	s.AddTool(mcplib.NewTool("small", mcplib.WithDescription("tiny")), nil)
	s.AddTool(mcplib.NewTool("medium", mcplib.WithDescription("a medium description here")), nil)
	s.AddTool(mcplib.NewTool("huge", mcplib.WithDescription("`+strings.Repeat("x", 500)+`")), nil)
`)
	est := estimateMCPTokens(dir)
	require.Equal(t, 3, est.ToolCount)
	require.Len(t, est.TopHeaviest, 3)
	assert.Equal(t, "huge", est.TopHeaviest[0].Name, "largest tool listed first in TopHeaviest")
	assert.Greater(t, est.TopHeaviest[0].Chars, est.TopHeaviest[1].Chars)
	assert.Greater(t, est.TopHeaviest[1].Chars, est.TopHeaviest[2].Chars)
}

func TestScoreMCPTokenEfficiency_FullMarksForLeanSurface(t *testing.T) {
	dir := writeMCPTools(t, `
	s.AddTool(mcplib.NewTool("get", mcplib.WithDescription("get item")), nil)
	s.AddTool(mcplib.NewTool("list", mcplib.WithDescription("list items")), nil)
`)
	score, scored := scoreMCPTokenEfficiency(dir)
	assert.True(t, scored)
	assert.Equal(t, 10, score, "small per-tool size earns full marks")
}

func TestScoreMCPTokenEfficiency_NotScoredWhenNoMCP(t *testing.T) {
	dir := t.TempDir()
	score, scored := scoreMCPTokenEfficiency(dir)
	assert.False(t, scored, "missing MCP surface must be unscored, not zero-scored")
	assert.Equal(t, 0, score)
}

func TestScoreMCPTokenEfficiency_PenalizesBloatedSurface(t *testing.T) {
	// One tool with a ~1600-char description → ~400 tokens, above the
	// 320-token ceiling → 0 score.
	dir := writeMCPTools(t, `
	s.AddTool(mcplib.NewTool("bloated", mcplib.WithDescription("`+strings.Repeat("x", 1600)+`")), nil)
`)
	score, scored := scoreMCPTokenEfficiency(dir)
	assert.True(t, scored)
	assert.Equal(t, 0, score, "oversized per-tool description scores zero")
}

func TestScoreMCPTokenEfficiency_PartialCreditMedium(t *testing.T) {
	// One tool at ~500 chars → ~125 tokens → partial credit (7)
	dir := writeMCPTools(t, `
	s.AddTool(mcplib.NewTool("medium", mcplib.WithDescription("`+strings.Repeat("x", 500)+`")), nil)
`)
	score, scored := scoreMCPTokenEfficiency(dir)
	assert.True(t, scored)
	assert.Equal(t, 7, score)
}
