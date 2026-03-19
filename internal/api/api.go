package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"wingmodels/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

type Server struct {
	snapshot *models.Snapshot

	// In-memory indexes
	labsByID              map[string]*models.Lab
	labModelsByID         map[string]*models.LabModel
	providersByID         map[string]*models.Provider
	providerModelsByID    map[string]*models.ProviderModel
	interfaceProfilesByID map[string]*models.InterfaceProfile
	capabilityDefsByID    map[string]*models.CapabilityDef
	parameterDefsByID     map[string]*models.ParameterDef
}

func NewServer(snapshotPath string) (*Server, error) {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var snap models.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}

	s := &Server{
		snapshot:              &snap,
		labsByID:              make(map[string]*models.Lab, len(snap.Labs)),
		labModelsByID:         make(map[string]*models.LabModel, len(snap.LabModels)),
		providersByID:         make(map[string]*models.Provider, len(snap.Providers)),
		providerModelsByID:    make(map[string]*models.ProviderModel, len(snap.ProviderModels)),
		interfaceProfilesByID: make(map[string]*models.InterfaceProfile, len(snap.InterfaceProfiles)),
		capabilityDefsByID:    make(map[string]*models.CapabilityDef, len(snap.CapabilityDefs)),
		parameterDefsByID:     make(map[string]*models.ParameterDef, len(snap.ParameterDefs)),
	}

	for i := range snap.Labs {
		s.labsByID[snap.Labs[i].ID] = &snap.Labs[i]
	}
	for i := range snap.LabModels {
		s.labModelsByID[snap.LabModels[i].ID] = &snap.LabModels[i]
	}
	for i := range snap.Providers {
		s.providersByID[snap.Providers[i].ID] = &snap.Providers[i]
	}
	for i := range snap.ProviderModels {
		s.providerModelsByID[snap.ProviderModels[i].ID] = &snap.ProviderModels[i]
	}
	for i := range snap.InterfaceProfiles {
		s.interfaceProfilesByID[snap.InterfaceProfiles[i].ID] = &snap.InterfaceProfiles[i]
	}
	for i := range snap.CapabilityDefs {
		s.capabilityDefsByID[snap.CapabilityDefs[i].ID] = &snap.CapabilityDefs[i]
	}
	for i := range snap.ParameterDefs {
		s.parameterDefsByID[snap.ParameterDefs[i].ID] = &snap.ParameterDefs[i]
	}

	return s, nil
}

func (s *Server) Router() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "application/json"))
	r.Use(middleware.Timeout(8 * time.Second))
	r.Use(httprate.LimitByRealIP(100, time.Minute))

	r.Get("/healthz", s.handleHealthz)

	r.Route("/v1", func(r chi.Router) {
		r.Use(jsonContentType)

		r.Get("/labs", s.handleListLabs)
		r.Get("/labs/{id}", s.handleGetLab)

		r.Get("/models", s.handleListModels)
		r.Get("/models/{id}", s.handleGetModel)

		r.Get("/providers", s.handleListProviders)
		r.Get("/providers/{id}", s.handleGetProvider)

		r.Get("/provider-models", s.handleListProviderModels)
		r.Get("/provider-models/{provider_id}/{model_id}", s.handleGetProviderModel)

		r.Get("/interface-profiles", s.handleListInterfaceProfiles)
		r.Get("/interface-profiles/{id}", s.handleGetInterfaceProfile)

		r.Get("/capabilities", s.handleListCapabilities)
		r.Get("/parameters", s.handleListParameters)

		r.Get("/snapshot", s.handleGetSnapshot)
	})

	return r
}

// jsonContentType sets Content-Type: application/json for all responses.
func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// --- Handlers ---

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListLabs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot.Labs)
}

func (s *Server) handleGetLab(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	lab, ok := s.labsByID[id]
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, lab)
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot.LabModels)
}

func (s *Server) handleGetModel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, ok := s.labModelsByID[id]
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot.Providers)
}

func (s *Server) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, ok := s.providersByID[id]
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleListProviderModels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filterProvider := q.Get("provider")
	filterModel := q.Get("model")
	filterInterface := q.Get("interface")
	filterCapability := q.Get("capability")
	filterInputModality := q.Get("input_modality")
	filterOutputModality := q.Get("output_modality")
	filterSupportsParam := q.Get("supports_parameter")
	filterSearch := strings.ToLower(q.Get("q"))

	hasFilters := filterProvider != "" || filterModel != "" || filterInterface != "" ||
		filterCapability != "" || filterInputModality != "" || filterOutputModality != "" ||
		filterSupportsParam != "" || filterSearch != ""

	if !hasFilters {
		writeJSON(w, http.StatusOK, s.snapshot.ProviderModels)
		return
	}

	results := make([]models.ProviderModel, 0)
	for _, pm := range s.snapshot.ProviderModels {
		if !matchProviderModel(&pm, filterProvider, filterModel, filterInterface,
			filterCapability, filterInputModality, filterOutputModality,
			filterSupportsParam, filterSearch) {
			continue
		}
		results = append(results, pm)
	}

	writeJSON(w, http.StatusOK, results)
}

func matchProviderModel(pm *models.ProviderModel, provider, model, iface,
	capability, inputModality, outputModality, supportsParam, search string) bool {

	if provider != "" {
		// Provider ID is the part before the first slash
		parts := strings.SplitN(pm.ID, "/", 2)
		if len(parts) < 2 || parts[0] != provider {
			return false
		}
	}

	if model != "" {
		// Model ID is the part after the first slash
		parts := strings.SplitN(pm.ID, "/", 2)
		if len(parts) < 2 || parts[1] != model {
			return false
		}
	}

	if iface != "" {
		if !containsString(pm.InterfaceProfiles, iface) {
			return false
		}
	}

	if capability != "" {
		found := false
		for _, c := range pm.Capabilities {
			if c.ID == capability {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if inputModality != "" {
		if pm.Modalities == nil || !containsString(pm.Modalities.Input, inputModality) {
			return false
		}
	}

	if outputModality != "" {
		if pm.Modalities == nil || !containsString(pm.Modalities.Output, outputModality) {
			return false
		}
	}

	if supportsParam != "" {
		if !containsString(pm.SupportedParameters, supportsParam) {
			return false
		}
	}

	if search != "" {
		idMatch := strings.Contains(strings.ToLower(pm.ID), search)
		nameMatch := strings.Contains(strings.ToLower(pm.DisplayName), search)
		if !idMatch && !nameMatch {
			return false
		}
	}

	return true
}

func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func (s *Server) handleGetProviderModel(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider_id")
	modelID := chi.URLParam(r, "model_id")
	id := providerID + "/" + modelID

	pm, ok := s.providerModelsByID[id]
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, pm)
}

func (s *Server) handleListInterfaceProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot.InterfaceProfiles)
}

func (s *Server) handleGetInterfaceProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ip, ok := s.interfaceProfilesByID[id]
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, ip)
}

func (s *Server) handleListCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot.CapabilityDefs)
}

func (s *Server) handleListParameters(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot.ParameterDefs)
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot)
}
