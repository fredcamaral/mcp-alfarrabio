package testutil

import (
	"context"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"testing"
	"time"
)

// Scenario represents a test scenario
type Scenario struct {
	name        string
	description string
	setup       func(*TestServer) error
	steps       []ScenarioStep
	cleanup     func(*TestServer) error
}

// ScenarioStep represents a single step in a scenario
type ScenarioStep struct {
	name    string
	action  func(*TestServer, *TestClient) error
	verify  func(*TestServer, *TestClient, *Assertions) error
	timeout time.Duration
}

// ScenarioRunner runs test scenarios
type ScenarioRunner struct {
	t         *testing.T
	scenarios []Scenario
}

// NewScenarioRunner creates a new scenario runner
func NewScenarioRunner(t *testing.T) *ScenarioRunner {
	return &ScenarioRunner{
		t:         t,
		scenarios: make([]Scenario, 0),
	}
}

// AddScenario adds a scenario to run
func (r *ScenarioRunner) AddScenario(scenario Scenario) {
	r.scenarios = append(r.scenarios, scenario)
}

// Run executes all scenarios
func (r *ScenarioRunner) Run() {
	for _, scenario := range r.scenarios {
		r.t.Run(scenario.name, func(t *testing.T) {
			r.runScenario(t, scenario)
		})
	}
}

func (r *ScenarioRunner) runScenario(t *testing.T, scenario Scenario) {
	// Create test server
	server := NewTestServer("scenario-server", "1.0.0")
	client := server.Client()
	assertions := NewAssertions(t)
	
	// Setup
	if scenario.setup != nil {
		if err := scenario.setup(server); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}
	
	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()
	
	// Initialize
	ctx := context.Background()
	if _, err := client.Initialize(ctx, "scenario-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Run steps
	for _, step := range scenario.steps {
		t.Run(step.name, func(t *testing.T) {
			// Set timeout
			timeout := step.timeout
			if timeout == 0 {
				timeout = 5 * time.Second
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			
			// Execute action
			if step.action != nil {
				if err := step.action(server, client); err != nil {
					t.Fatalf("Action failed: %v", err)
				}
			}
			
			// Verify
			if step.verify != nil {
				if err := step.verify(server, client, assertions); err != nil {
					t.Fatalf("Verification failed: %v", err)
				}
			}
			
			// Check context
			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					t.Fatal("Step timed out")
				}
			default:
			}
		})
	}
	
	// Cleanup
	if scenario.cleanup != nil {
		if err := scenario.cleanup(server); err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
	}
}

// CommonScenarios provides common test scenarios
var CommonScenarios = struct {
	BasicToolCall         func(toolName string, expectedResult string) Scenario
	ErrorHandling         func() Scenario
	ConcurrentRequests    func(numRequests int) Scenario
	ResourceReadWrite     func() Scenario
	PromptGeneration      func() Scenario
	LargePayloadHandling  func() Scenario
}{
	BasicToolCall: func(toolName string, expectedResult string) Scenario {
		return Scenario{
			name:        "BasicToolCall_" + toolName,
			description: "Tests basic tool calling functionality",
			setup: func(server *TestServer) error {
				server.AddTool(protocol.Tool{
					Name:        toolName,
					Description: "Test tool",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"input": map[string]interface{}{
								"type": "string",
							},
						},
					},
				}, protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					return expectedResult, nil
				}))
				return nil
			},
			steps: []ScenarioStep{
				{
					name: "list_tools",
					action: func(server *TestServer, client *TestClient) error {
						_, err := client.ListTools(context.Background())
						return err
					},
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						tools, err := client.ListTools(context.Background())
						a.AssertNoError(err)
						a.AssertEqual(1, len(tools))
						a.AssertEqual(toolName, tools[0].Name)
						return nil
					},
				},
				{
					name: "call_tool",
					action: func(server *TestServer, client *TestClient) error {
						_, err := client.CallTool(context.Background(), toolName, map[string]interface{}{
							"input": "test",
						})
						return err
					},
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						result, err := client.CallTool(context.Background(), toolName, map[string]interface{}{
							"input": "test",
						})
						a.AssertNoError(err)
						a.AssertToolCallSuccess(result)
						a.AssertEqual(1, len(result.Content))
						a.AssertEqual(expectedResult, result.Content[0].Text)
						return nil
					},
				},
			},
		}
	},
	
	ErrorHandling: func() Scenario {
		return Scenario{
			name:        "ErrorHandling",
			description: "Tests various error conditions",
			steps: []ScenarioStep{
				{
					name: "call_nonexistent_tool",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("tools/call", protocol.ToolCallRequest{
							Name: "nonexistent",
						})
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCError(resp, protocol.MethodNotFound)
						return nil
					},
				},
				{
					name: "invalid_method",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("invalid/method", nil)
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCError(resp, protocol.MethodNotFound)
						return nil
					},
				},
				{
					name: "malformed_params",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("tools/call", "invalid params")
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCError(resp, protocol.InvalidParams)
						return nil
					},
				},
			},
		}
	},
	
	ConcurrentRequests: func(numRequests int) Scenario {
		return Scenario{
			name:        fmt.Sprintf("ConcurrentRequests_%d", numRequests),
			description: "Tests handling of concurrent requests",
			setup: func(server *TestServer) error {
				// Add a simple echo tool
				server.AddTool(protocol.Tool{
					Name: "echo",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"message": map[string]interface{}{"type": "string"},
						},
					},
				}, protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					msg, _ := params["message"].(string)
					return msg, nil
				}))
				return nil
			},
			steps: []ScenarioStep{
				{
					name: "send_concurrent_requests",
					action: func(server *TestServer, client *TestClient) error {
						// Send multiple requests concurrently
						errCh := make(chan error, numRequests)
						
						for i := 0; i < numRequests; i++ {
							go func(idx int) {
								_, err := client.CallTool(context.Background(), "echo", map[string]interface{}{
									"message": fmt.Sprintf("request_%d", idx),
								})
								errCh <- err
							}(i)
						}
						
						// Wait for all requests
						for i := 0; i < numRequests; i++ {
							if err := <-errCh; err != nil {
								return err
							}
						}
						
						return nil
					},
					timeout: 10 * time.Second,
				},
			},
		}
	},
	
	ResourceReadWrite: func() Scenario {
		return Scenario{
			name:        "ResourceReadWrite",
			description: "Tests resource reading functionality",
			setup: func(server *TestServer) error {
				// Add test resources
				server.AddResource(protocol.Resource{
					URI:         "file:///test.txt",
					Name:        "test.txt",
					Description: "Test file",
					MimeType:    "text/plain",
				}, ResourceHandlerFunc(func(ctx context.Context, uri string) ([]protocol.Content, error) {
					return []protocol.Content{
						protocol.NewContent("Test file contents"),
					}, nil
				}))
				
				server.AddResource(protocol.Resource{
					URI:      "https://example.com/data.json",
					Name:     "data.json",
					MimeType: "application/json",
				}, ResourceHandlerFunc(func(ctx context.Context, uri string) ([]protocol.Content, error) {
					return []protocol.Content{
						protocol.NewContent(`{"key": "value"}`),
					}, nil
				}))
				
				return nil
			},
			steps: []ScenarioStep{
				{
					name: "list_resources",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("resources/list", nil)
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCSuccess(resp)
						
						// TODO: Add resource list parsing and verification
						return nil
					},
				},
				{
					name: "read_resource",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("resources/read", map[string]interface{}{
							"uri": "file:///test.txt",
						})
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCSuccess(resp)
						
						// TODO: Add content verification
						return nil
					},
				},
			},
		}
	},
	
	PromptGeneration: func() Scenario {
		return Scenario{
			name:        "PromptGeneration",
			description: "Tests prompt generation functionality",
			setup: func(server *TestServer) error {
				// Add test prompts
				server.AddPrompt(protocol.Prompt{
					Name:        "greeting",
					Description: "Generates a greeting",
					Arguments: []protocol.PromptArgument{
						{
							Name:        "name",
							Description: "Name to greet",
							Required:    true,
						},
						{
							Name:        "style",
							Description: "Greeting style",
							Required:    false,
						},
					},
				}, PromptHandlerFunc(func(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error) {
					name, _ := args["name"].(string)
					style, _ := args["style"].(string)
					
					greeting := "Hello"
					if style == "formal" {
						greeting = "Greetings"
					}
					
					return []protocol.Content{
						protocol.NewContent(fmt.Sprintf("%s, %s!", greeting, name)),
					}, nil
				}))
				
				return nil
			},
			steps: []ScenarioStep{
				{
					name: "list_prompts",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("prompts/list", nil)
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCSuccess(resp)
						
						return nil
					},
				},
				{
					name: "get_prompt",
					verify: func(server *TestServer, client *TestClient, a *Assertions) error {
						ctx := context.Background()
						id, err := client.SendRequest("prompts/get", map[string]interface{}{
							"name": "greeting",
							"arguments": map[string]interface{}{
								"name":  "World",
								"style": "formal",
							},
						})
						a.AssertNoError(err)
						
						resp, err := client.WaitForResponse(ctx, id)
						a.AssertNoError(err)
						a.AssertJSONRPCSuccess(resp)
						
						return nil
					},
				},
			},
		}
	},
	
	LargePayloadHandling: func() Scenario {
		return Scenario{
			name:        "LargePayloadHandling",
			description: "Tests handling of large payloads",
			setup: func(server *TestServer) error {
				// Add tool that handles large data
				server.AddTool(protocol.Tool{
					Name: "process_data",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"data": map[string]interface{}{"type": "string"},
						},
					},
				}, protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					data, _ := params["data"].(string)
					return fmt.Sprintf("Processed %d bytes", len(data)), nil
				}))
				
				return nil
			},
			steps: []ScenarioStep{
				{
					name: "send_large_payload",
					action: func(server *TestServer, client *TestClient) error {
						// Create large data (1MB)
						largeData := make([]byte, 1024*1024)
						for i := range largeData {
							largeData[i] = byte(i % 256)
						}
						
						_, err := client.CallTool(context.Background(), "process_data", map[string]interface{}{
							"data": string(largeData),
						})
						return err
					},
					timeout: 30 * time.Second,
				},
			},
		}
	},
}