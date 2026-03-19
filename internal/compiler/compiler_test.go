package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"wingmodels/internal/models"
)

func TestCompileSuccess(t *testing.T) {
	buildDir := t.TempDir()

	if err := Compile("../../data", buildDir); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Verify api.json exists and is valid JSON.
	apiPath := filepath.Join(buildDir, "api.json")
	data, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("reading api.json: %v", err)
	}

	var snap models.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("api.json is not valid JSON: %v", err)
	}

	// Verify expected entity counts.
	assertCount(t, "Labs", len(snap.Labs), 3)
	assertCount(t, "LabModels", len(snap.LabModels), 27)
	assertCount(t, "Providers", len(snap.Providers), 4)
	assertCount(t, "ProviderModels", len(snap.ProviderModels), 37)
	assertCount(t, "InterfaceProfiles", len(snap.InterfaceProfiles), 4)
	assertCount(t, "CapabilityDefs", len(snap.CapabilityDefs), 15)
	assertCount(t, "ParameterDefs", len(snap.ParameterDefs), 10)

	// Verify index files exist.
	indexes := []string{
		"indexes/provider-models-by-provider.json",
		"indexes/provider-models-by-model.json",
		"indexes/provider-models-by-capability.json",
	}
	for _, idx := range indexes {
		p := filepath.Join(buildDir, idx)
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("missing index file %s: %v", idx, err)
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("index %s is not valid JSON: %v", idx, err)
		}
	}
}

func TestCompileCrossReferenceValidation(t *testing.T) {
	// Create a minimal data directory with a provider model that references
	// a nonexistent lab model.
	dataDir := t.TempDir()

	// Create a lab.
	labDir := filepath.Join(dataDir, "labs", "testlab")
	mkdirAll(t, filepath.Join(labDir, "models"))
	writeFile(t, filepath.Join(labDir, "lab.toml"), `
id = "testlab"
display_name = "Test Lab"
`)
	// Create a lab model.
	writeFile(t, filepath.Join(labDir, "models", "m1.toml"), `
id = "m1"
lab_id = "testlab"
display_name = "Model One"
`)

	// Create a provider.
	provDir := filepath.Join(dataDir, "providers", "testprov")
	mkdirAll(t, filepath.Join(provDir, "models"))
	writeFile(t, filepath.Join(provDir, "provider.toml"), `
id = "testprov"
display_name = "Test Provider"
`)

	// Create a provider model referencing a nonexistent lab model "nonexistent".
	writeFile(t, filepath.Join(provDir, "models", "nonexistent.toml"), `
id = "testprov/nonexistent"
display_name = "Bad Model"
`)

	buildDir := t.TempDir()
	err := Compile(dataDir, buildDir)
	if err == nil {
		t.Fatal("expected error for bad cross-reference, got nil")
	}
	if got := err.Error(); !contains(got, "unknown lab model") {
		t.Errorf("expected error to mention 'unknown lab model', got: %s", got)
	}
}

func TestCompileRequiredFields(t *testing.T) {
	// Create a lab TOML missing required display_name.
	dataDir := t.TempDir()
	labDir := filepath.Join(dataDir, "labs", "badlab")
	mkdirAll(t, labDir)
	writeFile(t, filepath.Join(labDir, "lab.toml"), `
id = "badlab"
`)

	buildDir := t.TempDir()
	err := Compile(dataDir, buildDir)
	if err == nil {
		t.Fatal("expected validation error for missing display_name, got nil")
	}
	if got := err.Error(); !contains(got, "missing display_name") {
		t.Errorf("expected error to mention 'missing display_name', got: %s", got)
	}
}

// --- helpers ---

func assertCount(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s count = %d, want %d", name, got, want)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdirAll(%s): %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile(%s): %v", path, err)
	}
}
