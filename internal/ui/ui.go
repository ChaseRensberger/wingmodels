package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"wingmodels/internal/models"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	snapshot *models.Snapshot
	tmpl     *template.Template

	// Indexes
	providersByID            map[string]*models.Provider
	labsByID                 map[string]*models.Lab
	labModelsByID            map[string]*models.LabModel
	providerModelsByProvider map[string][]*models.ProviderModel
	providerModelsByModel    map[string][]*models.ProviderModel
	labModelsByLab           map[string][]*models.LabModel
}

type layoutData struct {
	Title   string
	Active  string
	Content template.HTML
}

// Template data types

type modelsPageData struct {
	Providers      []models.Provider
	Capabilities   []models.CapabilityDef
	ProviderModels []*models.ProviderModel
}

type modelDetailData struct {
	Provider      *models.Provider
	ProviderModel *models.ProviderModel
	LabModel      *models.LabModel
	Lab           *models.Lab
}

type providerListItem struct {
	Provider   *models.Provider
	ModelCount int
}

type providersPageData struct {
	Providers []providerListItem
}

type providerDetailData struct {
	Provider *models.Provider
	Models   []*models.ProviderModel
}

func NewServer(snapshotPath string) (*Server, error) {
	snap, err := loadSnapshot(snapshotPath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		snapshot:                 snap,
		providersByID:            make(map[string]*models.Provider, len(snap.Providers)),
		labsByID:                 make(map[string]*models.Lab, len(snap.Labs)),
		labModelsByID:            make(map[string]*models.LabModel, len(snap.LabModels)),
		providerModelsByProvider: make(map[string][]*models.ProviderModel),
		providerModelsByModel:    make(map[string][]*models.ProviderModel),
		labModelsByLab:           make(map[string][]*models.LabModel),
	}

	for i := range snap.Labs {
		s.labsByID[snap.Labs[i].ID] = &snap.Labs[i]
	}
	for i := range snap.LabModels {
		s.labModelsByID[snap.LabModels[i].ID] = &snap.LabModels[i]
		s.labModelsByLab[snap.LabModels[i].LabID] = append(s.labModelsByLab[snap.LabModels[i].LabID], &snap.LabModels[i])
	}
	for i := range snap.Providers {
		s.providersByID[snap.Providers[i].ID] = &snap.Providers[i]
	}
	for i := range snap.ProviderModels {
		pm := &snap.ProviderModels[i]
		provID := providerFromID(pm.ID)
		modelID := modelFromID(pm.ID)
		s.providerModelsByProvider[provID] = append(s.providerModelsByProvider[provID], pm)
		s.providerModelsByModel[modelID] = append(s.providerModelsByModel[modelID], pm)
	}

	funcMap := template.FuncMap{
		"formatPrice":    formatPrice,
		"formatTokens":   formatTokens,
		"hasCapability":  hasCapability,
		"providerFromID": providerFromID,
		"modelFromID":    modelFromID,
		"join":           strings.Join,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	s.tmpl = tmpl

	return s, nil
}

func loadSnapshot(path string) (*models.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}
	var snap models.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}
	return &snap, nil
}

// RegisterRoutes adds UI routes to an existing chi router.
func (s *Server) RegisterRoutes(r chi.Router) {
	r.Get("/", s.handleModels)
	r.Get("/models/{providerID}/{modelID}", s.handleModelDetail)
	r.Get("/providers", s.handleProviders)
	r.Get("/providers/{id}", s.handleProviderDetail)
	r.Get("/search", s.handleSearch)
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, active, title, bodyTemplate string, data any) {
	var body bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&body, bodyTemplate, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHXRequest(r) {
		w.Write(body.Bytes())
		return
	}

	layout := layoutData{
		Title:   title,
		Active:  active,
		Content: template.HTML(body.String()),
	}
	if err := s.tmpl.ExecuteTemplate(w, "layout", layout); err != nil {
		http.Error(w, "layout error: "+err.Error(), http.StatusInternalServerError)
	}
}

func isHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// --- Handlers ---

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	allPMs := make([]*models.ProviderModel, len(s.snapshot.ProviderModels))
	for i := range s.snapshot.ProviderModels {
		allPMs[i] = &s.snapshot.ProviderModels[i]
	}

	data := modelsPageData{
		Providers:      s.snapshot.Providers,
		Capabilities:   s.snapshot.CapabilityDefs,
		ProviderModels: allPMs,
	}
	s.render(w, r, "models", "Models", "models", data)
}

func (s *Server) handleModelDetail(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "providerID")
	modelID := chi.URLParam(r, "modelID")
	pmID := providerID + "/" + modelID

	// Find the provider model
	var pm *models.ProviderModel
	for i := range s.snapshot.ProviderModels {
		if s.snapshot.ProviderModels[i].ID == pmID {
			pm = &s.snapshot.ProviderModels[i]
			break
		}
	}
	if pm == nil {
		http.NotFound(w, r)
		return
	}

	provider := s.providersByID[providerID]
	if provider == nil {
		http.NotFound(w, r)
		return
	}

	// Try to find linked lab model (model portion of ID)
	labModel := s.labModelsByID[modelID]
	var lab *models.Lab
	if labModel != nil {
		lab = s.labsByID[labModel.LabID]
	}

	data := modelDetailData{
		Provider:      provider,
		ProviderModel: pm,
		LabModel:      labModel,
		Lab:           lab,
	}
	s.render(w, r, "models", pm.DisplayName, "modelDetail", data)
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	items := make([]providerListItem, len(s.snapshot.Providers))
	for i := range s.snapshot.Providers {
		p := &s.snapshot.Providers[i]
		items[i] = providerListItem{
			Provider:   p,
			ModelCount: len(s.providerModelsByProvider[p.ID]),
		}
	}

	data := providersPageData{Providers: items}
	s.render(w, r, "providers", "Providers", "providers", data)
}

func (s *Server) handleProviderDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	provider := s.providersByID[id]
	if provider == nil {
		http.NotFound(w, r)
		return
	}

	data := providerDetailData{
		Provider: provider,
		Models:   s.providerModelsByProvider[id],
	}
	s.render(w, r, "providers", provider.DisplayName, "providerDetail", data)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	filterProvider := r.URL.Query().Get("provider")
	filterCapability := r.URL.Query().Get("capability")

	var results []*models.ProviderModel
	for i := range s.snapshot.ProviderModels {
		pm := &s.snapshot.ProviderModels[i]

		// Provider filter
		if filterProvider != "" && providerFromID(pm.ID) != filterProvider {
			continue
		}

		// Capability filter
		if filterCapability != "" && !hasCapability(pm, filterCapability) {
			continue
		}

		// Text search
		if q != "" {
			idMatch := strings.Contains(strings.ToLower(pm.ID), q)
			nameMatch := strings.Contains(strings.ToLower(pm.DisplayName), q)
			if !idMatch && !nameMatch {
				continue
			}
		}

		results = append(results, pm)
	}

	data := modelsPageData{
		ProviderModels: results,
	}

	var body bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&body, "modelList", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(body.Bytes())
}

// --- Template functions ---

func formatPrice(v float64) string {
	return fmt.Sprintf("$%.2f", v)
}

func formatTokens(n int) string {
	if n == 0 {
		return "-"
	}
	if n >= 1_000_000 && n%1_000_000 == 0 {
		return fmt.Sprintf("%dM", n/1_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 && n%1_000 == 0 {
		return fmt.Sprintf("%dK", n/1_000)
	}
	if n >= 1_000 {
		return addCommas(n)
	}
	return fmt.Sprintf("%d", n)
}

func addCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

func hasCapability(pm *models.ProviderModel, capID string) bool {
	for _, c := range pm.Capabilities {
		if c.ID == capID {
			return true
		}
	}
	return false
}

func providerFromID(id string) string {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) < 2 {
		return id
	}
	return parts[0]
}

func modelFromID(id string) string {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) < 2 {
		return id
	}
	return parts[1]
}
