package metrics

import (
	"context"
	"io"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

// PublicEndpoints are endpoints that don't require authentication
var PublicEndpoints = []string{
	"/api/version",
	"/health",
}

// AuthenticatedEndpoints are endpoints that require authentication
var AuthenticatedEndpoints = []string{
	"/api/workouts",
	"/api/movements",
	"/api/wods",
	"/api/pr-movements",
	"/api/notifications/count",
}

// BenchmarkEndpoint measures the response time for a single endpoint
func BenchmarkEndpoint(ctx context.Context, c *client.Client, path string) internal.EndpointResult {
	result := internal.EndpointResult{
		Path: path,
	}

	start := time.Now()
	resp, err := c.Get(ctx, path)
	result.ResponseMs = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result
	}
	defer resp.Body.Close()

	// Drain the body to ensure accurate timing
	io.Copy(io.Discard, resp.Body)

	result.Status = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	return result
}

// BenchmarkEndpoints measures multiple endpoints and returns results
func BenchmarkEndpoints(ctx context.Context, c *client.Client, paths []string) []internal.EndpointResult {
	results := make([]internal.EndpointResult, 0, len(paths))

	for _, path := range paths {
		result := BenchmarkEndpoint(ctx, c, path)
		results = append(results, result)
	}

	return results
}

// GetEndpointsForAuth returns the appropriate endpoints based on auth status
func GetEndpointsForAuth(authenticated bool) []string {
	if authenticated {
		// Return both public and authenticated endpoints
		all := make([]string, 0, len(PublicEndpoints)+len(AuthenticatedEndpoints))
		all = append(all, PublicEndpoints...)
		all = append(all, AuthenticatedEndpoints...)
		return all
	}
	return PublicEndpoints
}
