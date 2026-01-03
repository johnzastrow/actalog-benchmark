package metrics

import (
	"context"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

// Common frontend asset patterns to look for in HTML
var (
	scriptPattern = regexp.MustCompile(`<script[^>]+src=["']([^"']+)["']`)
	linkPattern   = regexp.MustCompile(`<link[^>]+href=["']([^"']+)["']`)
)

// BenchmarkFrontend measures frontend asset loading performance
func BenchmarkFrontend(ctx context.Context, c *client.Client) *internal.FrontendResult {
	result := &internal.FrontendResult{
		Assets: make([]internal.AssetResult, 0),
	}

	// First, fetch the index.html
	indexResult := fetchAsset(ctx, c, "/", "html")
	result.IndexHTML = &indexResult

	if !indexResult.Success {
		return result
	}

	result.TotalSizeKB = indexResult.SizeKB
	result.TotalTimeMs = indexResult.ResponseMs

	// Parse HTML to find JS and CSS assets
	htmlContent := fetchContent(ctx, c, "/")
	if htmlContent == "" {
		return result
	}

	// Find all script sources
	scripts := scriptPattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range scripts {
		if len(match) > 1 {
			src := match[1]
			// Skip external scripts and inline data
			if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "data:") {
				continue
			}
			assetResult := fetchAsset(ctx, c, normalizePath(src), "js")
			result.Assets = append(result.Assets, assetResult)
			result.TotalSizeKB += assetResult.SizeKB
			result.TotalTimeMs += assetResult.ResponseMs
		}
	}

	// Find all CSS links
	links := linkPattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range links {
		if len(match) > 1 {
			href := match[1]
			// Only process CSS files, skip external
			if strings.HasPrefix(href, "http") || strings.HasPrefix(href, "data:") {
				continue
			}
			if strings.Contains(href, ".css") {
				assetResult := fetchAsset(ctx, c, normalizePath(href), "css")
				result.Assets = append(result.Assets, assetResult)
				result.TotalSizeKB += assetResult.SizeKB
				result.TotalTimeMs += assetResult.ResponseMs
			}
		}
	}

	return result
}

func fetchAsset(ctx context.Context, c *client.Client, path string, assetType string) internal.AssetResult {
	result := internal.AssetResult{
		Path: path,
		Type: assetType,
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

	result.Status = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Read body to get size
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = "failed to read body: " + err.Error()
		return result
	}

	result.SizeKB = float64(len(body)) / 1024.0

	return result
}

func fetchContent(ctx context.Context, c *client.Client, path string) string {
	resp, err := c.Get(ctx, path)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(body)
}

func normalizePath(path string) string {
	// Handle relative paths
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
