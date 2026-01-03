package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

func TestBenchmarkFrontend_Basic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<link href="/assets/style.css" rel="stylesheet">
</head>
<body>
	<script src="/assets/app.js"></script>
</body>
</html>`))
		case "/assets/style.css":
			w.Header().Set("Content-Type", "text/css")
			w.Write([]byte(`body { margin: 0; }`))
		case "/assets/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			w.Write([]byte(`console.log("hello");`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkFrontend(context.Background(), c)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check index.html
	if result.IndexHTML == nil {
		t.Fatal("expected non-nil IndexHTML")
	}
	if !result.IndexHTML.Success {
		t.Error("expected IndexHTML success")
	}
	if result.IndexHTML.Type != "html" {
		t.Errorf("expected type 'html', got '%s'", result.IndexHTML.Type)
	}
	if result.IndexHTML.SizeKB <= 0 {
		t.Error("expected positive size for IndexHTML")
	}

	// Check assets were discovered
	if len(result.Assets) != 2 {
		t.Errorf("expected 2 assets (CSS and JS), got %d", len(result.Assets))
	}

	// Check totals
	if result.TotalSizeKB <= 0 {
		t.Error("expected positive total size")
	}
	if result.TotalTimeMs <= 0 {
		t.Error("expected positive total time")
	}
}

func TestBenchmarkFrontend_NoAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>Hello</body></html>`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkFrontend(context.Background(), c)

	if result.IndexHTML == nil {
		t.Fatal("expected non-nil IndexHTML")
	}
	if !result.IndexHTML.Success {
		t.Error("expected IndexHTML success")
	}
	if len(result.Assets) != 0 {
		t.Errorf("expected 0 assets, got %d", len(result.Assets))
	}
}

func TestBenchmarkFrontend_IndexFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkFrontend(context.Background(), c)

	if result.IndexHTML == nil {
		t.Fatal("expected non-nil IndexHTML")
	}
	if result.IndexHTML.Success {
		t.Error("expected IndexHTML failure for 404")
	}
	// Should return early without checking assets
	if len(result.Assets) != 0 {
		t.Errorf("expected 0 assets when index fails, got %d", len(result.Assets))
	}
}

func TestBenchmarkFrontend_ExternalScriptsIgnored(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<script src="https://cdn.example.com/external.js"></script>
	<script src="/assets/local.js"></script>
</head>
</html>`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkFrontend(context.Background(), c)

	// Should only include local asset, not external
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (local only), got %d", len(result.Assets))
	}
}

func TestBenchmarkFrontend_DataURIsIgnored(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<link href="data:text/css,body{}" rel="stylesheet">
	<link href="/assets/style.css" rel="stylesheet">
</head>
</html>`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkFrontend(context.Background(), c)

	// Should only include real CSS file, not data URI
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (real CSS only), got %d", len(result.Assets))
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/assets/app.js", "/assets/app.js"},
		{"assets/app.js", "/assets/app.js"},
		{"/style.css", "/style.css"},
		{"style.css", "/style.css"},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestScriptPattern(t *testing.T) {
	html := `<script src="/assets/app.js"></script><script src='/assets/vendor.js'></script>`
	matches := scriptPattern.FindAllStringSubmatch(html, -1)

	if len(matches) != 2 {
		t.Errorf("expected 2 script matches, got %d", len(matches))
	}

	expected := []string{"/assets/app.js", "/assets/vendor.js"}
	for i, match := range matches {
		if len(match) < 2 {
			t.Errorf("expected match to have capture group")
			continue
		}
		if match[1] != expected[i] {
			t.Errorf("expected '%s', got '%s'", expected[i], match[1])
		}
	}
}

func TestLinkPattern(t *testing.T) {
	html := `<link href="/assets/style.css" rel="stylesheet"><link href='/assets/vendor.css' rel="stylesheet">`
	matches := linkPattern.FindAllStringSubmatch(html, -1)

	if len(matches) != 2 {
		t.Errorf("expected 2 link matches, got %d", len(matches))
	}

	expected := []string{"/assets/style.css", "/assets/vendor.css"}
	for i, match := range matches {
		if len(match) < 2 {
			t.Errorf("expected match to have capture group")
			continue
		}
		if match[1] != expected[i] {
			t.Errorf("expected '%s', got '%s'", expected[i], match[1])
		}
	}
}
