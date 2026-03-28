package naming

import "testing"

func TestTrimCLISuffix(t *testing.T) {
	tests := map[string]string{
		"notion-pp-cli":   "notion",
		"notion-pp-cli-2": "notion",
		"legacy-cli":      "legacy",
		"legacy-cli-4":    "legacy",
		"plain":           "plain",
	}

	for input, want := range tests {
		if got := TrimCLISuffix(input); got != want {
			t.Fatalf("TrimCLISuffix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsCLIDirName(t *testing.T) {
	if !IsCLIDirName("stripe-pp-cli-3") {
		t.Fatal("expected suffixed pp-cli directory to be recognized")
	}
	if IsCLIDirName("stripe-pp-mcp") {
		t.Fatal("mcp directories must not be treated as cli directories")
	}
}
