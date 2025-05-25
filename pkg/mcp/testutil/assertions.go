package testutil

import (
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"reflect"
	"testing"
)

// Assertions provides test assertion helpers for MCP testing
type Assertions struct {
	t *testing.T
}

// NewAssertions creates a new assertions helper
func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

// AssertNoError asserts that an error is nil
func (a *Assertions) AssertNoError(err error, msgAndArgs ...interface{}) {
	if err != nil {
		a.t.Helper()
		msg := fmt.Sprintf("Expected no error but got: %v", err)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertError asserts that an error is not nil
func (a *Assertions) AssertError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		a.t.Helper()
		msg := "Expected an error but got nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertEqual asserts that two values are equal
func (a *Assertions) AssertEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		a.t.Helper()
		msg := fmt.Sprintf("Expected %v but got %v", expected, actual)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertNotEqual asserts that two values are not equal
func (a *Assertions) AssertNotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	if reflect.DeepEqual(expected, actual) {
		a.t.Helper()
		msg := fmt.Sprintf("Expected values to be different but both were %v", actual)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertTrue asserts that a value is true
func (a *Assertions) AssertTrue(value bool, msgAndArgs ...interface{}) {
	if !value {
		a.t.Helper()
		msg := "Expected true but got false"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertFalse asserts that a value is false
func (a *Assertions) AssertFalse(value bool, msgAndArgs ...interface{}) {
	if value {
		a.t.Helper()
		msg := "Expected false but got true"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertNil asserts that a value is nil
func (a *Assertions) AssertNil(value interface{}, msgAndArgs ...interface{}) {
	if value != nil && !reflect.ValueOf(value).IsNil() {
		a.t.Helper()
		msg := fmt.Sprintf("Expected nil but got %v", value)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertNotNil asserts that a value is not nil
func (a *Assertions) AssertNotNil(value interface{}, msgAndArgs ...interface{}) {
	if value == nil || reflect.ValueOf(value).IsNil() {
		a.t.Helper()
		msg := "Expected value to not be nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertJSONRPCSuccess asserts that a JSON-RPC response is successful
func (a *Assertions) AssertJSONRPCSuccess(resp *protocol.JSONRPCResponse, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	if resp == nil {
		msg := "Expected response but got nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	if resp.Error != nil {
		msg := fmt.Sprintf("Expected successful response but got error: %s", resp.Error.Message)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	if resp.Result == nil {
		msg := "Expected result in response but got nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertJSONRPCError asserts that a JSON-RPC response has an error
func (a *Assertions) AssertJSONRPCError(resp *protocol.JSONRPCResponse, expectedCode int, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	if resp == nil {
		msg := "Expected response but got nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	if resp.Error == nil {
		msg := "Expected error response but got successful response"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	if resp.Error.Code != expectedCode {
		msg := fmt.Sprintf("Expected error code %d but got %d", expectedCode, resp.Error.Code)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertToolCallSuccess asserts that a tool call result is successful
func (a *Assertions) AssertToolCallSuccess(result *protocol.ToolCallResult, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	if result == nil {
		msg := "Expected tool call result but got nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	if result.IsError {
		msg := "Expected successful tool call but got error"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		if len(result.Content) > 0 {
			msg += ": " + result.Content[0].Text
		}
		a.t.Fatal(msg)
	}
}

// AssertToolCallError asserts that a tool call result is an error
func (a *Assertions) AssertToolCallError(result *protocol.ToolCallResult, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	if result == nil {
		msg := "Expected tool call result but got nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	if !result.IsError {
		msg := "Expected tool call error but got successful result"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
}

// AssertContentEqual asserts that two content slices are equal
func (a *Assertions) AssertContentEqual(expected, actual []protocol.Content, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	if len(expected) != len(actual) {
		msg := fmt.Sprintf("Expected %d content items but got %d", len(expected), len(actual))
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
		}
		a.t.Fatal(msg)
	}
	
	for i := range expected {
		if expected[i].Type != actual[i].Type {
			msg := fmt.Sprintf("Content[%d] type mismatch: expected %s but got %s", i, expected[i].Type, actual[i].Type)
			if len(msgAndArgs) > 0 {
				msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
			}
			a.t.Fatal(msg)
		}
		
		if expected[i].Text != actual[i].Text {
			msg := fmt.Sprintf("Content[%d] text mismatch: expected %s but got %s", i, expected[i].Text, actual[i].Text)
			if len(msgAndArgs) > 0 {
				msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...) + ": " + msg
			}
			a.t.Fatal(msg)
		}
	}
}

// ResponseMatcher provides fluent assertions for JSON-RPC responses
type ResponseMatcher struct {
	a    *Assertions
	resp *protocol.JSONRPCResponse
}

// AssertResponse creates a new response matcher
func (a *Assertions) AssertResponse(resp *protocol.JSONRPCResponse) *ResponseMatcher {
	return &ResponseMatcher{a: a, resp: resp}
}

// IsSuccess asserts the response is successful
func (m *ResponseMatcher) IsSuccess() *ResponseMatcher {
	m.a.AssertJSONRPCSuccess(m.resp)
	return m
}

// IsError asserts the response has an error with the given code
func (m *ResponseMatcher) IsError(code int) *ResponseMatcher {
	m.a.AssertJSONRPCError(m.resp, code)
	return m
}

// HasID asserts the response has the expected ID
func (m *ResponseMatcher) HasID(id interface{}) *ResponseMatcher {
	m.a.AssertEqual(id, m.resp.ID, "Response ID mismatch")
	return m
}

// ResultMatches asserts the result matches the expected value
func (m *ResponseMatcher) ResultMatches(expected interface{}) *ResponseMatcher {
	// Marshal both to JSON for comparison
	expectedJSON, err := json.Marshal(expected)
	m.a.AssertNoError(err, "Failed to marshal expected result")
	
	actualJSON, err := json.Marshal(m.resp.Result)
	m.a.AssertNoError(err, "Failed to marshal actual result")
	
	m.a.AssertEqual(string(expectedJSON), string(actualJSON), "Result mismatch")
	return m
}

// ResultContains checks if the result contains expected fields
func (m *ResponseMatcher) ResultContains(key string, value interface{}) *ResponseMatcher {
	resultMap, ok := m.resp.Result.(map[string]interface{})
	if !ok {
		m.a.t.Fatal("Result is not a map")
	}
	
	actual, exists := resultMap[key]
	if !exists {
		m.a.t.Fatalf("Result does not contain key %s", key)
	}
	
	m.a.AssertEqual(value, actual, "Result field %s mismatch", key)
	return m
}