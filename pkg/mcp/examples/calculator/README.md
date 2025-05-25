# Calculator MCP Server Example

A simple calculator MCP server that demonstrates basic tool implementation with the MCP-Go library.

## Features

This example includes the following arithmetic operations:

- **add** - Add two numbers
- **subtract** - Subtract two numbers  
- **multiply** - Multiply two numbers
- **divide** - Divide two numbers (with zero-check)
- **power** - Raise a number to a power
- **sqrt** - Calculate square root (with negative-check)

## What This Example Demonstrates

1. **Tool Registration**: How to register multiple tools with a server
2. **JSON Schema Validation**: Using schema builders for parameter validation
3. **Error Handling**: Proper error handling for edge cases (division by zero, negative square roots)
4. **Type Safety**: Safe parameter extraction with type checking
5. **Graceful Shutdown**: Handling system signals for clean shutdown

## Running the Example

### Build and Run

```bash
go build -o calculator
./calculator
```

### Test with Claude Desktop

1. Add to your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "calculator": {
      "command": "/path/to/calculator"
    }
  }
}
```

2. Restart Claude Desktop

3. Test the calculator:
   - "Use the calculator to add 42 and 58"
   - "What's the square root of 144?"
   - "Calculate 2 to the power of 10"

## Code Structure

```
calculator/
├── README.md       # This file
├── main.go         # Server implementation
└── go.mod         # Module definition
```

## Key Code Patterns

### Tool Registration Pattern

```go
tool := mcp.NewTool(
    "add",
    "Add two numbers",
    mcp.ObjectSchema("Addition parameters", map[string]interface{}{
        "a": mcp.NumberParam("First number", true),
        "b": mcp.NumberParam("Second number", true),
    }, []string{"a", "b"}),
)

handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Implementation
})

server.AddTool(tool, handler)
```

### Safe Parameter Extraction

```go
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
    // ... handle other numeric types
    }
}
```

### Error Handling

```go
// Check for division by zero
if b == 0 {
    return nil, fmt.Errorf("division by zero is not allowed")
}

// Check for negative square root
if number < 0 {
    return nil, fmt.Errorf("cannot calculate square root of negative number: %v", number)
}
```

## Extending the Example

Ideas for extending this calculator:

1. **Advanced Operations**: Add trigonometric functions, logarithms
2. **Memory Functions**: Store/recall previous results
3. **Expression Parsing**: Parse and evaluate complex expressions
4. **Unit Conversion**: Add tools for converting between units
5. **Statistics**: Add mean, median, standard deviation tools

## Troubleshooting

### Common Issues

1. **Tool not found**: Ensure the tool name matches exactly
2. **Parameter errors**: Check that all required parameters are provided
3. **Type errors**: Ensure numeric parameters are passed as numbers, not strings

### Debug Mode

Run with debug logging:
```bash
LOG_LEVEL=debug ./calculator
```

## Learning Points

This example teaches:

- How to structure an MCP server application
- Best practices for parameter validation
- Error handling strategies
- Tool naming conventions
- Response formatting

Next, try the [File Browser](../file-browser/) example to learn about resources and more complex operations!