# Sampling Feature Documentation

## Overview

The Sampling feature enables MCP servers to integrate with Large Language Models (LLMs) to provide AI-powered responses. This feature is particularly useful for clients that support advanced AI interactions.

## Implementation

### Basic Usage

```go
import (
    "github.com/yourusername/mcp-go/sampling"
    "github.com/yourusername/mcp-go/server"
)

// Create server with sampling support
srv := server.NewExtendedServer("AI Server", "1.0.0")

// Use default sampling handler (mock implementation)
// In production, replace with actual LLM integration
```

### Custom Sampling Handler

```go
type MyLLMHandler struct {
    apiKey string
    model  string
}

func (h *MyLLMHandler) CreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req sampling.CreateMessageRequest
    if err := json.Unmarshal(params, &req); err != nil {
        return nil, err
    }
    
    // Call your LLM API (OpenAI, Anthropic, etc.)
    response := callLLMAPI(h.apiKey, h.model, req.Messages, req.MaxTokens)
    
    return sampling.CreateMessageResponse{
        Role:    "assistant",
        Content: sampling.SamplingMessageContent{
            Type: "text",
            Text: response.Text,
        },
        Model:      h.model,
        StopReason: response.StopReason,
    }, nil
}

// Set custom handler
srv.SetSamplingHandler(&MyLLMHandler{
    apiKey: os.Getenv("LLM_API_KEY"),
    model:  "gpt-4",
})
```

## Request Format

```json
{
    "method": "sampling/createMessage",
    "params": {
        "messages": [
            {
                "role": "user",
                "content": {
                    "type": "text",
                    "text": "What is the Model Context Protocol?"
                }
            }
        ],
        "modelPreferences": {
            "hints": [{"name": "claude-3-opus"}],
            "intelligencePriority": 0.8,
            "speedPriority": 0.2
        },
        "maxTokens": 1000,
        "temperature": 0.7,
        "systemPrompt": "You are a helpful assistant."
    }
}
```

## Response Format

```json
{
    "role": "assistant",
    "content": {
        "type": "text",
        "text": "The Model Context Protocol (MCP) is..."
    },
    "model": "claude-3-opus",
    "stopReason": "stop_sequence"
}
```

## Model Preferences

The `modelPreferences` field allows clients to specify preferences:

- **hints**: Suggested models to use
- **intelligencePriority**: 0-1, higher = smarter model
- **speedPriority**: 0-1, higher = faster response
- **costPriority**: 0-1, higher = cheaper model

## Integration Examples

### OpenAI Integration

```go
func (h *OpenAIHandler) CreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
    client := openai.NewClient(h.apiKey)
    
    // Convert MCP messages to OpenAI format
    messages := convertToOpenAIMessages(req.Messages)
    
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       openai.GPT4,
        Messages:    messages,
        MaxTokens:   req.MaxTokens,
        Temperature: req.Temperature,
    })
    
    if err != nil {
        return nil, err
    }
    
    return sampling.CreateMessageResponse{
        Role: "assistant",
        Content: sampling.SamplingMessageContent{
            Type: "text",
            Text: resp.Choices[0].Message.Content,
        },
        Model:      string(resp.Model),
        StopReason: string(resp.Choices[0].FinishReason),
    }, nil
}
```

### Anthropic Integration

```go
func (h *AnthropicHandler) CreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
    client := anthropic.NewClient(h.apiKey)
    
    resp, err := client.Messages.Create(ctx, &anthropic.MessageRequest{
        Model:     anthropic.Claude3Opus,
        Messages:  convertToAnthropicMessages(req.Messages),
        MaxTokens: req.MaxTokens,
        System:    req.SystemPrompt,
    })
    
    if err != nil {
        return nil, err
    }
    
    return sampling.CreateMessageResponse{
        Role: "assistant",
        Content: sampling.SamplingMessageContent{
            Type: "text",
            Text: resp.Content[0].Text,
        },
        Model:      resp.Model,
        StopReason: resp.StopReason,
    }, nil
}
```

## Client Support

| Client | Sampling Support | Notes |
|--------|-----------------|-------|
| fast-agent | ✅ Full | Complete multimodal support |
| oterm | ✅ Full | Ollama integration |
| MCPOmni-Connect | ✅ Full | Agentic mode support |
| Claude Desktop | ❌ | Use built-in Claude |
| VS Code Copilot | ❌ | Uses GitHub's models |

## Best Practices

1. **Error Handling**: Always validate token limits and message formats
2. **Timeout Management**: Set appropriate timeouts for LLM calls
3. **Caching**: Consider caching responses for identical requests
4. **Rate Limiting**: Implement rate limits to prevent API abuse
5. **Fallback Models**: Have fallback models for availability issues

## Security Considerations

1. **API Key Management**: Never hardcode API keys
2. **Input Sanitization**: Validate and sanitize user inputs
3. **Output Filtering**: Filter sensitive information from responses
4. **Usage Tracking**: Monitor and limit usage per client
5. **Prompt Injection**: Implement safeguards against prompt attacks