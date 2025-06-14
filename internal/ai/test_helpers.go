package ai

import (
	"testing"

	"lerian-mcp-memory/internal/ai/testutil"
)

// testClientValidation is a shared helper function for testing client validation
// This reduces duplication between different client validation tests
func testClientValidation(t *testing.T, defaultModel string, maxTemperature float64, temperatureErrorMessage string, validator func(interface{}) error) {
	t.Helper()

	// Get standard test cases and update for client specifics
	tests := testutil.GetDefaultValidateRequestTestCases(defaultModel, maxTemperature)
	// Update the temperature error message for this client
	for i := range tests {
		if tests[i].Name == "invalid temperature" {
			tests[i].ErrMsg = temperatureErrorMessage
			break
		}
	}

	// Converter from testutil.CompletionRequest to actual CompletionRequest
	converter := func(testReq testutil.CompletionRequest) interface{} {
		return CompletionRequest{
			Prompt:        testReq.Prompt,
			Model:         testReq.Model,
			SystemMessage: testReq.SystemMessage,
			MaxTokens:     testReq.MaxTokens,
			Temperature:   testReq.Temperature,
			TopP:          testReq.TopP,
			StopSequences: testReq.StopSequences,
			Metadata:      testReq.Metadata,
			Timeout:       testReq.Timeout,
		}
	}

	testutil.TestValidateRequestsWithConverterAndValidator(t, tests, converter, validator)
}
