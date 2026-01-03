package reporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

// JSON reporter for machine-readable output
type JSON struct {
	outputPath string
}

// NewJSON creates a new JSON reporter
func NewJSON(outputPath string) *JSON {
	return &JSON{outputPath: outputPath}
}

// Report writes the benchmark results to a JSON file
func (j *JSON) Report(result *internal.BenchmarkResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	if err := os.WriteFile(j.outputPath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Printf("Results written to: %s\n", j.outputPath)
	return nil
}
