package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wingmodels/internal/models"

	"github.com/BurntSushi/toml"
)

// LoadResult holds all parsed entities from the data directory.
type LoadResult struct {
	Labs              []models.Lab
	LabModels         []models.LabModel
	Providers         []models.Provider
	ProviderModels    []models.ProviderModel
	InterfaceProfiles []models.InterfaceProfile
	CapabilityDefs    []models.CapabilityDef
	ParameterDefs     []models.ParameterDef
}

// yearMonthRe matches bare YYYY-MM values that are not valid TOML dates.
// These need to be quoted as strings before TOML parsing.
var yearMonthRe = regexp.MustCompile(`(?m)(=\s*)(\d{4}-\d{2})\s*$`)

// Load discovers and parses all TOML files from the data directory.
func Load(dataDir string) (*LoadResult, error) {
	result := &LoadResult{}
	var errs []string

	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes for consistent path parsing.
		rel = filepath.ToSlash(rel)

		entityType := classifyPath(rel)
		if entityType == "" {
			errs = append(errs, fmt.Sprintf("unknown entity type for file: %s", rel))
			return nil
		}

		if loadErr := loadFile(path, entityType, result); loadErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rel, loadErr))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking data directory: %w", err)
	}

	if len(errs) > 0 {
		return result, fmt.Errorf("load errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return result, nil
}

// classifyPath determines the entity type from a relative path within the data directory.
func classifyPath(rel string) string {
	parts := strings.Split(rel, "/")

	switch {
	// data/labs/{lab_id}/lab.toml
	case len(parts) == 3 && parts[0] == "labs" && parts[2] == "lab.toml":
		return "lab"
	// data/labs/{lab_id}/models/{model_id}.toml
	case len(parts) == 4 && parts[0] == "labs" && parts[2] == "models":
		return "lab_model"
	// data/providers/{provider_id}/provider.toml
	case len(parts) == 3 && parts[0] == "providers" && parts[2] == "provider.toml":
		return "provider"
	// data/providers/{provider_id}/models/{model_id}.toml
	case len(parts) == 4 && parts[0] == "providers" && parts[2] == "models":
		return "provider_model"
	// data/capabilities/{id}.toml
	case len(parts) == 2 && parts[0] == "capabilities":
		return "capability"
	// data/parameters/{id}.toml
	case len(parts) == 2 && parts[0] == "parameters":
		return "parameter"
	// data/interfaces/{id}.toml
	case len(parts) == 2 && parts[0] == "interfaces":
		return "interface"
	}

	return ""
}

// loadFile decodes a TOML file and appends the result to LoadResult.
// Strategy: read raw text -> fix bare YYYY-MM dates -> decode TOML -> map[string]interface{}
// -> convert time.Time to strings -> JSON marshal -> JSON unmarshal into typed struct.
func loadFile(path string, entityType string, result *LoadResult) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Pre-process: quote bare YYYY-MM values that aren't valid TOML dates.
	// Match lines like: knowledge_cutoff = 2024-04
	// Replace with: knowledge_cutoff = "2024-04"
	content := yearMonthRe.ReplaceAllString(string(raw), `${1}"${2}"`)

	var m map[string]interface{}
	if _, err := toml.Decode(content, &m); err != nil {
		return fmt.Errorf("decoding TOML: %w", err)
	}

	// Recursively convert time.Time values to YYYY-MM-DD strings.
	convertDates(m)

	// Marshal the cleaned map to JSON, then unmarshal into the typed struct.
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	switch entityType {
	case "lab":
		var v models.Lab
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling Lab: %w", err)
		}
		result.Labs = append(result.Labs, v)

	case "lab_model":
		var v models.LabModel
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling LabModel: %w", err)
		}
		result.LabModels = append(result.LabModels, v)

	case "provider":
		var v models.Provider
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling Provider: %w", err)
		}
		result.Providers = append(result.Providers, v)

	case "provider_model":
		var v models.ProviderModel
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling ProviderModel: %w", err)
		}
		result.ProviderModels = append(result.ProviderModels, v)

	case "capability":
		var v models.CapabilityDef
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling CapabilityDef: %w", err)
		}
		result.CapabilityDefs = append(result.CapabilityDefs, v)

	case "parameter":
		var v models.ParameterDef
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling ParameterDef: %w", err)
		}
		result.ParameterDefs = append(result.ParameterDefs, v)

	case "interface":
		var v models.InterfaceProfile
		if err := json.Unmarshal(jsonBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling InterfaceProfile: %w", err)
		}
		result.InterfaceProfiles = append(result.InterfaceProfiles, v)
	}

	return nil
}

// convertDates recursively walks a map and converts time.Time values
// to YYYY-MM-DD strings.
func convertDates(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case time.Time:
			m[k] = val.Format("2006-01-02")
		case map[string]interface{}:
			convertDates(val)
		case []map[string]interface{}:
			for _, item := range val {
				convertDates(item)
			}
		case []interface{}:
			for i, item := range val {
				if sub, ok := item.(map[string]interface{}); ok {
					convertDates(sub)
					val[i] = sub
				}
			}
		}
	}
}
