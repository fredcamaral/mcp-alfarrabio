// Package tools provides the clean 4-tool architecture for MCP Memory Server.
// This replaces the 9 fragmented tools with 4 logical tools with clear boundaries.
package tools

// StoreOperations defines all data persistence operations
type StoreOperations string

const (
	// OpStoreContent stores new content/memory chunks
	OpStoreContent StoreOperations = "store_content"
	
	// OpStoreDecision stores architectural or design decisions
	OpStoreDecision StoreOperations = "store_decision"
	
	// OpUpdateContent updates existing content
	OpUpdateContent StoreOperations = "update_content"
	
	// OpDeleteContent removes content
	OpDeleteContent StoreOperations = "delete_content"
	
	// OpCreateThread creates a new conversation thread
	OpCreateThread StoreOperations = "create_thread"
	
	// OpCreateRelation creates relationships between content
	OpCreateRelation StoreOperations = "create_relationship"
)

// RetrieveOperations defines all data retrieval operations
type RetrieveOperations string

const (
	// OpSearch performs semantic search across content
	OpSearch RetrieveOperations = "search"
	
	// OpGetContent retrieves specific content by ID
	OpGetContent RetrieveOperations = "get_content"
	
	// OpFindSimilar finds content similar to given text
	OpFindSimilar RetrieveOperations = "find_similar"
	
	// OpGetThreads retrieves conversation threads
	OpGetThreads RetrieveOperations = "get_threads"
	
	// OpGetRelationships retrieves relationships between content
	OpGetRelationships RetrieveOperations = "get_relationships"
	
	// OpGetHistory retrieves content history/changes
	OpGetHistory RetrieveOperations = "get_history"
)

// AnalyzeOperations defines all analysis and intelligence operations
type AnalyzeOperations string

const (
	// OpDetectPatterns identifies patterns in content/behavior
	OpDetectPatterns AnalyzeOperations = "detect_patterns"
	
	// OpSuggestRelated suggests related content or context
	OpSuggestRelated AnalyzeOperations = "suggest_related"
	
	// OpAnalyzeQuality analyzes content quality and completeness
	OpAnalyzeQuality AnalyzeOperations = "analyze_quality"
	
	// OpDetectConflicts identifies conflicting information
	OpDetectConflicts AnalyzeOperations = "detect_conflicts"
	
	// OpGenerateInsights generates insights from data patterns
	OpGenerateInsights AnalyzeOperations = "generate_insights"
	
	// OpPredictTrends predicts future trends based on data
	OpPredictTrends AnalyzeOperations = "predict_trends"
)

// SystemOperations defines all system and administrative operations
type SystemOperations string

const (
	// OpHealth checks system health and status
	OpHealth SystemOperations = "health"
	
	// OpExportProject exports project data
	OpExportProject SystemOperations = "export_project"
	
	// OpImportProject imports project data
	OpImportProject SystemOperations = "import_project"
	
	// OpGenerateCitation generates citations for content
	OpGenerateCitation SystemOperations = "generate_citation"
	
	// OpValidateIntegrity validates data integrity
	OpValidateIntegrity SystemOperations = "validate_integrity"
	
	// OpGetMetrics retrieves system metrics
	OpGetMetrics SystemOperations = "get_metrics"
)

// ToolName represents the 4 clean tool names
type ToolName string

const (
	// ToolMemoryStore handles all data persistence operations
	ToolMemoryStore ToolName = "memory_store"
	
	// ToolMemoryRetrieve handles all data retrieval operations
	ToolMemoryRetrieve ToolName = "memory_retrieve"
	
	// ToolMemoryAnalyze handles all analysis and intelligence operations
	ToolMemoryAnalyze ToolName = "memory_analyze"
	
	// ToolMemorySystem handles all system and administrative operations
	ToolMemorySystem ToolName = "memory_system"
)

// GetToolForOperation returns which tool handles a given operation
func GetToolForOperation(operation string) (ToolName, error) {
	// Store operations
	storeOps := map[string]bool{
		string(OpStoreContent):    true,
		string(OpStoreDecision):   true,
		string(OpUpdateContent):   true,
		string(OpDeleteContent):   true,
		string(OpCreateThread):    true,
		string(OpCreateRelation):  true,
	}
	
	// Retrieve operations
	retrieveOps := map[string]bool{
		string(OpSearch):            true,
		string(OpGetContent):        true,
		string(OpFindSimilar):       true,
		string(OpGetThreads):        true,
		string(OpGetRelationships):  true,
		string(OpGetHistory):        true,
	}
	
	// Analyze operations
	analyzeOps := map[string]bool{
		string(OpDetectPatterns):    true,
		string(OpSuggestRelated):    true,
		string(OpAnalyzeQuality):    true,
		string(OpDetectConflicts):   true,
		string(OpGenerateInsights):  true,
		string(OpPredictTrends):     true,
	}
	
	// System operations
	systemOps := map[string]bool{
		string(OpHealth):            true,
		string(OpExportProject):     true,
		string(OpImportProject):     true,
		string(OpGenerateCitation):  true,
		string(OpValidateIntegrity): true,
		string(OpGetMetrics):        true,
	}
	
	if storeOps[operation] {
		return ToolMemoryStore, nil
	}
	if retrieveOps[operation] {
		return ToolMemoryRetrieve, nil
	}
	if analyzeOps[operation] {
		return ToolMemoryAnalyze, nil
	}
	if systemOps[operation] {
		return ToolMemorySystem, nil
	}
	
	return "", fmt.Errorf("unknown operation: %s", operation)
}

// GetOperationsForTool returns all operations handled by a tool
func GetOperationsForTool(tool ToolName) []string {
	switch tool {
	case ToolMemoryStore:
		return []string{
			string(OpStoreContent),
			string(OpStoreDecision),
			string(OpUpdateContent),
			string(OpDeleteContent),
			string(OpCreateThread),
			string(OpCreateRelation),
		}
	case ToolMemoryRetrieve:
		return []string{
			string(OpSearch),
			string(OpGetContent),
			string(OpFindSimilar),
			string(OpGetThreads),
			string(OpGetRelationships),
			string(OpGetHistory),
		}
	case ToolMemoryAnalyze:
		return []string{
			string(OpDetectPatterns),
			string(OpSuggestRelated),
			string(OpAnalyzeQuality),
			string(OpDetectConflicts),
			string(OpGenerateInsights),
			string(OpPredictTrends),
		}
	case ToolMemorySystem:
		return []string{
			string(OpHealth),
			string(OpExportProject),
			string(OpImportProject),
			string(OpGenerateCitation),
			string(OpValidateIntegrity),
			string(OpGetMetrics),
		}
	default:
		return []string{}
	}
}

// IsWriteOperation returns true if the operation modifies data
func IsWriteOperation(operation string) bool {
	writeOps := map[string]bool{
		string(OpStoreContent):     true,
		string(OpStoreDecision):    true,
		string(OpUpdateContent):    true,
		string(OpDeleteContent):    true,
		string(OpCreateThread):     true,
		string(OpCreateRelation):   true,
		string(OpImportProject):    true,
	}
	
	return writeOps[operation]
}

// GetOperationDescription returns a human-readable description of the operation
func GetOperationDescription(operation string) string {
	descriptions := map[string]string{
		// Store operations
		string(OpStoreContent):    "Store new content or memory chunks in the system",
		string(OpStoreDecision):   "Store architectural or design decisions for future reference",
		string(OpUpdateContent):   "Update existing content with new information",
		string(OpDeleteContent):   "Remove content from the system",
		string(OpCreateThread):    "Create a new conversation thread for organizing related content",
		string(OpCreateRelation):  "Create relationships between different pieces of content",
		
		// Retrieve operations
		string(OpSearch):            "Perform semantic search across all stored content",
		string(OpGetContent):        "Retrieve specific content by its unique identifier",
		string(OpFindSimilar):       "Find content similar to the provided text or context",
		string(OpGetThreads):        "Retrieve conversation threads and their metadata",
		string(OpGetRelationships):  "Retrieve relationships between content items",
		string(OpGetHistory):        "Retrieve the change history for content items",
		
		// Analyze operations
		string(OpDetectPatterns):    "Identify patterns in content, behavior, or usage",
		string(OpSuggestRelated):    "Suggest related content or context based on current focus",
		string(OpAnalyzeQuality):    "Analyze the quality and completeness of stored content",
		string(OpDetectConflicts):   "Identify conflicting information or decisions",
		string(OpGenerateInsights):  "Generate insights from data patterns and relationships",
		string(OpPredictTrends):     "Predict future trends based on historical data",
		
		// System operations
		string(OpHealth):            "Check system health, status, and performance metrics",
		string(OpExportProject):     "Export all project data in a portable format",
		string(OpImportProject):     "Import project data from external sources",
		string(OpGenerateCitation):  "Generate proper citations for stored content",
		string(OpValidateIntegrity): "Validate the integrity of stored data and relationships",
		string(OpGetMetrics):        "Retrieve detailed system and usage metrics",
	}
	
	if desc, exists := descriptions[operation]; exists {
		return desc
	}
	
	return "Unknown operation"
}

import "fmt"