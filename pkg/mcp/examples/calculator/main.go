// Calculator MCP Server Example
//
// This example demonstrates a simple calculator MCP server with basic
// arithmetic operations. It shows:
//   - Tool registration with JSON Schema validation
//   - Parameter handling and type conversion
//   - Error handling for edge cases
//   - Clean server setup and lifecycle
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/mcp-go"
	"github.com/yourusername/mcp-go/transport"
)

func main() {
	// Create a new MCP server
	server := mcp.NewServer("calculator", "1.0.0")
	server.SetDescription("A simple calculator MCP server")

	// Register arithmetic operation tools
	registerAddTool(server)
	registerSubtractTool(server)
	registerMultiplyTool(server)
	registerDivideTool(server)
	registerPowerTool(server)
	registerSqrtTool(server)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down calculator server...")
		cancel()
	}()

	// Start the server with stdio transport
	log.Println("Calculator MCP server starting...")
	if err := server.Start(ctx, transport.Stdio()); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// registerAddTool registers the addition tool
func registerAddTool(server *mcp.Server) {
	tool := mcp.NewTool(
		"add",
		"Add two numbers",
		mcp.ObjectSchema("Addition parameters", map[string]interface{}{
			"a": mcp.NumberParam("First number", true),
			"b": mcp.NumberParam("Second number", true),
		}, []string{"a", "b"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		a, ok := getNumber(params, "a")
		if !ok {
			return nil, fmt.Errorf("parameter 'a' must be a number")
		}

		b, ok := getNumber(params, "b")
		if !ok {
			return nil, fmt.Errorf("parameter 'b' must be a number")
		}

		result := a + b
		return map[string]interface{}{
			"result": result,
			"operation": fmt.Sprintf("%v + %v = %v", a, b, result),
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerSubtractTool registers the subtraction tool
func registerSubtractTool(server *mcp.Server) {
	tool := mcp.NewTool(
		"subtract",
		"Subtract two numbers",
		mcp.ObjectSchema("Subtraction parameters", map[string]interface{}{
			"a": mcp.NumberParam("First number", true),
			"b": mcp.NumberParam("Second number", true),
		}, []string{"a", "b"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		a, ok := getNumber(params, "a")
		if !ok {
			return nil, fmt.Errorf("parameter 'a' must be a number")
		}

		b, ok := getNumber(params, "b")
		if !ok {
			return nil, fmt.Errorf("parameter 'b' must be a number")
		}

		result := a - b
		return map[string]interface{}{
			"result": result,
			"operation": fmt.Sprintf("%v - %v = %v", a, b, result),
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerMultiplyTool registers the multiplication tool
func registerMultiplyTool(server *mcp.Server) {
	tool := mcp.NewTool(
		"multiply",
		"Multiply two numbers",
		mcp.ObjectSchema("Multiplication parameters", map[string]interface{}{
			"a": mcp.NumberParam("First number", true),
			"b": mcp.NumberParam("Second number", true),
		}, []string{"a", "b"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		a, ok := getNumber(params, "a")
		if !ok {
			return nil, fmt.Errorf("parameter 'a' must be a number")
		}

		b, ok := getNumber(params, "b")
		if !ok {
			return nil, fmt.Errorf("parameter 'b' must be a number")
		}

		result := a * b
		return map[string]interface{}{
			"result": result,
			"operation": fmt.Sprintf("%v × %v = %v", a, b, result),
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerDivideTool registers the division tool with error handling
func registerDivideTool(server *mcp.Server) {
	tool := mcp.NewTool(
		"divide",
		"Divide two numbers",
		mcp.ObjectSchema("Division parameters", map[string]interface{}{
			"a": mcp.NumberParam("Dividend", true),
			"b": mcp.NumberParam("Divisor (cannot be zero)", true),
		}, []string{"a", "b"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		a, ok := getNumber(params, "a")
		if !ok {
			return nil, fmt.Errorf("parameter 'a' must be a number")
		}

		b, ok := getNumber(params, "b")
		if !ok {
			return nil, fmt.Errorf("parameter 'b' must be a number")
		}

		// Check for division by zero
		if b == 0 {
			return nil, fmt.Errorf("division by zero is not allowed")
		}

		result := a / b
		return map[string]interface{}{
			"result": result,
			"operation": fmt.Sprintf("%v ÷ %v = %v", a, b, result),
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerPowerTool registers the exponentiation tool
func registerPowerTool(server *mcp.Server) {
	tool := mcp.NewTool(
		"power",
		"Raise a number to a power",
		mcp.ObjectSchema("Power parameters", map[string]interface{}{
			"base":     mcp.NumberParam("Base number", true),
			"exponent": mcp.NumberParam("Exponent", true),
		}, []string{"base", "exponent"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		base, ok := getNumber(params, "base")
		if !ok {
			return nil, fmt.Errorf("parameter 'base' must be a number")
		}

		exponent, ok := getNumber(params, "exponent")
		if !ok {
			return nil, fmt.Errorf("parameter 'exponent' must be a number")
		}

		result := math.Pow(base, exponent)

		// Check for special cases
		if math.IsInf(result, 0) {
			return nil, fmt.Errorf("result is too large (infinity)")
		}
		if math.IsNaN(result) {
			return nil, fmt.Errorf("result is not a number (NaN)")
		}

		return map[string]interface{}{
			"result": result,
			"operation": fmt.Sprintf("%v^%v = %v", base, exponent, result),
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerSqrtTool registers the square root tool
func registerSqrtTool(server *mcp.Server) {
	tool := mcp.NewTool(
		"sqrt",
		"Calculate the square root of a number",
		mcp.ObjectSchema("Square root parameters", map[string]interface{}{
			"number": mcp.NumberParam("Number to find square root of (must be non-negative)", true),
		}, []string{"number"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		number, ok := getNumber(params, "number")
		if !ok {
			return nil, fmt.Errorf("parameter 'number' must be a number")
		}

		// Check for negative numbers
		if number < 0 {
			return nil, fmt.Errorf("cannot calculate square root of negative number: %v", number)
		}

		result := math.Sqrt(number)
		return map[string]interface{}{
			"result": result,
			"operation": fmt.Sprintf("√%v = %v", number, result),
		}, nil
	})

	server.AddTool(tool, handler)
}

// Helper function to safely extract numbers from parameters
func getNumber(params map[string]interface{}, key string) (float64, bool) {
	val, exists := params[key]
	if !exists {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}