package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wingmodels/internal/models"
)

var testServer *Server

func setup(t *testing.T) *Server {
	t.Helper()
	if testServer != nil {
		return testServer
	}
	s, err := NewServer("../../build/api.json")
	if err != nil {
		t.Fatalf("NewServer failed: %v (have you run 'go run ./cmd/wingmodels compile'?)", err)
	}
	testServer = s
	return s
}

func TestHealthz(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/healthz")

	assertStatus(t, rr, http.StatusOK)

	var body map[string]string
	decodeJSON(t, rr, &body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}

func TestGetLabs(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/labs")

	assertStatus(t, rr, http.StatusOK)

	var labs []models.Lab
	decodeJSON(t, rr, &labs)
	if len(labs) == 0 {
		t.Error("expected at least one lab, got empty array")
	}
}

func TestGetLabByID(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/labs/openai")

	assertStatus(t, rr, http.StatusOK)

	var lab models.Lab
	decodeJSON(t, rr, &lab)
	if lab.ID != "openai" {
		t.Errorf("lab.ID = %q, want %q", lab.ID, "openai")
	}
	if lab.DisplayName != "OpenAI" {
		t.Errorf("lab.DisplayName = %q, want %q", lab.DisplayName, "OpenAI")
	}
}

func TestGetLabByID_NotFound(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/labs/nonexistent")

	assertStatus(t, rr, http.StatusNotFound)
}

func TestGetProviderModels(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/provider-models")

	assertStatus(t, rr, http.StatusOK)

	var pms []models.ProviderModel
	decodeJSON(t, rr, &pms)
	if len(pms) == 0 {
		t.Error("expected provider models, got empty array")
	}
}

func TestGetProviderModelsFilter(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/provider-models?provider=openai")

	assertStatus(t, rr, http.StatusOK)

	var pms []models.ProviderModel
	decodeJSON(t, rr, &pms)
	if len(pms) == 0 {
		t.Fatal("expected OpenAI provider models, got empty array")
	}
	for _, pm := range pms {
		if pm.ID[:6] != "openai" {
			t.Errorf("expected all IDs to start with 'openai', got %q", pm.ID)
		}
	}
}

func TestGetProviderModelsSearch(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/provider-models?q=claude")

	assertStatus(t, rr, http.StatusOK)

	var pms []models.ProviderModel
	decodeJSON(t, rr, &pms)
	if len(pms) == 0 {
		t.Fatal("expected Claude models from search, got empty array")
	}
	for _, pm := range pms {
		idLower := toLower(pm.ID)
		nameLower := toLower(pm.DisplayName)
		if !contains(idLower, "claude") && !contains(nameLower, "claude") {
			t.Errorf("result %q does not match search 'claude'", pm.ID)
		}
	}
}

func TestGetProviderModelByID(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/provider-models/openai/gpt-4o")

	assertStatus(t, rr, http.StatusOK)

	var pm models.ProviderModel
	decodeJSON(t, rr, &pm)
	if pm.ID != "openai/gpt-4o" {
		t.Errorf("pm.ID = %q, want %q", pm.ID, "openai/gpt-4o")
	}
}

func TestGetSnapshot(t *testing.T) {
	s := setup(t)
	rr := doGet(t, s, "/v1/snapshot")

	assertStatus(t, rr, http.StatusOK)

	var snap models.Snapshot
	decodeJSON(t, rr, &snap)
	if snap.Version != "v1" {
		t.Errorf("snapshot.Version = %q, want %q", snap.Version, "v1")
	}
	if len(snap.Labs) == 0 {
		t.Error("snapshot has no labs")
	}
}

// --- helpers ---

func doGet(t *testing.T, s *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)
	return rr
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Errorf("status = %d, want %d; body: %s", rr.Code, want, rr.Body.String())
	}
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("decoding JSON response: %v; body: %s", err, rr.Body.String())
	}
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
