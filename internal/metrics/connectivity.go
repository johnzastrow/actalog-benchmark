package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

// MeasureConnectivity measures DNS, TCP, and TLS connection timing
func MeasureConnectivity(ctx context.Context, targetURL string, timeout time.Duration) *internal.ConnectivityResult {
	result := &internal.ConnectivityResult{}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		result.Error = fmt.Sprintf("parse URL: %v", err)
		return result
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	// DNS Resolution
	dnsStart := time.Now()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	dnsDuration := time.Since(dnsStart)
	result.DNSMs = float64(dnsDuration.Microseconds()) / 1000.0

	if err != nil {
		result.Error = fmt.Sprintf("DNS lookup failed: %v", err)
		return result
	}

	if len(ips) == 0 {
		result.Error = "DNS lookup returned no addresses"
		return result
	}

	// TCP Connection
	address := net.JoinHostPort(ips[0].IP.String(), port)
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	tcpStart := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	tcpDuration := time.Since(tcpStart)
	result.TCPMs = float64(tcpDuration.Microseconds()) / 1000.0

	if err != nil {
		result.Error = fmt.Sprintf("TCP connection failed: %v", err)
		return result
	}

	// TLS Handshake (if HTTPS)
	if parsedURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			ServerName: host,
		}

		tlsStart := time.Now()
		tlsConn := tls.Client(conn, tlsConfig)
		err = tlsConn.HandshakeContext(ctx)
		tlsDuration := time.Since(tlsStart)
		result.TLSMs = float64(tlsDuration.Microseconds()) / 1000.0

		if err != nil {
			conn.Close()
			result.Error = fmt.Sprintf("TLS handshake failed: %v", err)
			return result
		}

		tlsConn.Close()
	} else {
		conn.Close()
	}

	result.TotalMs = result.DNSMs + result.TCPMs + result.TLSMs
	result.Connected = true

	return result
}
