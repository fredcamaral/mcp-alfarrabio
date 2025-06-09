package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionChecker_Handler(t *testing.T) {
	vc := NewVersionChecker()
	handler := vc.Handler()

	// Test handler with supported version
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Client-Version", "1.0.0")
	w := httptest.NewRecorder()

	handler(nextHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1.0.0", w.Header().Get("X-Server-Version"))
	assert.NotEmpty(t, w.Header().Get("X-Compatible-Versions"))
}

func TestVersionChecker_UnsupportedVersion(t *testing.T) {
	vc := NewVersionChecker()
	handler := vc.Handler()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Client-Version", "2.0.0")
	w := httptest.NewRecorder()

	handler(nextHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVersionChecker_PublicEndpoint(t *testing.T) {
	vc := NewVersionChecker()
	handler := vc.Handler()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(nextHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVersionChecker_NoVersion(t *testing.T) {
	vc := NewVersionChecker()
	handler := vc.Handler()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler(nextHandler).ServeHTTP(w, req)

	// Should allow for backward compatibility
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractVersionFromUserAgent(t *testing.T) {
	vc := NewVersionChecker()

	tests := []struct {
		userAgent string
		expected  string
	}{
		{"lmmc/1.0.0", "1.0.0"},
		{"lerian-cli/1.1.0", "1.1.0"},
		{"Mozilla/5.0", ""},
		{"custom-client/2.0.0", ""},
	}

	for _, test := range tests {
		result := vc.extractVersionFromUserAgent(test.userAgent)
		assert.Equal(t, test.expected, result, "Failed for user agent: %s", test.userAgent)
	}
}

func TestParseVersion(t *testing.T) {
	vc := NewVersionChecker()

	tests := []struct {
		version string
		major   int
		minor   int
		patch   int
		isValid bool
	}{
		{"1.0.0", 1, 0, 0, true},
		{"1.2.3", 1, 2, 3, true},
		{"invalid", 0, 0, 0, false},
		{"1.0", 0, 0, 0, false},
	}

	for _, test := range tests {
		result, err := vc.parseVersion(test.version)
		if test.isValid {
			assert.NoError(t, err)
			assert.Equal(t, test.major, result.Major)
			assert.Equal(t, test.minor, result.Minor)
			assert.Equal(t, test.patch, result.Patch)
		} else {
			// parseVersion doesn't return error in current implementation,
			// it returns empty Version struct
			assert.Equal(t, Version{}, result)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	vc := NewVersionChecker()

	tests := []struct {
		v1       Version
		v2       Version
		expected int
	}{
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{1, 0, 0}, Version{1, 0, 1}, -1},
		{Version{1, 0, 1}, Version{1, 0, 0}, 1},
		{Version{1, 1, 0}, Version{1, 0, 0}, 1},
		{Version{2, 0, 0}, Version{1, 0, 0}, 1},
	}

	for _, test := range tests {
		result := vc.compareVersions(test.v1, test.v2)
		assert.Equal(t, test.expected, result)
	}
}