// Package foodimport runs a scheduled bulk import of the global food catalog
// from external nutrition sources (USDA, OpenFoodFacts, TACO) into the local
// foods table, so per-item lookups can hit a warm local cache instead of a
// live API call. Disabled by default (FOOD_IMPORT_ENABLED); opt-in per
// FOOD_IMPORT_SOURCES.
package foodimport

import (
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/openfoodfacts"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/usda"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

// LocalPaths returns the configured on-disk dataset for each file-backed bulk
// source. Empty paths deliberately remain API (or embedded TACO) mode.
func LocalPaths(cfg *config.Config) map[string]string {
	paths := make(map[string]string, 3)
	if cfg.USDABulkFile != "" {
		paths["usda"] = cfg.USDABulkFile
	}
	if cfg.OFFBulkFile != "" {
		paths["openfoodfacts"] = cfg.OFFBulkFile
	}
	if cfg.TacoDataPath != "" {
		paths["taco"] = cfg.TacoDataPath
	}
	return paths
}

// BuildSource constructs the ports.BulkSource and ports.BulkFilter for a
// named source ("usda", "openfoodfacts", "taco"), reading all adapter-specific
// settings from cfg. Returns an error for an unrecognized name, or if a
// required setting (e.g. USDA_FDC_API_KEY) is missing when that source needs it.
func BuildSource(name string, cfg *config.Config) (ports.BulkSource, ports.BulkFilter, error) {
	switch name {
	case "usda":
		if cfg.USDAFDCAPIKey == "" {
			return nil, ports.BulkFilter{}, fmt.Errorf("foodimport: USDA_FDC_API_KEY required for source %q", name)
		}
		src := usda.NewBulk(cfg.USDAFDCAPIKey, cfg.USDABulkFile)
		filter := ports.BulkFilter{DataTypes: cfg.USDABulkDataTypes, MaxRows: cfg.USDABulkMaxRows}
		return src, filter, nil
	case "openfoodfacts":
		src := openfoodfacts.NewBulk(cfg.OFFBulkFile)
		filter := ports.BulkFilter{MinPopularity: cfg.OFFBulkMinPopularity, MaxRows: cfg.OFFBulkMaxRows}
		return src, filter, nil
	case "taco":
		src, err := taco.New(cfg.TacoDataPath)
		if err != nil {
			return nil, ports.BulkFilter{}, fmt.Errorf("foodimport: build taco source: %w", err)
		}
		filter := ports.BulkFilter{MaxRows: cfg.TacoBulkMaxRows}
		return src, filter, nil
	default:
		return nil, ports.BulkFilter{}, fmt.Errorf("foodimport: unknown source %q", name)
	}
}
