// Package testutil provides common test utilities for AI client testing.
package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// NewClientTestCase represents a test case for client creation
type NewClientTestCase struct {
	Name    string
	APIKey  string
	BaseURL string
	WantErr bool
}

// GetDefaultNewClientTestCases returns standard test cases for client creation
func GetDefaultNewClientTestCases(defaultBaseURL string) []NewClientTestCase {
	return []NewClientTestCase{
		{
			Name:    "valid configuration",
			APIKey:  "test-api-key",
			BaseURL: defaultBaseURL,
			WantErr: false,
		},
		{
			Name:    "empty API key",
			APIKey:  "",
			BaseURL: defaultBaseURL,
			WantErr: true,
		},
		{
			Name:    "custom base URL",
			APIKey:  "test-api-key",
			BaseURL: "https://custom.example.com/v1",
			WantErr: false,
		},
		{
			Name:    "empty base URL uses default",
			APIKey:  "test-api-key",
			BaseURL: "",
			WantErr: false,
		},
	}
}

// ClientTestInterface defines methods that clients should have for testing
type ClientTestInterface interface {
	GetAPIKey() string
	GetBaseURL() string
}

// ClientWithAPIKey interface for clients that expose their API key
type ClientWithAPIKey interface {
	GetAPIKey() string
}

// ClientWithBaseURL interface for clients that expose their base URL
type ClientWithBaseURL interface {
	GetBaseURL() string
}

// AssertClientCreation validates client creation test results
// This is a generic function that can work with any client that has apiKey and baseURL fields
func AssertClientCreation(t *testing.T, client interface{}, err error, testCase NewClientTestCase, expectedDefaultURL string, getAPIKey, getBaseURL func(interface{}) string) {
	t.Helper()

	if testCase.WantErr {
		assert.Error(t, err)
		assert.Nil(t, client)
	} else {
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, testCase.APIKey, getAPIKey(client))
		if testCase.BaseURL == "" {
			assert.Equal(t, expectedDefaultURL, getBaseURL(client))
		} else {
			assert.Equal(t, testCase.BaseURL, getBaseURL(client))
		}
	}
}

// ValidateRequestTestCase represents a test case for request validation
type ValidateRequestTestCase struct {
	Name    string
	Request interface{}
	WantErr bool
	ErrMsg  string
}

// AssertRequestValidation validates request validation test results
func AssertRequestValidation(t *testing.T, err error, testCase ValidateRequestTestCase) {
	t.Helper()

	if testCase.WantErr {
		assert.Error(t, err)
		if testCase.ErrMsg != "" {
			assert.Contains(t, err.Error(), testCase.ErrMsg)
		}
	} else {
		assert.NoError(t, err)
	}
}

// TestValidateRequestsWithConverterAndValidator runs validation tests with a custom validator
func TestValidateRequestsWithConverterAndValidator(t *testing.T, testCases []ValidateRequestTestCase, converter func(CompletionRequest) interface{}, validator func(interface{}) error) {
	t.Helper()

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			// Convert testutil.CompletionRequest to the actual type
			testReq := tt.Request.(CompletionRequest)
			actualReq := converter(testReq)
			err := validator(actualReq)
			AssertRequestValidation(t, err, tt)
		})
	}
}

// CompletionRequest represents a generic completion request for testing
// This mirrors the actual CompletionRequest structure
type CompletionRequest struct {
	Prompt        string                 `json:"prompt"`
	Model         string                 `json:"model"`
	SystemMessage string                 `json:"system_message,omitempty"`
	MaxTokens     int                    `json:"max_tokens,omitempty"`
	Temperature   float64                `json:"temperature,omitempty"`
	TopP          float64                `json:"top_p,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
}

// TestClientRequestValidation provides a standardized way to test client request validation
// This function reduces duplication between similar validation tests
func TestClientRequestValidation(
	t *testing.T,
	defaultModel string,
	maxTemperature float64,
	temperatureErrorMessage string,
	validator func(interface{}) error,
) {
	t.Helper()

	// Get standard test cases
	tests := GetDefaultValidateRequestTestCases(defaultModel, maxTemperature)

	// Update the temperature error message for this client
	for i := range tests {
		if tests[i].Name == "invalid temperature" {
			tests[i].ErrMsg = temperatureErrorMessage
			break
		}
	}

	// Standard converter from testutil.CompletionRequest to interface{}
	converter := func(testReq CompletionRequest) interface{} {
		return testReq
	}

	TestValidateRequestsWithConverterAndValidator(t, tests, converter, validator)
}

// GetDefaultValidateRequestTestCases returns standard validation test cases
func GetDefaultValidateRequestTestCases(defaultModel string, maxTemp float64) []ValidateRequestTestCase {
	return []ValidateRequestTestCase{
		{
			Name: "valid request",
			Request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  defaultModel,
			},
			WantErr: false,
		},
		{
			Name: "empty prompt",
			Request: CompletionRequest{
				Prompt: "",
				Model:  defaultModel,
			},
			WantErr: true,
			ErrMsg:  "prompt cannot be empty",
		},
		{
			Name: "empty model",
			Request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "",
			},
			WantErr: true,
			ErrMsg:  "model cannot be empty",
		},
		{
			Name: "negative max tokens",
			Request: CompletionRequest{
				Prompt:    "Test prompt",
				Model:     defaultModel,
				MaxTokens: -1,
			},
			WantErr: true,
			ErrMsg:  "max tokens must be positive",
		},
		{
			Name: "invalid temperature",
			Request: CompletionRequest{
				Prompt:      "Test prompt",
				Model:       defaultModel,
				Temperature: maxTemp + 0.5,
			},
			WantErr: true,
		},
	}
}
