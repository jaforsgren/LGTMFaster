package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

// LoggingTransport wraps an http.RoundTripper to log all requests and responses
type LoggingTransport struct {
	Transport http.RoundTripper
}

// NewLoggingTransport creates a new logging transport wrapper
func NewLoggingTransport(transport http.RoundTripper) *LoggingTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &LoggingTransport{
		Transport: transport,
	}
}

// RoundTrip executes a single HTTP transaction with full logging
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log request
	t.logRequest(req)

	// Execute the request
	resp, err := t.Transport.RoundTrip(req)

	duration := time.Since(start)

	// Log response
	if err != nil {
		logger.LogError("HTTP_REQUEST", fmt.Sprintf("%s %s", req.Method, req.URL.String()), err)
		logger.Log("HTTP: %s %s - ERROR after %v: %v", req.Method, req.URL.Path, duration, err)
		return nil, err
	}

	t.logResponse(req, resp, duration)

	return resp, nil
}

func (t *LoggingTransport) logRequest(req *http.Request) {
	var buf bytes.Buffer

	// Request line
	buf.WriteString(fmt.Sprintf("=== HTTP REQUEST ===\n"))
	buf.WriteString(fmt.Sprintf("%s %s %s\n", req.Method, req.URL.String(), req.Proto))

	// Headers
	buf.WriteString("Headers:\n")
	for name, values := range req.Header {
		// Redact sensitive headers
		if isSensitiveHeader(name) {
			buf.WriteString(fmt.Sprintf("  %s: [REDACTED]\n", name))
		} else {
			for _, value := range values {
				buf.WriteString(fmt.Sprintf("  %s: %s\n", name, value))
			}
		}
	}

	// Body (if present and not too large)
	if req.Body != nil && req.ContentLength > 0 && req.ContentLength < 10000 {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			// Restore the body for the actual request
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			buf.WriteString(fmt.Sprintf("Body (%d bytes):\n", len(bodyBytes)))
			buf.WriteString(string(bodyBytes))
			buf.WriteString("\n")
		}
	} else if req.ContentLength > 0 {
		buf.WriteString(fmt.Sprintf("Body: (%d bytes, too large to log)\n", req.ContentLength))
	}

	buf.WriteString("===================\n")

	logger.Log(buf.String())
}

func (t *LoggingTransport) logResponse(req *http.Request, resp *http.Response, duration time.Duration) {
	var buf bytes.Buffer

	// Response line
	buf.WriteString(fmt.Sprintf("=== HTTP RESPONSE ===\n"))
	buf.WriteString(fmt.Sprintf("%s %s - %s (%v)\n", req.Method, req.URL.Path, resp.Status, duration))

	// Headers
	buf.WriteString("Headers:\n")
	for name, values := range resp.Header {
		for _, value := range values {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", name, value))
		}
	}

	// Body (if present and not too large)
	if resp.Body != nil && resp.ContentLength != 0 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			// Restore the body for the caller
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > 0 && len(bodyBytes) < 10000 {
				buf.WriteString(fmt.Sprintf("Body (%d bytes):\n", len(bodyBytes)))
				buf.WriteString(string(bodyBytes))
				buf.WriteString("\n")
			} else if len(bodyBytes) > 0 {
				buf.WriteString(fmt.Sprintf("Body: (%d bytes, too large to log)\n", len(bodyBytes)))
			}
		}
	}

	buf.WriteString("====================\n")

	logger.Log(buf.String())
}

func isSensitiveHeader(name string) bool {
	lowerName := strings.ToLower(name)
	sensitiveHeaders := []string{
		"authorization",
		"x-api-key",
		"api-key",
		"x-auth-token",
		"cookie",
		"set-cookie",
	}

	for _, sensitive := range sensitiveHeaders {
		if lowerName == sensitive {
			return true
		}
	}

	return false
}
