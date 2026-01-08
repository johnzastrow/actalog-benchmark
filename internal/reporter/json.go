package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// If outputPath is a directory, generates a timestamped filename
// If outputPath is a file, uses it directly
func (j *JSON) Report(result *internal.BenchmarkResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal results: %w", err)
	}

	// Determine the actual file path
	outputFile := j.outputPath

	// Check if outputPath is a directory or should be treated as one
	info, err := os.Stat(j.outputPath)
	isDir := (err == nil && info.IsDir()) || strings.HasSuffix(j.outputPath, "/")

	if isDir || !strings.HasSuffix(strings.ToLower(j.outputPath), ".json") {
		// Treat as directory, generate timestamped filename
		timestamp := result.Timestamp.Format("2006-01-02_150405")
		filename := fmt.Sprintf("benchmark_%s.json", timestamp)
		outputFile = filepath.Join(j.outputPath, filename)
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(outputFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create directory: %w", err)
		}
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return outputFile, nil
}
