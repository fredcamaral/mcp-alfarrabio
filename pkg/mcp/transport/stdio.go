// Package transport implements MCP transport layers
package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mcp-memory/pkg/mcp/protocol"
	"os"
	"sync"
)

// StdioTransport implements the Transport interface using stdio
type StdioTransport struct {
	input   io.Reader
	output  io.Writer
	scanner *bufio.Scanner
	encoder *json.Encoder
	mutex   sync.Mutex
	running bool
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		input:   os.Stdin,
		output:  os.Stdout,
		scanner: bufio.NewScanner(os.Stdin),
		encoder: json.NewEncoder(os.Stdout),
	}
}

// NewStdioTransportWithIO creates a new stdio transport with custom IO
func NewStdioTransportWithIO(input io.Reader, output io.Writer) *StdioTransport {
	return &StdioTransport{
		input:   input,
		output:  output,
		scanner: bufio.NewScanner(input),
		encoder: json.NewEncoder(output),
	}
}

// Start starts the stdio transport
func (t *StdioTransport) Start(ctx context.Context, handler RequestHandler) error {
	t.mutex.Lock()
	if t.running {
		t.mutex.Unlock()
		return fmt.Errorf("transport already running")
	}
	t.running = true
	t.mutex.Unlock()

	defer func() {
		t.mutex.Lock()
		t.running = false
		t.mutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !t.scanner.Scan() {
				if err := t.scanner.Err(); err != nil {
					return fmt.Errorf("scanning input: %w", err)
				}
				// EOF reached
				return nil
			}

			line := t.scanner.Text()
			if line == "" {
				continue
			}

			var req protocol.JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				// Send error response
				errResp := &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					Error:   protocol.NewJSONRPCError(protocol.ParseError, "Parse error", err.Error()),
				}
				if err := t.sendResponse(errResp); err != nil {
					return fmt.Errorf("failed to send error response: %w", err)
				}
				continue
			}

			// Handle the request
			resp := handler.HandleRequest(ctx, &req)
			if resp != nil {
				if err := t.sendResponse(resp); err != nil {
					return fmt.Errorf("sending response: %w", err)
				}
			}
		}
	}
}

// Stop stops the stdio transport
func (t *StdioTransport) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.running = false
	return nil
}

// sendResponse sends a JSON-RPC response
func (t *StdioTransport) sendResponse(resp *protocol.JSONRPCResponse) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if err := t.encoder.Encode(resp); err != nil {
		return fmt.Errorf("encoding response: %w", err)
	}

	return nil
}

// IsRunning returns whether the transport is running
func (t *StdioTransport) IsRunning() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.running
}