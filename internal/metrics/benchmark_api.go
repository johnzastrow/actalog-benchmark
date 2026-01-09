package metrics

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

// RunBenchmarkAPI calls the /api/benchmark endpoint and returns structured results
func RunBenchmarkAPI(ctx context.Context, c *client.Client, includeConcurrent bool) *internal.BenchmarkAPIResult {
	result := &internal.BenchmarkAPIResult{}

	// Build URL with query params
	path := "/api/benchmark"
	if includeConcurrent {
		path += "?concurrent=true"
	}

	start := time.Now()
	resp, err := c.Post(ctx, path, nil)
	result.TotalDurationMs = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode

	if resp.StatusCode != 200 {
		result.Success = false
		body, _ := io.ReadAll(resp.Body)
		result.Error = string(body)
		return result
	}

	// Parse the response
	var apiResp internal.BenchmarkAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		result.Success = false
		result.Error = "failed to decode benchmark response: " + err.Error()
		return result
	}

	result.Success = true
	result.Response = &apiResp

	return result
}
