package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSuccess(t *testing.T) {
	result, err := Load("../../data")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	assertCount(t, "Labs", len(result.Labs), 3)
	assertCount(t, "LabModels", len(result.LabModels), 27)
	assertCount(t, "Providers", len(result.Providers), 4)
	assertCount(t, "ProviderModels", len(result.ProviderModels), 37)
	assertCount(t, "InterfaceProfiles", len(result.InterfaceProfiles), 4)
	assertCount(t, "CapabilityDefs", len(result.CapabilityDefs), 15)
	assertCount(t, "ParameterDefs", len(result.ParameterDefs), 10)
}

func TestLoadDateHandling(t *testing.T) {
	// Create a temp data dir with a lab model that has date fields.
	dataDir := t.TempDir()
	labDir := filepath.Join(dataDir, "labs", "datelab")
	modelsDir := filepath.Join(labDir, "models")
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(labDir, "lab.toml"), `
id = "datelab"
display_name = "Date Lab"
`)

	writeFile(t, filepath.Join(modelsDir, "m1.toml"), `
id = "m1"
lab_id = "datelab"
display_name = "Model One"
release_date = 2024-05-13
knowledge_cutoff = 2023-09
`)

	result, err := Load(dataDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(result.LabModels) != 1 {
		t.Fatalf("expected 1 lab model, got %d", len(result.LabModels))
	}

	m := result.LabModels[0]

	// Full TOML date -> YYYY-MM-DD string.
	if m.ReleaseDate != "2024-05-13" {
		t.Errorf("release_date = %q, want %q", m.ReleaseDate, "2024-05-13")
	}

	// Bare YYYY-MM -> quoted string (no date conversion needed).
	if !strings.HasPrefix(m.KnowledgeCutoff, "2023-09") {
		t.Errorf("knowledge_cutoff = %q, want prefix %q", m.KnowledgeCutoff, "2023-09")
	}
}

// --- helpers ---

func assertCount(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s count = %d, want %d", name, got, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile(%s): %v", path, err)
	}
}
