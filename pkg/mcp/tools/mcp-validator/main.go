package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"mcp-memory/pkg/mcp/protocol"
)

const version = "1.0.0"

type Validator struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	dec    *json.Decoder
	enc    *json.Encoder
}

func main() {
	var (
		serverPath = flag.String("server", "", "Path to MCP server executable")
		timeout    = flag.Duration("timeout", 30*time.Second, "Validation timeout")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
		showVersion = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("MCP Validator v%s\n", version)
		os.Exit(0)
	}

	if *serverPath == "" {
		if flag.NArg() > 0 {
			*serverPath = flag.Arg(0)
		} else {
			fmt.Fprintf(os.Stderr, "Error: server path required\n")
			flag.Usage()
			os.Exit(1)
		}
	}

	validator, err := NewValidator(*serverPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating validator: %v\n", err)
		os.Exit(1)
	}
	defer validator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	results, err := validator.Validate(ctx, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
		os.Exit(1)
	}

	printResults(results)

	if !results.Valid {
		os.Exit(1)
	}
}

func NewValidator(serverPath string) (*Validator, error) {
	cmd := exec.Command(serverPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	return &Validator{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		dec:    json.NewDecoder(stdout),
		enc:    json.NewEncoder(stdin),
	}, nil
}

type ValidationResults struct {
	Valid        bool
	ProtocolInfo protocol.ServerInfo
	Capabilities protocol.ServerCapabilities
	Tests        []TestResult
}

type TestResult struct {
	Name    string
	Passed  bool
	Message string
	Duration time.Duration
}

func (v *Validator) Validate(ctx context.Context, verbose bool) (*ValidationResults, error) {
	results := &ValidationResults{
		Valid: true,
		Tests: []TestResult{},
	}

	// Test 1: Initialize
	initResult := v.testInitialize(ctx, verbose)
	if !initResult.Passed {
		results.Valid = false
	} else {
		results.ProtocolInfo = initResult.ServerInfo
		results.Capabilities = initResult.Capabilities
	}
	results.Tests = append(results.Tests, initResult.TestResult)

	// Test 2: List Tools
	if results.Capabilities.Tools != nil {
		toolsResult := v.testListTools(ctx, verbose)
		if !toolsResult.Passed {
			results.Valid = false
		}
		results.Tests = append(results.Tests, toolsResult)
	}

	// Test 3: List Resources
	if results.Capabilities.Resources != nil {
		resourcesResult := v.testListResources(ctx, verbose)
		if !resourcesResult.Passed {
			results.Valid = false
		}
		results.Tests = append(results.Tests, resourcesResult)
	}

	// Test 4: List Prompts
	if results.Capabilities.Prompts != nil {
		promptsResult := v.testListPrompts(ctx, verbose)
		if !promptsResult.Passed {
			results.Valid = false
		}
		results.Tests = append(results.Tests, promptsResult)
	}

	return results, nil
}

type InitializeResult struct {
	TestResult
	ServerInfo   protocol.ServerInfo
	Capabilities protocol.ServerCapabilities
}

func (v *Validator) testInitialize(ctx context.Context, verbose bool) *InitializeResult {
	start := time.Now()
	
	// Send initialize request
	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "1.0",
			"capabilities": {},
			"clientInfo": {
				"name": "mcp-validator",
				"version": "` + version + `"
			}
		}`),
	}

	if verbose {
		fmt.Printf("Sending initialize request...\n")
	}

	if err := v.enc.Encode(req); err != nil {
		return &InitializeResult{
			TestResult: TestResult{
				Name:     "Initialize",
				Passed:   false,
				Message:  fmt.Sprintf("Failed to send request: %v", err),
				Duration: time.Since(start),
			},
		}
	}

	// Read response
	var resp protocol.JSONRPCResponse
	if err := v.dec.Decode(&resp); err != nil {
		return &InitializeResult{
			TestResult: TestResult{
				Name:     "Initialize",
				Passed:   false,
				Message:  fmt.Sprintf("Failed to read response: %v", err),
				Duration: time.Since(start),
			},
		}
	}

	if resp.Error != nil {
		return &InitializeResult{
			TestResult: TestResult{
				Name:     "Initialize",
				Passed:   false,
				Message:  fmt.Sprintf("Server error: %s", resp.Error.Message),
				Duration: time.Since(start),
			},
		}
	}

	// Parse result
	var initResult protocol.InitializeResult
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return &InitializeResult{
			TestResult: TestResult{
				Name:     "Initialize",
				Passed:   false,
				Message:  fmt.Sprintf("Failed to marshal result: %v", err),
				Duration: time.Since(start),
			},
		}
	}
	if err := json.Unmarshal(resultBytes, &initResult); err != nil {
		return &InitializeResult{
			TestResult: TestResult{
				Name:     "Initialize",
				Passed:   false,
				Message:  fmt.Sprintf("Failed to parse result: %v", err),
				Duration: time.Since(start),
			},
		}
	}

	// Send initialized notification
	notif := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := v.enc.Encode(notif); err != nil {
		return &InitializeResult{
			TestResult: TestResult{
				Name:     "Initialize",
				Passed:   false,
				Message:  fmt.Sprintf("Failed to send initialized notification: %v", err),
				Duration: time.Since(start),
			},
		}
	}

	return &InitializeResult{
		TestResult: TestResult{
			Name:     "Initialize",
			Passed:   true,
			Message:  "Successfully initialized",
			Duration: time.Since(start),
		},
		ServerInfo:   initResult.ServerInfo,
		Capabilities: initResult.Capabilities,
	}
}

func (v *Validator) testListTools(ctx context.Context, verbose bool) TestResult {
	start := time.Now()

	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}

	if verbose {
		fmt.Printf("Sending tools/list request...\n")
	}

	if err := v.enc.Encode(req); err != nil {
		return TestResult{
			Name:     "List Tools",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to send request: %v", err),
			Duration: time.Since(start),
		}
	}

	var resp protocol.JSONRPCResponse
	if err := v.dec.Decode(&resp); err != nil {
		return TestResult{
			Name:     "List Tools",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to read response: %v", err),
			Duration: time.Since(start),
		}
	}

	if resp.Error != nil {
		return TestResult{
			Name:     "List Tools",
			Passed:   false,
			Message:  fmt.Sprintf("Server error: %s", resp.Error.Message),
			Duration: time.Since(start),
		}
	}

	var tools []protocol.Tool
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return TestResult{
			Name:     "List Tools",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to marshal result: %v", err),
			Duration: time.Since(start),
		}
	}
	if err := json.Unmarshal(resultBytes, &tools); err != nil {
		return TestResult{
			Name:     "List Tools",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to parse result: %v", err),
			Duration: time.Since(start),
		}
	}

	return TestResult{
		Name:     "List Tools",
		Passed:   true,
		Message:  fmt.Sprintf("Found %d tools", len(tools)),
		Duration: time.Since(start),
	}
}

func (v *Validator) testListResources(ctx context.Context, verbose bool) TestResult {
	start := time.Now()

	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "resources/list",
	}

	if verbose {
		fmt.Printf("Sending resources/list request...\n")
	}

	if err := v.enc.Encode(req); err != nil {
		return TestResult{
			Name:     "List Resources",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to send request: %v", err),
			Duration: time.Since(start),
		}
	}

	var resp protocol.JSONRPCResponse
	if err := v.dec.Decode(&resp); err != nil {
		return TestResult{
			Name:     "List Resources",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to read response: %v", err),
			Duration: time.Since(start),
		}
	}

	if resp.Error != nil {
		return TestResult{
			Name:     "List Resources",
			Passed:   false,
			Message:  fmt.Sprintf("Server error: %s", resp.Error.Message),
			Duration: time.Since(start),
		}
	}

	var resources []protocol.Resource
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return TestResult{
			Name:     "List Resources",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to marshal result: %v", err),
			Duration: time.Since(start),
		}
	}
	if err := json.Unmarshal(resultBytes, &resources); err != nil {
		return TestResult{
			Name:     "List Resources",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to parse result: %v", err),
			Duration: time.Since(start),
		}
	}

	return TestResult{
		Name:     "List Resources",
		Passed:   true,
		Message:  fmt.Sprintf("Found %d resources", len(resources)),
		Duration: time.Since(start),
	}
}

func (v *Validator) testListPrompts(ctx context.Context, verbose bool) TestResult {
	start := time.Now()

	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "prompts/list",
	}

	if verbose {
		fmt.Printf("Sending prompts/list request...\n")
	}

	if err := v.enc.Encode(req); err != nil {
		return TestResult{
			Name:     "List Prompts",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to send request: %v", err),
			Duration: time.Since(start),
		}
	}

	var resp protocol.JSONRPCResponse
	if err := v.dec.Decode(&resp); err != nil {
		return TestResult{
			Name:     "List Prompts",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to read response: %v", err),
			Duration: time.Since(start),
		}
	}

	if resp.Error != nil {
		return TestResult{
			Name:     "List Prompts",
			Passed:   false,
			Message:  fmt.Sprintf("Server error: %s", resp.Error.Message),
			Duration: time.Since(start),
		}
	}

	var prompts []protocol.Prompt
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return TestResult{
			Name:     "List Prompts",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to marshal result: %v", err),
			Duration: time.Since(start),
		}
	}
	if err := json.Unmarshal(resultBytes, &prompts); err != nil {
		return TestResult{
			Name:     "List Prompts",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to parse result: %v", err),
			Duration: time.Since(start),
		}
	}

	return TestResult{
		Name:     "List Prompts",
		Passed:   true,
		Message:  fmt.Sprintf("Found %d prompts", len(prompts)),
		Duration: time.Since(start),
	}
}

func (v *Validator) Close() error {
	if v.stdin != nil {
		v.stdin.Close()
	}
	if v.cmd != nil {
		v.cmd.Process.Kill()
		v.cmd.Wait()
	}
	return nil
}

func printResults(results *ValidationResults) {
	fmt.Printf("\nMCP Server Validation Results\n")
	fmt.Printf("==============================\n\n")

	if results.Valid {
		fmt.Printf("Status: ✅ VALID\n\n")
	} else {
		fmt.Printf("Status: ❌ INVALID\n\n")
	}

	fmt.Printf("Server Info:\n")
	fmt.Printf("  Name: %s\n", results.ProtocolInfo.Name)
	fmt.Printf("  Version: %s\n\n", results.ProtocolInfo.Version)

	fmt.Printf("Capabilities:\n")
	if results.Capabilities.Tools != nil {
		fmt.Printf("  ✅ Tools\n")
	} else {
		fmt.Printf("  ❌ Tools\n")
	}
	if results.Capabilities.Resources != nil {
		fmt.Printf("  ✅ Resources\n")
	} else {
		fmt.Printf("  ❌ Resources\n")
	}
	if results.Capabilities.Prompts != nil {
		fmt.Printf("  ✅ Prompts\n")
	} else {
		fmt.Printf("  ❌ Prompts\n")
	}

	fmt.Printf("\nTest Results:\n")
	fmt.Printf("%-20s %-10s %-40s %s\n", "Test", "Status", "Message", "Duration")
	fmt.Printf("%s\n", strings.Repeat("-", 80))

	for _, test := range results.Tests {
		status := "✅ PASS"
		if !test.Passed {
			status = "❌ FAIL"
		}
		fmt.Printf("%-20s %-10s %-40s %v\n", test.Name, status, test.Message, test.Duration)
	}
}