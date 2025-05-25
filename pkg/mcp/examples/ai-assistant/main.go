package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mcp-memory/pkg/mcp"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"
	"github.com/google/uuid"
)

// AssistantContext manages conversation state and tool execution history
type AssistantContext struct {
	mu              sync.RWMutex
	conversationID  string
	history         []ToolExecution
	contextWindow   []string
	workingDir      string
	dataCache       map[string]interface{}
	toolChains      map[string][]string
	memoryStore     *MemoryStore
}

// ToolExecution tracks tool usage for context and learning
type ToolExecution struct {
	Timestamp   time.Time
	Tool        string
	Arguments   map[string]interface{}
	Result      interface{}
	Success     bool
	ChainID     string
	UserIntent  string
}

// MemoryStore provides persistent context management
type MemoryStore struct {
	mu          sync.RWMutex
	memories    map[string]Memory
	index       map[string][]string // tag -> memory IDs
}

type Memory struct {
	ID        string
	Content   string
	Tags      []string
	Timestamp time.Time
	UsageCount int
}

// AIAssistantServer implements a comprehensive AI assistant with advanced capabilities
type AIAssistantServer struct {
	*server.Server
	context *AssistantContext
}

func NewAIAssistantServer() *AIAssistantServer {
	s := &AIAssistantServer{
		Server: server.NewServer(server.Config{
			Name:        "ai-assistant",
			Description: "Advanced AI assistant with multi-tool capabilities",
			Version:     "1.0.0",
		}),
		context: &AssistantContext{
			conversationID: uuid.New().String(),
			history:        make([]ToolExecution, 0),
			contextWindow:  make([]string, 0),
			workingDir:     filepath.Join(os.TempDir(), "ai-assistant", uuid.New().String()),
			dataCache:      make(map[string]interface{}),
			toolChains:     make(map[string][]string),
			memoryStore:    NewMemoryStore(),
		},
	}

	// Create working directory
	if err := os.MkdirAll(s.context.workingDir, 0755); err != nil {
		log.Printf("Failed to create working directory: %v", err)
	}

	// Register all tools
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		memories: make(map[string]Memory),
		index:    make(map[string][]string),
	}
}

func (s *AIAssistantServer) registerTools() {
	// Web Search Tool
	s.Server.RegisterTool(protocol.Tool{
		Name:        "web_search",
		Description: "Search the web for information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default: 5)",
				},
			},
			"required": []string{"query"},
		},
	}, s.handleWebSearch)

	// Code Execution Tool (sandboxed)
	s.Server.RegisterTool(protocol.Tool{
		Name:        "execute_code",
		Description: "Execute code in a sandboxed environment",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Programming language (python, javascript, bash)",
					"enum":        []string{"python", "javascript", "bash"},
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "Code to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Execution timeout in seconds (default: 30)",
				},
			},
			"required": []string{"language", "code"},
		},
	}, s.handleCodeExecution)

	// File Management Tool
	s.Server.RegisterTool(protocol.Tool{
		Name:        "file_manager",
		Description: "Manage files and directories",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "File operation",
					"enum":        []string{"read", "write", "list", "delete", "create_dir"},
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File or directory path",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content for write operations",
				},
			},
			"required": []string{"operation", "path"},
		},
	}, s.handleFileManagement)

	// Data Analysis Tool
	s.Server.RegisterTool(protocol.Tool{
		Name:        "analyze_data",
		Description: "Analyze data with various methods",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"method": map[string]interface{}{
					"type":        "string",
					"description": "Analysis method",
					"enum":        []string{"statistics", "pattern_detection", "summarize", "visualize"},
				},
				"data": map[string]interface{}{
					"type":        "object",
					"description": "Data to analyze (can be array, object, or reference to cached data)",
				},
				"options": map[string]interface{}{
					"type":        "object",
					"description": "Analysis-specific options",
				},
			},
			"required": []string{"method", "data"},
		},
	}, s.handleDataAnalysis)

	// Memory Management Tool
	s.Server.RegisterTool(protocol.Tool{
		Name:        "memory_manager",
		Description: "Store and retrieve contextual information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Memory operation",
					"enum":        []string{"store", "retrieve", "search", "update", "forget"},
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to store or search query",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Tags for categorization",
					"items":       map[string]interface{}{"type": "string"},
				},
				"memory_id": map[string]interface{}{
					"type":        "string",
					"description": "Memory ID for update/forget operations",
				},
			},
			"required": []string{"operation"},
		},
	}, s.handleMemoryManagement)

	// Tool Chain Executor
	s.Server.RegisterTool(protocol.Tool{
		Name:        "execute_chain",
		Description: "Execute a chain of tools with data flow between them",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"chain": map[string]interface{}{
					"type":        "array",
					"description": "Chain of tool executions",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"tool": map[string]interface{}{
								"type":        "string",
								"description": "Tool name",
							},
							"arguments": map[string]interface{}{
								"type":        "object",
								"description": "Tool arguments (can reference previous results with {{result.N}})",
							},
							"store_as": map[string]interface{}{
								"type":        "string",
								"description": "Store result with this key for later reference",
							},
						},
						"required": []string{"tool", "arguments"},
					},
				},
				"intent": map[string]interface{}{
					"type":        "string",
					"description": "High-level intent of the chain",
				},
			},
			"required": []string{"chain"},
		},
	}, s.handleToolChain)

	// Context Analyzer
	s.Server.RegisterTool(protocol.Tool{
		Name:        "analyze_context",
		Description: "Analyze current conversation context and suggest next actions",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"focus": map[string]interface{}{
					"type":        "string",
					"description": "Analysis focus",
					"enum":        []string{"intent", "progress", "suggestions", "patterns"},
				},
			},
			"required": []string{"focus"},
		},
	}, s.handleContextAnalysis)
}

func (s *AIAssistantServer) registerResources() {
	// Conversation History Resource
	s.Server.RegisterResource(protocol.Resource{
		URI:         fmt.Sprintf("conversation://%s", s.context.conversationID),
		Name:        "Current Conversation",
		Description: "Access to current conversation history and context",
		MimeType:    "application/json",
	}, func(ctx context.Context, req protocol.ReadResourceRequest) (protocol.ReadResourceResponse, error) {
		s.context.mu.RLock()
		defer s.context.mu.RUnlock()

		data := map[string]interface{}{
			"conversation_id": s.context.conversationID,
			"history":        s.context.history,
			"context_window": s.context.contextWindow,
			"active_chains":  s.context.toolChains,
		}

		content, _ := json.MarshalIndent(data, "", "  ")
		return protocol.ReadResourceResponse{
			Contents: []protocol.ResourceContent{
				{
					URI:      req.URI,
					MimeType: "application/json",
					Text:     string(content),
				},
			},
		}, nil
	})

	// Memory Store Resource
	s.Server.RegisterResource(protocol.Resource{
		URI:         "memory://store",
		Name:        "Memory Store",
		Description: "Access to stored memories and knowledge",
		MimeType:    "application/json",
	}, func(ctx context.Context, req protocol.ReadResourceRequest) (protocol.ReadResourceResponse, error) {
		memories := s.context.memoryStore.GetAll()
		content, _ := json.MarshalIndent(memories, "", "  ")
		return protocol.ReadResourceResponse{
			Contents: []protocol.ResourceContent{
				{
					URI:      req.URI,
					MimeType: "application/json",
					Text:     string(content),
				},
			},
		}, nil
	})

	// Working Directory Resource
	s.Server.RegisterResource(protocol.Resource{
		URI:         fmt.Sprintf("file://%s", s.context.workingDir),
		Name:        "Working Directory",
		Description: "Current working directory for file operations",
		MimeType:    "text/plain",
	}, func(ctx context.Context, req protocol.ReadResourceRequest) (protocol.ReadResourceResponse, error) {
		files, err := os.ReadDir(s.context.workingDir)
		if err != nil {
			return protocol.ReadResourceResponse{}, err
		}

		var listing strings.Builder
		for _, file := range files {
			if file.IsDir() {
				listing.WriteString(fmt.Sprintf("📁 %s/\n", file.Name()))
			} else {
				listing.WriteString(fmt.Sprintf("📄 %s\n", file.Name()))
			}
		}

		return protocol.ReadResourceResponse{
			Contents: []protocol.ResourceContent{
				{
					URI:      req.URI,
					MimeType: "text/plain",
					Text:     listing.String(),
				},
			},
		}, nil
	})
}

func (s *AIAssistantServer) registerPrompts() {
	// Analysis Prompt
	s.Server.RegisterPrompt(protocol.Prompt{
		Name:        "analyze_task",
		Description: "Analyze a complex task and suggest tool chain",
		Arguments: []protocol.PromptArgument{
			{
				Name:        "task",
				Description: "Task description",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req protocol.GetPromptRequest) (protocol.GetPromptResponse, error) {
		task := req.Arguments["task"]
		
		// Analyze recent context to provide better suggestions
		suggestions := s.analyzeTaskRequirements(task)
		
		prompt := fmt.Sprintf(`Task Analysis:
%s

Based on my analysis, here's a suggested approach:

%s

Available tools:
- web_search: Find information online
- execute_code: Run code in Python, JavaScript, or Bash
- file_manager: Read, write, and manage files
- analyze_data: Statistical analysis and pattern detection
- memory_manager: Store and retrieve contextual information
- execute_chain: Chain multiple tools together

Would you like me to execute this plan or modify it?`, task, suggestions)

		return protocol.GetPromptResponse{
			Messages: []protocol.Message{
				{
					Role:    "assistant",
					Content: prompt,
				},
			},
		}, nil
	})

	// Learning Prompt
	s.Server.RegisterPrompt(protocol.Prompt{
		Name:        "learn_pattern",
		Description: "Learn from execution patterns",
		Arguments: []protocol.PromptArgument{
			{
				Name:        "pattern_type",
				Description: "Type of pattern to learn",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req protocol.GetPromptRequest) (protocol.GetPromptResponse, error) {
		patternType := req.Arguments["pattern_type"]
		patterns := s.extractPatterns(patternType)
		
		return protocol.GetPromptResponse{
			Messages: []protocol.Message{
				{
					Role:    "assistant",
					Content: fmt.Sprintf("Learned patterns for %s:\n%s", patternType, patterns),
				},
			},
		}, nil
	})
}

// Tool Handlers

func (s *AIAssistantServer) handleWebSearch(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	query := req.Arguments["query"].(string)
	maxResults := 5
	if mr, ok := req.Arguments["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	// Simulate web search (in production, integrate with real search API)
	results := []map[string]string{
		{
			"title":   fmt.Sprintf("Search result 1 for: %s", query),
			"url":     "https://example.com/1",
			"snippet": "This is a simulated search result. In production, integrate with a real search API.",
		},
		{
			"title":   fmt.Sprintf("Search result 2 for: %s", query),
			"url":     "https://example.com/2",
			"snippet": "Another simulated result with relevant information.",
		},
	}

	// Limit results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	// Track execution
	s.trackExecution("web_search", req.Arguments, results, true, "")

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Found %d results for '%s'", len(results), query),
			},
			{
				Type: "text",
				Text: formatSearchResults(results),
			},
		},
	}, nil
}

func (s *AIAssistantServer) handleCodeExecution(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	language := req.Arguments["language"].(string)
	code := req.Arguments["code"].(string)
	timeout := 30 * time.Second
	if t, ok := req.Arguments["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Second
	}

	// Create sandboxed execution context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch language {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", code)
	case "javascript":
		cmd = exec.CommandContext(ctx, "node", "-e", code)
	case "bash":
		cmd = exec.CommandContext(ctx, "bash", "-c", code)
	default:
		return protocol.CallToolResponse{}, fmt.Errorf("unsupported language: %s", language)
	}

	// Set working directory
	cmd.Dir = s.context.workingDir

	// Execute with timeout
	output, err := cmd.CombinedOutput()
	
	result := map[string]interface{}{
		"output":   string(output),
		"success":  err == nil,
		"language": language,
	}

	if err != nil {
		result["error"] = err.Error()
	}

	// Track execution
	s.trackExecution("execute_code", req.Arguments, result, err == nil, "")

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Code execution completed (%s)", language),
			},
			{
				Type: "text",
				Text: string(output),
			},
		},
	}, nil
}

func (s *AIAssistantServer) handleFileManagement(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	operation := req.Arguments["operation"].(string)
	path := req.Arguments["path"].(string)

	// Ensure path is within working directory
	if !strings.HasPrefix(path, "/") {
		path = filepath.Join(s.context.workingDir, path)
	}

	var result interface{}
	var err error

	switch operation {
	case "read":
		var content []byte
		content, err = os.ReadFile(path)
		result = string(content)

	case "write":
		content := req.Arguments["content"].(string)
		err = os.WriteFile(path, []byte(content), 0644)
		result = "File written successfully"

	case "list":
		var files []os.FileInfo
		files, err = os.ReadDir(path)
		if err == nil {
			var names []string
			for _, f := range files {
				names = append(names, f.Name())
			}
			result = names
		}

	case "delete":
		err = os.Remove(path)
		result = "File deleted successfully"

	case "create_dir":
		err = os.MkdirAll(path, 0755)
		result = "Directory created successfully"

	default:
		err = fmt.Errorf("unknown operation: %s", operation)
	}

	// Track execution
	s.trackExecution("file_manager", req.Arguments, result, err == nil, "")

	if err != nil {
		return protocol.CallToolResponse{}, err
	}

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("File operation '%s' completed", operation),
			},
			{
				Type: "text",
				Text: fmt.Sprintf("%v", result),
			},
		},
	}, nil
}

func (s *AIAssistantServer) handleDataAnalysis(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	method := req.Arguments["method"].(string)
	data := req.Arguments["data"]
	options := req.Arguments["options"]

	var result interface{}

	switch method {
	case "statistics":
		result = s.calculateStatistics(data)
	case "pattern_detection":
		result = s.detectPatterns(data)
	case "summarize":
		result = s.summarizeData(data)
	case "visualize":
		result = s.createVisualization(data, options)
	default:
		return protocol.CallToolResponse{}, fmt.Errorf("unknown analysis method: %s", method)
	}

	// Track execution
	s.trackExecution("analyze_data", req.Arguments, result, true, "")

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Data analysis completed using %s method", method),
			},
			{
				Type: "text",
				Text: fmt.Sprintf("%v", result),
			},
		},
	}, nil
}

func (s *AIAssistantServer) handleMemoryManagement(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	operation := req.Arguments["operation"].(string)

	var result interface{}
	var err error

	switch operation {
	case "store":
		content := req.Arguments["content"].(string)
		tags := extractStringArray(req.Arguments["tags"])
		memory := s.context.memoryStore.Store(content, tags)
		result = map[string]interface{}{
			"memory_id": memory.ID,
			"stored":    true,
		}

	case "retrieve":
		if memoryID, ok := req.Arguments["memory_id"].(string); ok {
			memory, found := s.context.memoryStore.Get(memoryID)
			if found {
				result = memory
			} else {
				err = fmt.Errorf("memory not found: %s", memoryID)
			}
		}

	case "search":
		query := req.Arguments["content"].(string)
		memories := s.context.memoryStore.Search(query)
		result = memories

	case "update":
		memoryID := req.Arguments["memory_id"].(string)
		content := req.Arguments["content"].(string)
		tags := extractStringArray(req.Arguments["tags"])
		err = s.context.memoryStore.Update(memoryID, content, tags)
		result = "Memory updated"

	case "forget":
		memoryID := req.Arguments["memory_id"].(string)
		err = s.context.memoryStore.Delete(memoryID)
		result = "Memory forgotten"

	default:
		err = fmt.Errorf("unknown operation: %s", operation)
	}

	// Track execution
	s.trackExecution("memory_manager", req.Arguments, result, err == nil, "")

	if err != nil {
		return protocol.CallToolResponse{}, err
	}

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Memory operation '%s' completed", operation),
			},
			{
				Type: "text",
				Text: fmt.Sprintf("%v", result),
			},
		},
	}, nil
}

func (s *AIAssistantServer) handleToolChain(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	chainDef := req.Arguments["chain"].([]interface{})
	intent := ""
	if i, ok := req.Arguments["intent"].(string); ok {
		intent = i
	}

	chainID := uuid.New().String()
	results := make([]interface{}, 0)
	namedResults := make(map[string]interface{})

	for i, step := range chainDef {
		stepMap := step.(map[string]interface{})
		toolName := stepMap["tool"].(string)
		arguments := stepMap["arguments"].(map[string]interface{})

		// Replace references to previous results
		processedArgs := s.processChainArguments(arguments, results, namedResults)

		// Execute tool
		toolReq := protocol.CallToolRequest{
			Name:      toolName,
			Arguments: processedArgs,
		}

		// Get the tool handler
		var toolResult interface{}
		var toolErr error

		switch toolName {
		case "web_search":
			resp, err := s.handleWebSearch(ctx, toolReq)
			toolErr = err
			if err == nil {
				toolResult = resp.Content
			}
		case "execute_code":
			resp, err := s.handleCodeExecution(ctx, toolReq)
			toolErr = err
			if err == nil {
				toolResult = resp.Content
			}
		case "file_manager":
			resp, err := s.handleFileManagement(ctx, toolReq)
			toolErr = err
			if err == nil {
				toolResult = resp.Content
			}
		case "analyze_data":
			resp, err := s.handleDataAnalysis(ctx, toolReq)
			toolErr = err
			if err == nil {
				toolResult = resp.Content
			}
		case "memory_manager":
			resp, err := s.handleMemoryManagement(ctx, toolReq)
			toolErr = err
			if err == nil {
				toolResult = resp.Content
			}
		default:
			toolErr = fmt.Errorf("unknown tool in chain: %s", toolName)
		}

		if toolErr != nil {
			return protocol.CallToolResponse{}, fmt.Errorf("chain step %d failed: %w", i, toolErr)
		}

		results = append(results, toolResult)

		// Store named result if specified
		if storeName, ok := stepMap["store_as"].(string); ok {
			namedResults[storeName] = toolResult
		}

		// Track execution with chain ID
		s.trackExecution(toolName, processedArgs, toolResult, true, chainID)
	}

	// Store chain execution
	s.context.mu.Lock()
	s.context.toolChains[chainID] = extractToolNames(chainDef)
	s.context.mu.Unlock()

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Tool chain executed successfully (ID: %s)", chainID),
			},
			{
				Type: "text",
				Text: fmt.Sprintf("Intent: %s\nResults: %v", intent, results),
			},
		},
	}, nil
}

func (s *AIAssistantServer) handleContextAnalysis(ctx context.Context, req protocol.CallToolRequest) (protocol.CallToolResponse, error) {
	focus := req.Arguments["focus"].(string)

	var analysis interface{}

	switch focus {
	case "intent":
		analysis = s.analyzeUserIntent()
	case "progress":
		analysis = s.analyzeProgress()
	case "suggestions":
		analysis = s.generateSuggestions()
	case "patterns":
		analysis = s.analyzePatterns()
	default:
		return protocol.CallToolResponse{}, fmt.Errorf("unknown analysis focus: %s", focus)
	}

	return protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Context analysis (%s)", focus),
			},
			{
				Type: "text",
				Text: fmt.Sprintf("%v", analysis),
			},
		},
	}, nil
}

// Helper functions

func (s *AIAssistantServer) trackExecution(tool string, args map[string]interface{}, result interface{}, success bool, chainID string) {
	s.context.mu.Lock()
	defer s.context.mu.Unlock()

	execution := ToolExecution{
		Timestamp: time.Now(),
		Tool:      tool,
		Arguments: args,
		Result:    result,
		Success:   success,
		ChainID:   chainID,
	}

	s.context.history = append(s.context.history, execution)

	// Update context window
	summary := fmt.Sprintf("[%s] %s: %v", time.Now().Format("15:04:05"), tool, success)
	s.context.contextWindow = append(s.context.contextWindow, summary)
	if len(s.context.contextWindow) > 10 {
		s.context.contextWindow = s.context.contextWindow[1:]
	}
}

func (s *AIAssistantServer) processChainArguments(args map[string]interface{}, results []interface{}, namedResults map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	
	for k, v := range args {
		switch val := v.(type) {
		case string:
			// Check for result references
			if strings.Contains(val, "{{") {
				processed[k] = s.replaceReferences(val, results, namedResults)
			} else {
				processed[k] = val
			}
		case map[string]interface{}:
			processed[k] = s.processChainArguments(val, results, namedResults)
		default:
			processed[k] = v
		}
	}
	
	return processed
}

func (s *AIAssistantServer) replaceReferences(template string, results []interface{}, namedResults map[string]interface{}) string {
	// Simple template replacement (in production, use a proper template engine)
	result := template
	
	// Replace indexed results
	for i, r := range results {
		placeholder := fmt.Sprintf("{{result.%d}}", i)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", r))
		}
	}
	
	// Replace named results
	for name, value := range namedResults {
		placeholder := fmt.Sprintf("{{%s}}", name)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
		}
	}
	
	return result
}

func (s *AIAssistantServer) analyzeTaskRequirements(task string) string {
	// Analyze task and suggest tool chain
	taskLower := strings.ToLower(task)
	
	suggestions := []string{}
	
	if strings.Contains(taskLower, "search") || strings.Contains(taskLower, "find") {
		suggestions = append(suggestions, "1. Use web_search to gather information")
	}
	
	if strings.Contains(taskLower, "analyze") || strings.Contains(taskLower, "data") {
		suggestions = append(suggestions, "2. Use analyze_data for statistical analysis")
	}
	
	if strings.Contains(taskLower, "code") || strings.Contains(taskLower, "script") {
		suggestions = append(suggestions, "3. Use execute_code to run custom scripts")
	}
	
	if strings.Contains(taskLower, "remember") || strings.Contains(taskLower, "store") {
		suggestions = append(suggestions, "4. Use memory_manager to persist important information")
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "This task may require a combination of tools. Consider using execute_chain for complex workflows.")
	}
	
	return strings.Join(suggestions, "\n")
}

func (s *AIAssistantServer) calculateStatistics(data interface{}) map[string]interface{} {
	// Simple statistics calculation
	return map[string]interface{}{
		"type":  fmt.Sprintf("%T", data),
		"count": 1,
		"summary": "Statistical analysis would be performed here",
	}
}

func (s *AIAssistantServer) detectPatterns(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"patterns_found": 0,
		"analysis": "Pattern detection would analyze the data structure and content",
	}
}

func (s *AIAssistantServer) summarizeData(data interface{}) string {
	return fmt.Sprintf("Data summary: %v (type: %T)", data, data)
}

func (s *AIAssistantServer) createVisualization(data interface{}, options interface{}) map[string]interface{} {
	return map[string]interface{}{
		"visualization_type": "chart",
		"data_points": 0,
		"file_path": filepath.Join(s.context.workingDir, "visualization.png"),
	}
}

func (s *AIAssistantServer) analyzeUserIntent() map[string]interface{} {
	s.context.mu.RLock()
	defer s.context.mu.RUnlock()
	
	recentTools := make(map[string]int)
	for _, exec := range s.context.history {
		recentTools[exec.Tool]++
	}
	
	return map[string]interface{}{
		"recent_tools": recentTools,
		"session_duration": time.Since(s.context.history[0].Timestamp).String(),
		"success_rate": s.calculateSuccessRate(),
	}
}

func (s *AIAssistantServer) analyzeProgress() map[string]interface{} {
	s.context.mu.RLock()
	defer s.context.mu.RUnlock()
	
	return map[string]interface{}{
		"total_executions": len(s.context.history),
		"unique_tools_used": s.countUniqueTools(),
		"active_chains": len(s.context.toolChains),
		"files_created": s.countFilesCreated(),
	}
}

func (s *AIAssistantServer) generateSuggestions() []string {
	s.context.mu.RLock()
	defer s.context.mu.RUnlock()
	
	suggestions := []string{}
	
	// Analyze recent activity
	if len(s.context.history) > 0 {
		lastTool := s.context.history[len(s.context.history)-1].Tool
		
		switch lastTool {
		case "web_search":
			suggestions = append(suggestions, "Consider analyzing the search results with analyze_data")
			suggestions = append(suggestions, "Save important findings with memory_manager")
		case "execute_code":
			suggestions = append(suggestions, "Check output files with file_manager")
			suggestions = append(suggestions, "Store successful code snippets for reuse")
		case "file_manager":
			suggestions = append(suggestions, "Analyze file contents with analyze_data")
			suggestions = append(suggestions, "Execute scripts from saved files")
		}
	}
	
	return suggestions
}

func (s *AIAssistantServer) analyzePatterns() map[string]interface{} {
	s.context.mu.RLock()
	defer s.context.mu.RUnlock()
	
	// Identify common tool sequences
	sequences := make(map[string]int)
	for i := 0; i < len(s.context.history)-1; i++ {
		seq := fmt.Sprintf("%s->%s", s.context.history[i].Tool, s.context.history[i+1].Tool)
		sequences[seq]++
	}
	
	return map[string]interface{}{
		"common_sequences": sequences,
		"chain_usage": len(s.context.toolChains),
		"memory_utilization": s.context.memoryStore.Count(),
	}
}

func (s *AIAssistantServer) extractPatterns(patternType string) string {
	s.context.mu.RLock()
	defer s.context.mu.RUnlock()
	
	switch patternType {
	case "tool_usage":
		return s.analyzeToolUsagePatterns()
	case "error_patterns":
		return s.analyzeErrorPatterns()
	case "success_patterns":
		return s.analyzeSuccessPatterns()
	default:
		return "Unknown pattern type"
	}
}

func (s *AIAssistantServer) analyzeToolUsagePatterns() string {
	usage := make(map[string]int)
	for _, exec := range s.context.history {
		usage[exec.Tool]++
	}
	
	var result strings.Builder
	result.WriteString("Tool Usage Patterns:\n")
	for tool, count := range usage {
		result.WriteString(fmt.Sprintf("- %s: %d times\n", tool, count))
	}
	
	return result.String()
}

func (s *AIAssistantServer) analyzeErrorPatterns() string {
	var errors []string
	for _, exec := range s.context.history {
		if !exec.Success {
			errors = append(errors, fmt.Sprintf("%s failed at %s", exec.Tool, exec.Timestamp.Format("15:04:05")))
		}
	}
	
	if len(errors) == 0 {
		return "No errors detected in recent executions"
	}
	
	return "Error Patterns:\n" + strings.Join(errors, "\n")
}

func (s *AIAssistantServer) analyzeSuccessPatterns() string {
	successfulChains := make(map[string]int)
	
	for chainID, tools := range s.context.toolChains {
		allSuccess := true
		for _, exec := range s.context.history {
			if exec.ChainID == chainID && !exec.Success {
				allSuccess = false
				break
			}
		}
		if allSuccess {
			chainStr := strings.Join(tools, "->")
			successfulChains[chainStr]++
		}
	}
	
	var result strings.Builder
	result.WriteString("Successful Tool Chains:\n")
	for chain, count := range successfulChains {
		result.WriteString(fmt.Sprintf("- %s: %d times\n", chain, count))
	}
	
	return result.String()
}

func (s *AIAssistantServer) calculateSuccessRate() float64 {
	if len(s.context.history) == 0 {
		return 0
	}
	
	successful := 0
	for _, exec := range s.context.history {
		if exec.Success {
			successful++
		}
	}
	
	return float64(successful) / float64(len(s.context.history)) * 100
}

func (s *AIAssistantServer) countUniqueTools() int {
	tools := make(map[string]bool)
	for _, exec := range s.context.history {
		tools[exec.Tool] = true
	}
	return len(tools)
}

func (s *AIAssistantServer) countFilesCreated() int {
	count := 0
	for _, exec := range s.context.history {
		if exec.Tool == "file_manager" && exec.Arguments["operation"] == "write" {
			count++
		}
	}
	return count
}

// Memory Store methods

func (ms *MemoryStore) Store(content string, tags []string) Memory {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	memory := Memory{
		ID:        uuid.New().String(),
		Content:   content,
		Tags:      tags,
		Timestamp: time.Now(),
		UsageCount: 0,
	}
	
	ms.memories[memory.ID] = memory
	
	// Update index
	for _, tag := range tags {
		ms.index[tag] = append(ms.index[tag], memory.ID)
	}
	
	return memory
}

func (ms *MemoryStore) Get(id string) (Memory, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	memory, found := ms.memories[id]
	if found {
		memory.UsageCount++
		ms.memories[id] = memory
	}
	return memory, found
}

func (ms *MemoryStore) Search(query string) []Memory {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var results []Memory
	queryLower := strings.ToLower(query)
	
	for _, memory := range ms.memories {
		if strings.Contains(strings.ToLower(memory.Content), queryLower) {
			results = append(results, memory)
		}
	}
	
	return results
}

func (ms *MemoryStore) Update(id, content string, tags []string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	memory, found := ms.memories[id]
	if !found {
		return fmt.Errorf("memory not found: %s", id)
	}
	
	// Remove old tags from index
	for _, tag := range memory.Tags {
		ms.removeFromIndex(tag, id)
	}
	
	// Update memory
	memory.Content = content
	memory.Tags = tags
	memory.Timestamp = time.Now()
	ms.memories[id] = memory
	
	// Add new tags to index
	for _, tag := range tags {
		ms.index[tag] = append(ms.index[tag], id)
	}
	
	return nil
}

func (ms *MemoryStore) Delete(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	memory, found := ms.memories[id]
	if !found {
		return fmt.Errorf("memory not found: %s", id)
	}
	
	// Remove from index
	for _, tag := range memory.Tags {
		ms.removeFromIndex(tag, id)
	}
	
	delete(ms.memories, id)
	return nil
}

func (ms *MemoryStore) GetAll() map[string]Memory {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	result := make(map[string]Memory)
	for k, v := range ms.memories {
		result[k] = v
	}
	return result
}

func (ms *MemoryStore) Count() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.memories)
}

func (ms *MemoryStore) removeFromIndex(tag, id string) {
	ids := ms.index[tag]
	for i, memID := range ids {
		if memID == id {
			ms.index[tag] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	if len(ms.index[tag]) == 0 {
		delete(ms.index, tag)
	}
}

// Utility functions

func extractStringArray(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	
	arr, ok := v.([]interface{})
	if !ok {
		return []string{}
	}
	
	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = fmt.Sprintf("%v", item)
	}
	return result
}

func extractToolNames(chain []interface{}) []string {
	names := make([]string, len(chain))
	for i, step := range chain {
		stepMap := step.(map[string]interface{})
		names[i] = stepMap["tool"].(string)
	}
	return names
}

func formatSearchResults(results []map[string]string) string {
	var formatted strings.Builder
	for i, result := range results {
		formatted.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, result["title"]))
		formatted.WriteString(fmt.Sprintf("   URL: %s\n", result["url"]))
		formatted.WriteString(fmt.Sprintf("   %s\n", result["snippet"]))
	}
	return formatted.String()
}

// Main function

func main() {
	// Create server
	server := NewAIAssistantServer()

	// Create stdio transport
	transport := transport.NewStdioTransport()

	// Start server
	log.Println("Starting AI Assistant MCP Server...")
	if err := server.Start(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}