package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AugmentREADME reads evidence files from a dogfood run and injects
// real command output into the generated README. It looks for marker
// comments like <!-- HELP_OUTPUT --> and replaces them with captured output.
//
// If no markers exist, it appends a "Real Usage Examples" section.
func AugmentREADME(readmePath, evidenceDir string) error {
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("reading README: %w", err)
	}

	readme := string(content)
	modified := false

	// Try marker-based replacement first
	markers := map[string]string{
		"<!-- HELP_OUTPUT -->":    filepath.Join(evidenceDir, "tier1-help.txt"),
		"<!-- VERSION_OUTPUT -->": filepath.Join(evidenceDir, "tier1-version.txt"),
		"<!-- DOCTOR_OUTPUT -->":  filepath.Join(evidenceDir, "tier1-doctor.txt"),
	}

	for marker, evidenceFile := range markers {
		if strings.Contains(readme, marker) {
			output, err := os.ReadFile(evidenceFile)
			if err != nil {
				continue
			}
			replacement := fmt.Sprintf("```\n%s```", string(output))
			readme = strings.Replace(readme, marker, replacement, 1)
			modified = true
		}
	}

	// If no markers found, append a section with real output
	if !modified {
		var section strings.Builder
		section.WriteString("\n## Real Usage Examples\n\n")
		section.WriteString("*Captured from automated dogfood testing.*\n\n")

		helpOutput, err := os.ReadFile(filepath.Join(evidenceDir, "tier1-help.txt"))
		if err == nil && len(helpOutput) > 0 {
			section.WriteString("### Help\n\n```\n")
			section.Write(helpOutput)
			section.WriteString("```\n\n")
			modified = true
		}

		doctorOutput, err := os.ReadFile(filepath.Join(evidenceDir, "tier1-doctor.txt"))
		if err == nil && len(doctorOutput) > 0 {
			section.WriteString("### Health Check\n\n```\n")
			section.Write(doctorOutput)
			section.WriteString("```\n\n")
			modified = true
		}

		// Add per-resource help if available
		resDir := filepath.Join(evidenceDir, "tier1-resources")
		entries, _ := os.ReadDir(resDir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), "-help.txt") {
				continue
			}
			resName := strings.TrimSuffix(e.Name(), "-help.txt")
			output, err := os.ReadFile(filepath.Join(resDir, e.Name()))
			if err == nil && len(output) > 0 {
				fmt.Fprintf(&section, "### %s\n\n```\n", resName)
				section.Write(output)
				section.WriteString("```\n\n")
			}
		}

		// Add tier 2 real API output if available
		tier2Dir := filepath.Join(evidenceDir, "tier2-reads")
		t2entries, _ := os.ReadDir(tier2Dir)
		if len(t2entries) > 0 {
			section.WriteString("### Real API Output\n\n")
			for _, e := range t2entries {
				if e.IsDir() {
					continue
				}
				output, err := os.ReadFile(filepath.Join(tier2Dir, e.Name()))
				if err == nil && len(output) > 0 {
					cmdName := strings.TrimSuffix(e.Name(), ".txt")
					fmt.Fprintf(&section, "#### %s\n\n```json\n", cmdName)
					// Truncate very long output
					s := string(output)
					if len(s) > 2000 {
						s = s[:2000] + "\n... (truncated)\n"
					}
					section.WriteString(s)
					section.WriteString("```\n\n")
				}
			}
		}

		if modified {
			readme += section.String()
		}
	}

	if !modified {
		return nil // no evidence to inject
	}

	// Run anti-AI text filter on the README
	warnings := CheckText(readme)
	if len(warnings) > 0 {
		fmt.Fprint(os.Stderr, FormatWarnings(warnings))
	}

	return os.WriteFile(readmePath, []byte(readme), 0o644)
}
