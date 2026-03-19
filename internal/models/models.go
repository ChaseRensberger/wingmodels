package models

import "time"

type Source struct {
	Label       string `json:"label"`
	URL         string `json:"url"`
	RetrievedAt string `json:"retrieved_at,omitempty"`
}

type Lab struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Sources     []Source `json:"sources,omitempty"`
}

type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type LabModel struct {
	ID              string      `json:"id"`
	LabID           string      `json:"lab_id"`
	DisplayName     string      `json:"display_name"`
	Family          string      `json:"family,omitempty"`
	ReleaseDate     string      `json:"release_date,omitempty"`
	KnowledgeCutoff string      `json:"knowledge_cutoff,omitempty"`
	Modalities      *Modalities `json:"modalities,omitempty"`
	OpenWeights     bool        `json:"open_weights"`
	Sources         []Source    `json:"sources,omitempty"`
}

type ProviderAuth struct {
	EnvVars []string `json:"env_vars"`
}

type Provider struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"display_name"`
	BaseURLs    []string      `json:"base_urls,omitempty"`
	Auth        *ProviderAuth `json:"auth,omitempty"`
	Sources     []Source      `json:"sources,omitempty"`
}

type CapabilityInstance struct {
	ID      string                 `json:"id"`
	Version string                 `json:"version,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type Limits struct {
	ContextWindow   int `json:"context_window,omitempty"`
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`
	MaxInputTokens  int `json:"max_input_tokens,omitempty"`
}

type Pricing struct {
	InputPerMillion      float64 `json:"input_per_million,omitempty"`
	OutputPerMillion     float64 `json:"output_per_million,omitempty"`
	CacheReadPerMillion  float64 `json:"cache_read_per_million,omitempty"`
	CacheWritePerMillion float64 `json:"cache_write_per_million,omitempty"`
}

type ProviderModel struct {
	ID                  string                 `json:"id"`
	DisplayName         string                 `json:"display_name"`
	InterfaceProfiles   []string               `json:"interface_profiles,omitempty"`
	Capabilities        []CapabilityInstance   `json:"capabilities,omitempty"`
	SupportedParameters []string               `json:"supported_parameters,omitempty"`
	Limits              *Limits                `json:"limits,omitempty"`
	Pricing             *Pricing               `json:"pricing,omitempty"`
	Modalities          *Modalities            `json:"modalities,omitempty"`
	Sources             []Source               `json:"sources,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
}

type InterfaceProfile struct {
	ID                string            `json:"id"`
	Family            string            `json:"family"`
	Version           string            `json:"version"`
	Description       string            `json:"description,omitempty"`
	CanonicalMappings map[string]string `json:"canonical_mappings,omitempty"`
	Sources           []Source          `json:"sources,omitempty"`
}

type CapabilityDef struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Category         string   `json:"category,omitempty"`
	DetailsSchemaRef string   `json:"details_schema_ref,omitempty"`
	Sources          []Source `json:"sources,omitempty"`
}

type ParameterDef struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Category    string   `json:"category,omitempty"`
	Sources     []Source `json:"sources,omitempty"`
}

type Snapshot struct {
	Version           string             `json:"version"`
	GeneratedAt       time.Time          `json:"generated_at"`
	Labs              []Lab              `json:"labs"`
	LabModels         []LabModel         `json:"lab_models"`
	Providers         []Provider         `json:"providers"`
	ProviderModels    []ProviderModel    `json:"provider_models"`
	InterfaceProfiles []InterfaceProfile `json:"interface_profiles"`
	CapabilityDefs    []CapabilityDef    `json:"capability_defs"`
	ParameterDefs     []ParameterDef     `json:"parameter_defs"`
}
