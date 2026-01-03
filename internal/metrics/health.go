package metrics

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

// HealthResponse represents the /health endpoint response
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

// CheckHealth checks the health endpoint and returns results
func CheckHealth(ctx context.Context, c *client.Client) *internal.HealthResult {
	result := &internal.HealthResult{}

	start := time.Now()
	resp, err := c.Get(ctx, "/health")
	result.ResponseMs = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode

	if resp.StatusCode != 200 {
		result.Status = "unhealthy"
		body, _ := io.ReadAll(resp.Body)
		result.Error = string(body)
		return result
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		result.Status = "error"
		result.Error = "failed to decode health response: " + err.Error()
		return result
	}

	result.Status = healthResp.Status

	return result
}
