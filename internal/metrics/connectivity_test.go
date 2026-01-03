package metrics

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeasureConnectivity_HTTP(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()

	result := MeasureConnectivity(context.Background(), server.URL, 10*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Connected {
		t.Errorf("expected connected=true, error: %s", result.Error)
	}
	if result.Error != "" {
		t.Errorf("expected no error, got: %s", result.Error)
	}

	// DNS should have some time (even for localhost)
	if result.DNSMs < 0 {
		t.Error("expected non-negative DNS time")
	}

	// TCP connect should have some time
	if result.TCPMs <= 0 {
		t.Error("expected positive TCP connect time")
	}

	// HTTP (not HTTPS) should have no TLS time
	if result.TLSMs != 0 {
		t.Errorf("expected 0 TLS time for HTTP, got %f", result.TLSMs)
	}

	// Total should be sum of components
	expectedTotal := result.DNSMs + result.TCPMs + result.TLSMs
	if result.TotalMs != expectedTotal {
		t.Errorf("expected total %f, got %f", expectedTotal, result.TotalMs)
	}
}

func TestMeasureConnectivity_HTTPS(t *testing.T) {
	server := httptest.NewTLSServer(nil)
	defer server.Close()

	result := MeasureConnectivity(context.Background(), server.URL, 10*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// For TLS server, we expect TLS handshake time
	// Note: httptest.NewTLSServer uses a self-signed cert which may fail TLS verification
	// The test server uses localhost which should work
	if result.Connected {
		if result.TLSMs <= 0 {
			t.Error("expected positive TLS handshake time for HTTPS")
		}
	}
}

func TestMeasureConnectivity_InvalidURL(t *testing.T) {
	result := MeasureConnectivity(context.Background(), "://invalid-url", 10*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Connected {
		t.Error("expected connected=false for invalid URL")
	}
	if result.Error == "" {
		t.Error("expected error message for invalid URL")
	}
}

func TestMeasureConnectivity_UnreachableHost(t *testing.T) {
	// Use a non-routable IP address
	result := MeasureConnectivity(context.Background(), "http://192.0.2.1:12345", 1*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Connected {
		t.Error("expected connected=false for unreachable host")
	}
	if result.Error == "" {
		t.Error("expected error message for unreachable host")
	}
}

func TestMeasureConnectivity_DNSFailure(t *testing.T) {
	// Use a non-existent domain
	result := MeasureConnectivity(context.Background(), "http://this-domain-does-not-exist-12345.invalid", 5*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Connected {
		t.Error("expected connected=false for DNS failure")
	}
	if result.Error == "" {
		t.Error("expected error message for DNS failure")
	}
	// DNS time should still be recorded
	if result.DNSMs <= 0 {
		t.Error("expected positive DNS time even for failure")
	}
}

func TestMeasureConnectivity_DefaultPorts(t *testing.T) {
	tests := []struct {
		url          string
		expectedPort string
	}{
		{"http://example.com", "80"},
		{"https://example.com", "443"},
		{"http://example.com:8080", "8080"},
		{"https://example.com:8443", "8443"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			// We can't easily test the actual port used without mocking,
			// but we can at least verify the function doesn't panic
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result := MeasureConnectivity(ctx, tt.url, 100*time.Millisecond)
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			// Result will likely have an error due to timeout or unreachable,
			// but should not panic
		})
	}
}

func TestMeasureConnectivity_Timeout(t *testing.T) {
	// Use a non-routable IP that will timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := MeasureConnectivity(ctx, "http://10.255.255.1:12345", 100*time.Millisecond)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Connected {
		t.Error("expected connected=false for timeout")
	}
	// Should have an error (either timeout or connection refused)
	if result.Error == "" {
		t.Error("expected error message for timeout")
	}
}

func TestMeasureConnectivity_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := MeasureConnectivity(ctx, "http://example.com", 10*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Connected {
		t.Error("expected connected=false when context is cancelled")
	}
}
