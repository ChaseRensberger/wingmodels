package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"wingmodels/internal/loader"
	"wingmodels/internal/models"
)

// Compile loads TOML data, validates, and emits build artifacts.
func Compile(dataDir, buildDir string) error {
	fmt.Println("Loading TOML files...")
	result, err := loader.Load(dataDir)
	if err != nil {
		return fmt.Errorf("loading data: %w", err)
	}

	fmt.Println("Validating required fields...")
	if err := validateRequired(result); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println("Validating cross-references...")
	if err := validateRefs(result); err != nil {
		return fmt.Errorf("reference validation failed: %w", err)
	}

	fmt.Println("Sorting entities...")
	sortAll(result)

	fmt.Println("Writing build artifacts...")
	if err := writeBuild(result, buildDir); err != nil {
		return fmt.Errorf("writing build output: %w", err)
	}

	fmt.Printf("\nCompiled: %d labs, %d lab models, %d providers, %d provider models, %d interface profiles, %d capability defs, %d parameter defs\n",
		len(result.Labs),
		len(result.LabModels),
		len(result.Providers),
		len(result.ProviderModels),
		len(result.InterfaceProfiles),
		len(result.CapabilityDefs),
		len(result.ParameterDefs),
	)

	return nil
}

// validateRequired checks that all entities have their required fields populated.
func validateRequired(r *loader.LoadResult) error {
	var errs []string

	for _, v := range r.Labs {
		if v.ID == "" {
			errs = append(errs, "lab missing id")
		}
		if v.DisplayName == "" {
			errs = append(errs, fmt.Sprintf("lab %q missing display_name", v.ID))
		}
	}

	for _, v := range r.LabModels {
		if v.ID == "" {
			errs = append(errs, "lab_model missing id")
		}
		if v.LabID == "" {
			errs = append(errs, fmt.Sprintf("lab_model %q missing lab_id", v.ID))
		}
		if v.DisplayName == "" {
			errs = append(errs, fmt.Sprintf("lab_model %q missing display_name", v.ID))
		}
	}

	for _, v := range r.Providers {
		if v.ID == "" {
			errs = append(errs, "provider missing id")
		}
		if v.DisplayName == "" {
			errs = append(errs, fmt.Sprintf("provider %q missing display_name", v.ID))
		}
	}

	for _, v := range r.ProviderModels {
		if v.ID == "" {
			errs = append(errs, "provider_model missing id")
		}
		if v.DisplayName == "" {
			errs = append(errs, fmt.Sprintf("provider_model %q missing display_name", v.ID))
		}
	}

	for _, v := range r.InterfaceProfiles {
		if v.ID == "" {
			errs = append(errs, "interface_profile missing id")
		}
		if v.Family == "" {
			errs = append(errs, fmt.Sprintf("interface_profile %q missing family", v.ID))
		}
		if v.Version == "" {
			errs = append(errs, fmt.Sprintf("interface_profile %q missing version", v.ID))
		}
	}

	for _, v := range r.CapabilityDefs {
		if v.ID == "" {
			errs = append(errs, "capability_def missing id")
		}
		if v.Name == "" {
			errs = append(errs, fmt.Sprintf("capability_def %q missing name", v.ID))
		}
		if v.Description == "" {
			errs = append(errs, fmt.Sprintf("capability_def %q missing description", v.ID))
		}
	}

	for _, v := range r.ParameterDefs {
		if v.ID == "" {
			errs = append(errs, "parameter_def missing id")
		}
		if v.Name == "" {
			errs = append(errs, fmt.Sprintf("parameter_def %q missing name", v.ID))
		}
		if v.Description == "" {
			errs = append(errs, fmt.Sprintf("parameter_def %q missing description", v.ID))
		}
		if v.Type == "" {
			errs = append(errs, fmt.Sprintf("parameter_def %q missing type", v.ID))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d errors:\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}

// validateRefs checks all foreign key relationships.
func validateRefs(r *loader.LoadResult) error {
	var errs []string

	// Build lookup sets.
	labIDs := make(map[string]bool, len(r.Labs))
	for _, v := range r.Labs {
		labIDs[v.ID] = true
	}

	labModelIDs := make(map[string]bool, len(r.LabModels))
	for _, v := range r.LabModels {
		labModelIDs[v.ID] = true
	}

	providerIDs := make(map[string]bool, len(r.Providers))
	for _, v := range r.Providers {
		providerIDs[v.ID] = true
	}

	interfaceIDs := make(map[string]bool, len(r.InterfaceProfiles))
	for _, v := range r.InterfaceProfiles {
		interfaceIDs[v.ID] = true
	}

	capabilityIDs := make(map[string]bool, len(r.CapabilityDefs))
	for _, v := range r.CapabilityDefs {
		capabilityIDs[v.ID] = true
	}

	parameterIDs := make(map[string]bool, len(r.ParameterDefs))
	for _, v := range r.ParameterDefs {
		parameterIDs[v.ID] = true
	}

	// lab_model.lab_id -> lab
	for _, v := range r.LabModels {
		if !labIDs[v.LabID] {
			errs = append(errs, fmt.Sprintf("lab_model %q references unknown lab %q", v.ID, v.LabID))
		}
	}

	// provider_model.id -> provider and lab_model
	for _, v := range r.ProviderModels {
		parts := strings.SplitN(v.ID, "/", 2)
		if len(parts) != 2 {
			errs = append(errs, fmt.Sprintf("provider_model %q has invalid id format (expected provider_id/model_id)", v.ID))
			continue
		}
		providerPart := parts[0]
		modelPart := parts[1]

		if !providerIDs[providerPart] {
			errs = append(errs, fmt.Sprintf("provider_model %q references unknown provider %q", v.ID, providerPart))
		}
		if !labModelIDs[modelPart] {
			errs = append(errs, fmt.Sprintf("provider_model %q references unknown lab model %q", v.ID, modelPart))
		}

		// interface_profiles
		for _, ip := range v.InterfaceProfiles {
			if !interfaceIDs[ip] {
				errs = append(errs, fmt.Sprintf("provider_model %q references unknown interface profile %q", v.ID, ip))
			}
		}

		// capabilities
		for _, cap := range v.Capabilities {
			if !capabilityIDs[cap.ID] {
				errs = append(errs, fmt.Sprintf("provider_model %q references unknown capability %q", v.ID, cap.ID))
			}
		}

		// supported_parameters
		for _, p := range v.SupportedParameters {
			if !parameterIDs[p] {
				errs = append(errs, fmt.Sprintf("provider_model %q references unknown parameter %q", v.ID, p))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d errors:\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}

// sortAll sorts all entity slices by ID for deterministic output.
func sortAll(r *loader.LoadResult) {
	sort.Slice(r.Labs, func(i, j int) bool { return r.Labs[i].ID < r.Labs[j].ID })
	sort.Slice(r.LabModels, func(i, j int) bool { return r.LabModels[i].ID < r.LabModels[j].ID })
	sort.Slice(r.Providers, func(i, j int) bool { return r.Providers[i].ID < r.Providers[j].ID })
	sort.Slice(r.ProviderModels, func(i, j int) bool { return r.ProviderModels[i].ID < r.ProviderModels[j].ID })
	sort.Slice(r.InterfaceProfiles, func(i, j int) bool { return r.InterfaceProfiles[i].ID < r.InterfaceProfiles[j].ID })
	sort.Slice(r.CapabilityDefs, func(i, j int) bool { return r.CapabilityDefs[i].ID < r.CapabilityDefs[j].ID })
	sort.Slice(r.ParameterDefs, func(i, j int) bool { return r.ParameterDefs[i].ID < r.ParameterDefs[j].ID })
}

// writeBuild creates the build/ directory and writes api.json and indexes.
func writeBuild(r *loader.LoadResult, buildDir string) error {
	indexDir := filepath.Join(buildDir, "indexes")

	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return fmt.Errorf("creating build directories: %w", err)
	}

	// Write api.json
	snapshot := models.Snapshot{
		Version:           "v1",
		GeneratedAt:       time.Now().UTC(),
		Labs:              r.Labs,
		LabModels:         r.LabModels,
		Providers:         r.Providers,
		ProviderModels:    r.ProviderModels,
		InterfaceProfiles: r.InterfaceProfiles,
		CapabilityDefs:    r.CapabilityDefs,
		ParameterDefs:     r.ParameterDefs,
	}

	if err := writeJSON(filepath.Join(buildDir, "api.json"), snapshot); err != nil {
		return fmt.Errorf("writing api.json: %w", err)
	}
	fmt.Println("  -> build/api.json")

	// Build indexes
	byProvider := make(map[string][]string)
	byModel := make(map[string][]string)
	byCapability := make(map[string][]string)

	for _, pm := range r.ProviderModels {
		parts := strings.SplitN(pm.ID, "/", 2)
		if len(parts) == 2 {
			providerID := parts[0]
			modelID := parts[1]
			byProvider[providerID] = append(byProvider[providerID], pm.ID)
			byModel[modelID] = append(byModel[modelID], pm.ID)
		}

		for _, cap := range pm.Capabilities {
			byCapability[cap.ID] = append(byCapability[cap.ID], pm.ID)
		}
	}

	if err := writeJSON(filepath.Join(indexDir, "provider-models-by-provider.json"), byProvider); err != nil {
		return fmt.Errorf("writing provider index: %w", err)
	}
	fmt.Println("  -> build/indexes/provider-models-by-provider.json")

	if err := writeJSON(filepath.Join(indexDir, "provider-models-by-model.json"), byModel); err != nil {
		return fmt.Errorf("writing model index: %w", err)
	}
	fmt.Println("  -> build/indexes/provider-models-by-model.json")

	if err := writeJSON(filepath.Join(indexDir, "provider-models-by-capability.json"), byCapability); err != nil {
		return fmt.Errorf("writing capability index: %w", err)
	}
	fmt.Println("  -> build/indexes/provider-models-by-capability.json")

	return nil
}

// writeJSON marshals a value to indented JSON and writes it to a file.
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
