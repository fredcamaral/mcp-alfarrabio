package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/constants"
)

// createMemoryCommand creates the memory command group
func (c *CLI) createMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Memory management commands",
		Long: `Memory management commands for storing, retrieving, and analyzing project knowledge.

The memory system allows you to:
- Store PRDs, TRDs, and code review results
- Search through past conversations and decisions
- Learn from patterns across projects
- Get intelligent suggestions based on history`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createMemoryStoreCommand(),
		c.createMemorySearchCommand(),
		c.createMemoryGetCommand(),
		c.createMemoryListCommand(),
		c.createMemoryLearnCommand(),
		c.createMemoryPatternsCommand(),
		c.createMemorySuggestCommand(),
		c.createMemoryInsightsCommand(),
		c.createMemoryCompareCommand(),
	)

	return cmd
}

// createMemoryStoreCommand creates the memory store command group
func (c *CLI) createMemoryStoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Store documents in memory",
		Long:  "Store PRDs, TRDs, reviews, and other documents in the memory system for future reference and learning.",
	}

	// Add store subcommands
	cmd.AddCommand(
		c.createMemoryStorePRDCommand(),
		c.createMemoryStoreTRDCommand(),
		c.createMemoryStoreReviewCommand(),
		c.createMemoryStoreDecisionCommand(),
	)

	return cmd
}

// createMemoryStorePRDCommand stores a PRD in memory
func (c *CLI) createMemoryStorePRDCommand() *cobra.Command {
	var file string
	var project string
	var tags []string

	cmd := &cobra.Command{
		Use:   "prd",
		Short: "Store a PRD in memory",
		Long:  "Store a Product Requirements Document in the memory system for future reference and pattern learning.",
		Example: `  # Store a PRD file
  lmmc memory store prd --file prd-auth.md --project myproject
  
  # Store with tags
  lmmc memory store prd --file prd-auth.md --project myproject --tag authentication --tag security`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryStorePRD(cmd, file, project, tags)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "PRD file to store (required)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringSliceVar(&tags, "tag", []string{}, "Tags to associate with the PRD")

	markFlagRequired(cmd, "file", "project")

	return cmd
}

// createMemoryStoreTRDCommand stores a TRD in memory
func (c *CLI) createMemoryStoreTRDCommand() *cobra.Command {
	var file string
	var project string
	var prdID string
	var tags []string

	cmd := &cobra.Command{
		Use:   "trd",
		Short: "Store a TRD in memory",
		Long:  "Store a Technical Requirements Document in the memory system, linking it to the associated PRD.",
		Example: `  # Store a TRD file
  lmmc memory store trd --file trd-auth.md --project myproject
  
  # Store with explicit PRD link
  lmmc memory store trd --file trd-auth.md --project myproject --prd-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryStoreTRD(cmd, file, project, prdID, tags)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "TRD file to store (required)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringVar(&prdID, "prd-id", "", "Associated PRD ID (optional)")
	cmd.Flags().StringSliceVar(&tags, "tag", []string{}, "Tags to associate with the TRD")

	markFlagRequired(cmd, "file", "project")

	return cmd
}

// createMemoryStoreReviewCommand stores a code review in memory
func (c *CLI) createMemoryStoreReviewCommand() *cobra.Command {
	var sessionID string
	var project string
	var tags []string

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Store a code review session in memory",
		Long:  "Store code review results and findings in the memory system for learning and pattern detection.",
		Example: `  # Store current review session
  lmmc memory store review --session current --project myproject
  
  # Store specific review session
  lmmc memory store review --session abc-123-def --project myproject`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryStoreReview(cmd, sessionID, project, tags)
		},
	}

	cmd.Flags().StringVarP(&sessionID, "session", "s", "current", "Review session ID (default: current)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringSliceVar(&tags, "tag", []string{}, "Tags to associate with the review")

	markFlagRequired(cmd, "project")

	return cmd
}

// createMemoryStoreDecisionCommand stores an architectural decision
func (c *CLI) createMemoryStoreDecisionCommand() *cobra.Command {
	var decision string
	var rationale string
	var project string
	var tags []string

	cmd := &cobra.Command{
		Use:   "decision",
		Short: "Store an architectural decision",
		Long:  "Store an architectural or design decision with its rationale for future reference.",
		Example: `  # Store a decision
  lmmc memory store decision --decision "Use PostgreSQL for user data" \
    --rationale "Better support for complex queries and JSONB fields" \
    --project myproject`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryStoreDecision(cmd, decision, rationale, project, tags)
		},
	}

	cmd.Flags().StringVarP(&decision, "decision", "d", "", "The decision made (required)")
	cmd.Flags().StringVarP(&rationale, "rationale", "r", "", "Rationale for the decision (required)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringSliceVar(&tags, "tag", []string{}, "Tags to associate with the decision")

	markFlagRequired(cmd, "decision", "rationale", "project")

	return cmd
}

// createMemorySearchCommand searches memory
func (c *CLI) createMemorySearchCommand() *cobra.Command {
	var project string
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search through stored memories",
		Long:  "Search for memories using natural language queries. Returns relevant content from PRDs, TRDs, reviews, and decisions.",
		Example: `  # Search for authentication-related memories
  lmmc memory search "authentication PRDs"
  
  # Search within a specific project
  lmmc memory search "database decisions" --project myproject
  
  # Limit results
  lmmc memory search "bug fixes" --limit 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemorySearch(cmd, args[0], project, limit)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project")
	addCommonFlags(cmd, &format, &limit)

	return cmd
}

// createMemoryGetCommand retrieves a specific memory
func (c *CLI) createMemoryGetCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "get [type] [id]",
		Short: "Get a specific memory by ID",
		Long:  "Retrieve a specific memory item (PRD, TRD, review, or decision) by its ID.",
		Example: `  # Get a specific PRD
  lmmc memory get prd abc123
  
  # Get a TRD with JSON output
  lmmc memory get trd xyz789 --format ` + constants.OutputFormatJSON + ``,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryGet(cmd, args[0], args[1], format)
		},
		ValidArgs: []string{"prd", "trd", "review", "decision"},
	}

	addFormatFlag(cmd, &format, constants.OutputFormatTable)

	return cmd
}

// createMemoryListCommand lists memories
func (c *CLI) createMemoryListCommand() *cobra.Command {
	var memoryType string
	var project string
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored memories",
		Long:  "List memories filtered by type and project.",
		Example: `  # List all memories
  lmmc memory list
  
  # List PRDs for a project
  lmmc memory list --type prd --project myproject
  
  # List recent reviews
  lmmc memory list --type review --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryList(cmd, memoryType, project, limit, format)
		},
	}

	cmd.Flags().StringVarP(&memoryType, "type", "t", "", "Filter by type (prd, trd, review, decision)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results")
	addFormatFlag(cmd, &format, constants.OutputFormatTable)

	return cmd
}

// createMemoryLearnCommand learns from memories
func (c *CLI) createMemoryLearnCommand() *cobra.Command {
	var reviewID string
	var projectID string

	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Learn from stored memories",
		Long:  "Extract patterns and insights from reviews, decisions, and other memories to improve future suggestions.",
		Example: `  # Learn from a review session
  lmmc memory learn --from-review abc-123-def
  
  # Learn from all reviews in a project
  lmmc memory learn --from-project myproject`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reviewFlag, _ := cmd.Flags().GetString("from-review")
			projectFlag, _ := cmd.Flags().GetString("from-project")

			if reviewFlag != "" {
				return c.runMemoryLearnFromReview(cmd, reviewFlag)
			} else if projectFlag != "" {
				return c.runMemoryLearnFromProject(cmd, projectFlag)
			}
			return errors.New("please specify either --from-review or --from-project")
		},
	}

	cmd.Flags().StringVar(&reviewID, "from-review", "", "Learn from a specific review session")
	cmd.Flags().StringVar(&projectID, "from-project", "", "Learn from all memories in a project")

	return cmd
}

// createMemoryPatternsCommand shows patterns
func (c *CLI) createMemoryPatternsCommand() *cobra.Command {
	var project string
	var patternType string
	var format string

	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "Show detected patterns",
		Long:  "Display patterns detected across stored memories, including common issues, best practices, and recurring themes.",
		Example: `  # Show all patterns for a project
  lmmc memory patterns --project myproject
  
  # Show specific pattern types
  lmmc memory patterns --type security --project myproject`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryPatterns(cmd, project, patternType, format)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project (required)")
	cmd.Flags().StringVarP(&patternType, "type", "t", "", "Filter by pattern type")
	addFormatFlag(cmd, &format, constants.OutputFormatTable)

	markFlagRequired(cmd, "project")

	return cmd
}

// createMemorySuggestCommand provides suggestions
func (c *CLI) createMemorySuggestCommand() *cobra.Command {
	var feature string
	var project string
	var format string

	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Get suggestions based on memory",
		Long:  "Get intelligent suggestions for features based on past experiences and patterns.",
		Example: `  # Get suggestions for a feature
  lmmc memory suggest --for-feature "user authentication" --project myproject
  
  # Get general suggestions
  lmmc memory suggest --project myproject`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemorySuggest(cmd, feature, project, format)
		},
	}

	cmd.Flags().StringVarP(&feature, "for-feature", "f", "", "Feature to get suggestions for")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project context (required)")
	cmd.Flags().StringVar(&format, "format", constants.OutputFormatTable, "Output format ("+constants.OutputFormatTable+", "+constants.OutputFormatJSON+", "+constants.OutputFormatPlain+")")

	markFlagRequired(cmd, "project")

	return cmd
}

// createMemoryInsightsCommand shows insights
func (c *CLI) createMemoryInsightsCommand() *cobra.Command {
	var topic string
	var project string
	var crossProject bool
	var format string

	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Get insights from memories",
		Long:  "Generate insights from stored memories, including cross-project learnings and best practices.",
		Example: `  # Get insights for a topic
  lmmc memory insights --topic authentication
  
  # Get project-specific insights
  lmmc memory insights --project myproject
  
  # Get cross-project insights
  lmmc memory insights --topic security --cross-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryInsights(cmd, topic, project, crossProject, format)
		},
	}

	cmd.Flags().StringVarP(&topic, "topic", "t", "", "Topic to analyze")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project to analyze")
	cmd.Flags().BoolVar(&crossProject, "cross-project", false, "Include cross-project insights")
	addFormatFlag(cmd, &format, constants.OutputFormatTable)

	return cmd
}

// createMemoryCompareCommand compares projects
func (c *CLI) createMemoryCompareCommand() *cobra.Command {
	var projects []string
	var aspect string
	var format string

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare memories across projects",
		Long:  "Compare patterns, decisions, and practices across multiple projects.",
		Example: `  # Compare two projects
  lmmc memory compare --projects proj1,proj2
  
  # Compare specific aspects
  lmmc memory compare --projects proj1,proj2,proj3 --aspect security`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runMemoryCompare(cmd, projects, aspect, format)
		},
	}

	cmd.Flags().StringSliceVar(&projects, "projects", []string{}, "Projects to compare (comma-separated)")
	cmd.Flags().StringVarP(&aspect, "aspect", "a", "", "Specific aspect to compare")
	addFormatFlag(cmd, &format, constants.OutputFormatTable)

	markFlagRequired(cmd, "projects")

	return cmd
}

// Implementation methods

func (c *CLI) runMemoryStorePRD(cmd *cobra.Command, file, project string, tags []string) error {
	// Read PRD file content
	content, err := validateAndReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read PRD file: %w", err)
	}

	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_create tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_create", map[string]interface{}{
		"operation": "store_chunk",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": project,
			"session_id": fmt.Sprintf("cli-prd-%d", time.Now().Unix()),
			"content":    string(content),
			"metadata": map[string]interface{}{
				"type":     "prd",
				"filename": file,
				"tags":     tags,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store PRD: %w", err)
	}

	// Format output
	fmt.Fprintf(cmd.OutOrStdout(), "PRD stored successfully: %s\n", result["id"])
	return nil
}

func (c *CLI) runMemoryStoreTRD(cmd *cobra.Command, file, project, prdID string, tags []string) error {
	// Read TRD file content
	content, err := validateAndReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read TRD file: %w", err)
	}

	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Build metadata
	metadata := map[string]interface{}{
		"type":     "trd",
		"filename": file,
		"tags":     tags,
	}
	if prdID != "" {
		metadata["prd_id"] = prdID
	}

	// Create request for memory_create tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_create", map[string]interface{}{
		"operation": "store_chunk",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": project,
			"session_id": fmt.Sprintf("cli-trd-%d", time.Now().Unix()),
			"content":    string(content),
			"metadata":   metadata,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store TRD: %w", err)
	}

	// Format output
	fmt.Fprintf(cmd.OutOrStdout(), "TRD stored successfully: %s\n", result["id"])
	return nil
}

func (c *CLI) runMemoryStoreReview(cmd *cobra.Command, sessionID, project string, tags []string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// If sessionID is "current", get the current review session
	if sessionID == "current" {
		if c.reviewService == nil {
			return errors.New("review service not available")
		}
		// TODO: Get current session from review service
		sessionID = fmt.Sprintf("review-%d", time.Now().Unix())
	}

	// TODO: Get review content from review service
	reviewContent := map[string]interface{}{
		"session_id": sessionID,
		"project":    project,
		"timestamp":  time.Now().Format(time.RFC3339),
		"tags":       tags,
		// Add actual review findings here
	}

	// Create request for memory_create tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_create", map[string]interface{}{
		"operation": "store_chunk",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": project,
			"session_id": sessionID,
			"content":    "Code review session " + sessionID,
			"metadata": map[string]interface{}{
				"type":    "review",
				"tags":    tags,
				"details": reviewContent,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store review: %w", err)
	}

	// Format output
	fmt.Fprintf(cmd.OutOrStdout(), "Review stored successfully: %s\n", result["id"])
	return nil
}

func (c *CLI) runMemoryStoreDecision(cmd *cobra.Command, decision, rationale, project string, tags []string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_create tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_create", map[string]interface{}{
		"operation": "store_decision",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": project,
			"session_id": fmt.Sprintf("cli-decision-%d", time.Now().Unix()),
			"decision":   decision,
			"rationale":  rationale,
			"metadata": map[string]interface{}{
				"tags":      tags,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store decision: %w", err)
	}

	// Format output
	fmt.Fprintf(cmd.OutOrStdout(), "Decision stored successfully: %s\n", result["id"])
	return nil
}

func (c *CLI) runMemorySearch(cmd *cobra.Command, query, project string, limit int) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Build search options
	options := map[string]interface{}{
		"query": query,
		"limit": limit,
	}
	if project != "" {
		options["repository"] = project
	} else {
		options["repository"] = constants.RepositoryGlobal
	}

	// Create request for memory_read tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_read", map[string]interface{}{
		"operation": "search",
		"scope":     "single",
		"options":   options,
	})

	if err != nil {
		return fmt.Errorf("failed to search memories: %w", err)
	}

	// Debug: log the result structure
	if c.logger != nil {
		c.logger.Debug("Search result", "result", result)
	}

	// Format output
	if chunks, ok := result["chunks"].([]interface{}); ok && len(chunks) > 0 {
		for _, chunk := range chunks {
			if chunkMap, ok := chunk.(map[string]interface{}); ok {
				fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", chunkMap["id"].(string))
				fmt.Fprintf(cmd.OutOrStdout(), "Content: %s\n", chunkMap["content"].(string))
				fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\n", project)
				fmt.Fprintf(cmd.OutOrStdout(), "Score: %.2f\n\n", chunkMap["score"])
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "No results found\n")
	}

	return nil
}

func (c *CLI) runMemoryGet(cmd *cobra.Command, _ string, id, format string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_read tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_read", map[string]interface{}{
		"operation": "get_context",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": constants.RepositoryGlobal, // Search across all repositories
			"chunk_id":   id,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}

	// Format output
	if format == constants.OutputFormatJSON {
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(jsonData))
	} else {
		if content, ok := result["content"].(string); ok {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", content)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%v\n", result)
		}
	}

	return nil
}

func (c *CLI) runMemoryList(cmd *cobra.Command, memoryType, project string, limit int, _ string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Build list options
	options := map[string]interface{}{
		"limit": limit,
	}
	if project != "" {
		options["repository"] = project
	} else {
		options["repository"] = constants.RepositoryGlobal
	}
	if memoryType != "" {
		options["type"] = memoryType
	}

	// Create request for memory_read tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_read", map[string]interface{}{
		"operation": "get_context",
		"scope":     "single",
		"options":   options,
	})

	if err != nil {
		return fmt.Errorf("failed to list memories: %w", err)
	}

	// Format output
	if chunks, ok := result["recent_chunks"].([]interface{}); ok {
		for _, chunk := range chunks {
			if chunkMap, ok := chunk.(map[string]interface{}); ok {
				fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", chunkMap["id"].(string))
				fmt.Fprintf(cmd.OutOrStdout(), "Type: [%s]\n", chunkMap["type"])
				fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\n\n", project)
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "No memories found\n")
	}

	return nil
}

func (c *CLI) runMemoryLearnFromReview(cmd *cobra.Command, reviewID string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_analyze tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_analyze", map[string]interface{}{
		"operation": "detect_threads",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": constants.RepositoryGlobal,
			"session_id": reviewID,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to learn from review: %w", err)
	}

	// Format output
	fmt.Fprintf(cmd.OutOrStdout(), "Learning completed. Patterns detected: %v\n", result["threads_detected"])
	return nil
}

func (c *CLI) runMemoryLearnFromProject(cmd *cobra.Command, project string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_analyze tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_analyze", map[string]interface{}{
		"operation": "cross_repo_patterns",
		"scope":     "cross_repo",
		"options": map[string]interface{}{
			"repository": project,
			"session_id": fmt.Sprintf("cli-learn-%d", time.Now().Unix()),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to learn from project: %w", err)
	}

	// Format output
	if patterns, ok := result["patterns"].([]interface{}); ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Found %d patterns across the project\n", len(patterns))
		for _, pattern := range patterns {
			if patternMap, ok := pattern.(map[string]interface{}); ok {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s (frequency: %v)\n", patternMap["pattern"], patternMap["frequency"])
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Learning completed\n")
	}
	return nil
}

func (c *CLI) runMemoryPatterns(cmd *cobra.Command, project, patternType, _ string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_read tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_read", map[string]interface{}{
		"operation": "get_patterns",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": project,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to get patterns: %w", err)
	}

	// Format output
	if patterns, ok := result["patterns"].([]interface{}); ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Patterns detected in %s:\n", project)
		for _, pattern := range patterns {
			if patternMap, ok := pattern.(map[string]interface{}); ok {
				if patternType == "" || patternMap["type"] == patternType {
					fmt.Fprintf(cmd.OutOrStdout(), "\n[%s] %s\n", patternMap["type"], patternMap["name"])
					fmt.Fprintf(cmd.OutOrStdout(), "  Frequency: %v\n", patternMap["frequency"])
					fmt.Fprintf(cmd.OutOrStdout(), "  Confidence: %v\n", patternMap["confidence"])
				}
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "No patterns found\n")
	}
	return nil
}

func (c *CLI) runMemorySuggest(cmd *cobra.Command, feature, project, _ string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Build context for suggestions
	currentContext := "Working on project " + project
	if feature != "" {
		currentContext = fmt.Sprintf("Working on feature '%s' in project %s", feature, project)
	}

	// Create request for memory_intelligence tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_intelligence", map[string]interface{}{
		"operation": "suggest_related",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository":      project,
			"session_id":      fmt.Sprintf("cli-suggest-%d", time.Now().Unix()),
			"current_context": currentContext,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to get suggestions: %w", err)
	}

	// Format output
	if suggestions, ok := result["suggestions"].([]interface{}); ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Suggestions for %s:\n", currentContext)
		for i, suggestion := range suggestions {
			if suggMap, ok := suggestion.(map[string]interface{}); ok {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%d. %s\n", i+1, suggMap["title"])
				fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", suggMap["description"])
				if confidence, ok := suggMap["confidence"]; ok {
					fmt.Fprintf(cmd.OutOrStdout(), "   Confidence: %.2f\n", confidence)
				}
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "No suggestions available\n")
	}
	return nil
}

func (c *CLI) runMemoryInsights(cmd *cobra.Command, topic, project string, crossProject bool, _ string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Determine scope and repository
	scope := "single"
	repository := project
	if crossProject {
		scope = "cross_repo"
		if repository == "" {
			repository = constants.RepositoryGlobal
		}
	}

	// Create request for memory_intelligence tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_intelligence", map[string]interface{}{
		"operation": "auto_insights",
		"scope":     scope,
		"options": map[string]interface{}{
			"repository": repository,
			"session_id": fmt.Sprintf("cli-insights-%d", time.Now().Unix()),
			"context":    topic,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to generate insights: %w", err)
	}

	// Format output
	if insights, ok := result["insights"].([]interface{}); ok {
		title := "Insights"
		if topic != "" {
			title = fmt.Sprintf("Insights for '%s'", topic)
		}
		if crossProject {
			title += " (cross-project)"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", title)

		for _, insight := range insights {
			if insightMap, ok := insight.(map[string]interface{}); ok {
				fmt.Fprintf(cmd.OutOrStdout(), "\n• %s\n", insightMap["insight"])
				if evidence, ok := insightMap["evidence"]; ok {
					fmt.Fprintf(cmd.OutOrStdout(), "  Evidence: %v\n", evidence)
				}
				if recommendation, ok := insightMap["recommendation"]; ok {
					fmt.Fprintf(cmd.OutOrStdout(), "  Recommendation: %v\n", recommendation)
				}
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "No insights available\n")
	}
	return nil
}

func (c *CLI) runMemoryCompare(cmd *cobra.Command, projects []string, aspect, _ string) error {
	// Get MCP client
	mcpClient := c.getMCPClient()
	if mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Create request for memory_analyze tool
	ctx := c.getContext()
	result, err := mcpClient.CallMCPTool(ctx, "memory_analyze", map[string]interface{}{
		"operation": "cross_repo_insights",
		"scope":     "cross_repo",
		"options": map[string]interface{}{
			"repository": strings.Join(projects, ","),
			"session_id": fmt.Sprintf("cli-compare-%d", time.Now().Unix()),
			"aspect":     aspect,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to compare projects: %w", err)
	}

	// Format output
	fmt.Fprintf(cmd.OutOrStdout(), "Comparison of projects: %s\n", strings.Join(projects, ", "))
	if aspect != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Aspect: %s\n", aspect)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	if comparison, ok := result["comparison"].(map[string]interface{}); ok {
		// Show similarities
		if similarities, ok := comparison["similarities"].([]interface{}); ok && len(similarities) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Similarities:\n")
			for _, sim := range similarities {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %v\n", sim)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
		}

		// Show differences
		if differences, ok := comparison["differences"].([]interface{}); ok && len(differences) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Differences:\n")
			for _, diff := range differences {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %v\n", diff)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
		}

		// Show recommendations
		if recommendations, ok := comparison["recommendations"].([]interface{}); ok && len(recommendations) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Recommendations:\n")
			for _, rec := range recommendations {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %v\n", rec)
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Comparison completed\n")
	}
	return nil
}
