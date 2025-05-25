package testutil

import (
	"context"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"
	"sync"
	"time"
)

// TestServer wraps an MCP server for testing
type TestServer struct {
	*server.Server
	transport *transport.StdioTransport
	client    *TestClient
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewTestServer creates a new test server
func NewTestServer(name, version string) *TestServer {
	srv := server.NewServer(name, version)
	client := NewTestClient()
	
	// Create transport with test client's IO
	trans := transport.NewStdioTransportWithIO(client.GetInput(), client.GetOutput())
	srv.SetTransport(trans)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TestServer{
		Server:    srv,
		transport: trans,
		client:    client,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the test server
func (ts *TestServer) Start() error {
	// Start reading responses in background
	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()
		ts.client.ReadResponses(ts.ctx)
	}()
	
	// Start server in background
	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()
		ts.Server.Start(ts.ctx)
	}()
	
	// Give server time to start
	time.Sleep(10 * time.Millisecond)
	
	return nil
}

// Stop stops the test server
func (ts *TestServer) Stop() error {
	ts.cancel()
	ts.wg.Wait()
	return ts.client.Close()
}

// Client returns the test client
func (ts *TestServer) Client() *TestClient {
	return ts.client
}

// ServerBuilder provides a fluent API for building test servers
type ServerBuilder struct {
	server          *TestServer
	tools           []toolConfig
	resources       []resourceConfig
	prompts         []promptConfig
	autoStart       bool
	initialized     bool
	clientName      string
	clientVersion   string
}

type toolConfig struct {
	tool    protocol.Tool
	handler protocol.ToolHandler
}

type resourceConfig struct {
	resource protocol.Resource
	handler  server.ResourceHandler
}

type promptConfig struct {
	prompt  protocol.Prompt
	handler server.PromptHandler
}

// NewServerBuilder creates a new server builder
func NewServerBuilder(name, version string) *ServerBuilder {
	return &ServerBuilder{
		server: NewTestServer(name, version),
	}
}

// WithTool adds a tool to the server
func (b *ServerBuilder) WithTool(name, description string, handler protocol.ToolHandler) *ServerBuilder {
	b.tools = append(b.tools, toolConfig{
		tool: protocol.Tool{
			Name:        name,
			Description: description,
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		handler: handler,
	})
	return b
}

// WithSimpleTool adds a simple tool that returns a string
func (b *ServerBuilder) WithSimpleTool(name string, result string) *ServerBuilder {
	handler := protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return result, nil
	})
	return b.WithTool(name, "Test tool: "+name, handler)
}

// WithResource adds a resource to the server
func (b *ServerBuilder) WithResource(uri, name string, content string) *ServerBuilder {
	handler := ResourceHandlerFunc(func(ctx context.Context, resourceURI string) ([]protocol.Content, error) {
		return []protocol.Content{protocol.NewContent(content)}, nil
	})
	
	b.resources = append(b.resources, resourceConfig{
		resource: protocol.Resource{
			URI:      uri,
			Name:     name,
			MimeType: "text/plain",
		},
		handler: handler,
	})
	return b
}

// WithPrompt adds a prompt to the server
func (b *ServerBuilder) WithPrompt(name string, content string) *ServerBuilder {
	handler := PromptHandlerFunc(func(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error) {
		return []protocol.Content{protocol.NewContent(content)}, nil
	})
	
	b.prompts = append(b.prompts, promptConfig{
		prompt: protocol.Prompt{
			Name:        name,
			Description: "Test prompt: " + name,
		},
		handler: handler,
	})
	return b
}

// WithAutoStart configures the server to start automatically
func (b *ServerBuilder) WithAutoStart() *ServerBuilder {
	b.autoStart = true
	return b
}

// WithAutoInitialize configures the server to initialize automatically
func (b *ServerBuilder) WithAutoInitialize(clientName, clientVersion string) *ServerBuilder {
	b.initialized = true
	b.autoStart = true
	b.clientName = clientName
	b.clientVersion = clientVersion
	return b
}

// Build builds and returns the configured test server
func (b *ServerBuilder) Build() *TestServer {
	// Add all configured tools
	for _, tc := range b.tools {
		b.server.AddTool(tc.tool, tc.handler)
	}
	
	// Add all configured resources
	for _, rc := range b.resources {
		b.server.AddResource(rc.resource, rc.handler)
	}
	
	// Add all configured prompts
	for _, pc := range b.prompts {
		b.server.AddPrompt(pc.prompt, pc.handler)
	}
	
	// Start if configured
	if b.autoStart {
		if err := b.server.Start(); err != nil {
			panic(err)
		}
	}
	
	// Initialize if configured
	if b.initialized && b.clientName != "" {
		go func() {
			time.Sleep(20 * time.Millisecond) // Give server time to start
			b.server.Client().Initialize(context.Background(), b.clientName, b.clientVersion)
		}()
	}
	
	return b.server
}

// Handler function types
type ResourceHandlerFunc func(ctx context.Context, uri string) ([]protocol.Content, error)

func (f ResourceHandlerFunc) Handle(ctx context.Context, uri string) ([]protocol.Content, error) {
	return f(ctx, uri)
}

type PromptHandlerFunc func(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error)

func (f PromptHandlerFunc) Handle(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error) {
	return f(ctx, args)
}