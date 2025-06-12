// Package tools provides the clean 4-tool architecture for MCP Memory Server.
// This replaces the 9 fragmented tools with 4 logical tools with clear boundaries.
package tools

import (
	"fmt"
	"lerian-mcp-memory/internal/operations"
)

// StoreOperations defines all data persistence operations using clear names
type StoreOperations string

const (
	// OpStoreContent stores new content/memory chunks
	OpStoreContent StoreOperations = operations.StoreContent
	
	// OpStoreDecision stores architectural or design decisions
	OpStoreDecision StoreOperations = operations.StoreDecision
	
	// OpStoreInsight stores generated insights
	OpStoreInsight StoreOperations = operations.StoreInsight
	
	// OpStorePattern stores detected patterns
	OpStorePattern StoreOperations = operations.StorePattern
	
	// OpStoreRelationship creates relationships between content
	OpStoreRelationship StoreOperations = operations.StoreRelationship
	
	// OpUpdateContent updates existing content
	OpUpdateContent StoreOperations = operations.UpdateExistingContent
	
	// OpUpdateMetadata updates content metadata only
	OpUpdateMetadata StoreOperations = operations.UpdateContentMetadata
	
	// OpDeleteContent removes content
	OpDeleteContent StoreOperations = operations.DeleteOldContent
	
	// OpDeleteRelationship removes content relationships
	OpDeleteRelationship StoreOperations = operations.DeleteRelationship
	
	// OpExpireContent removes stale content
	OpExpireContent StoreOperations = operations.ExpireStaleContent
)

// RetrieveOperations defines all data retrieval operations using clear names
type RetrieveOperations string

const (
	// OpSearchContent performs semantic search across content
	OpSearchContent RetrieveOperations = operations.SearchContent
	
	// OpGetContentByID retrieves specific content by ID
	OpGetContentByID RetrieveOperations = operations.GetContentByID
	
	// OpGetContentByProject gets all content in a project
	OpGetContentByProject RetrieveOperations = operations.GetContentByProject
	
	// OpGetContentBySession gets session-specific content
	OpGetContentBySession RetrieveOperations = operations.GetContentBySession
	
	// OpFindSimilarContent finds content similar to given text
	OpFindSimilarContent RetrieveOperations = operations.FindSimilarContent
	
	// OpFindRelatedContent finds connected content via relationships
	OpFindRelatedContent RetrieveOperations = operations.FindRelatedContent
	
	// OpGetContentHistory retrieves content version history
	OpGetContentHistory RetrieveOperations = operations.GetContentHistory
	
	// OpGetContentRelationships retrieves relationships for content
	OpGetContentRelationships RetrieveOperations = operations.GetContentRelationships
	
	// OpExploreContentGraph traverses relationship graph
	OpExploreContentGraph RetrieveOperations = operations.ExploreContentGraph
	
	// OpListProjectContent lists content with pagination
	OpListProjectContent RetrieveOperations = operations.ListProjectContent
)

// AnalyzeOperations defines all analysis and intelligence operations using clear names
type AnalyzeOperations string

const (
	// OpDetectContentPatterns identifies patterns in content and behavior
	OpDetectContentPatterns AnalyzeOperations = operations.DetectContentPatterns
	
	// OpAnalyzeContentQuality analyzes content quality and completeness
	OpAnalyzeContentQuality AnalyzeOperations = operations.AnalyzeContentQuality
	
	// OpFindContentRelationships discovers relationships between content
	OpFindContentRelationships AnalyzeOperations = operations.FindContentRelationships
	
	// OpGenerateContentInsights generates insights from content analysis
	OpGenerateContentInsights AnalyzeOperations = operations.GenerateContentInsights
	
	// OpRecommendRelatedContent suggests content that might be of interest
	OpRecommendRelatedContent AnalyzeOperations = operations.RecommendRelatedContent
	
	// OpDetectContentConflicts identifies conflicting information
	OpDetectContentConflicts AnalyzeOperations = operations.DetectContentConflicts
	
	// OpAnalyzeDecisionConflicts checks for conflicting decisions
	OpAnalyzeDecisionConflicts AnalyzeOperations = operations.AnalyzeDecisionConflicts
	
	// OpPredictContentTrends forecasts content evolution
	OpPredictContentTrends AnalyzeOperations = operations.PredictContentTrends
	
	// OpIdentifyKnowledgeGaps finds missing information areas
	OpIdentifyKnowledgeGaps AnalyzeOperations = operations.IdentifyKnowledgeGaps
	
	// OpAnalyzeUsagePatterns studies how content is accessed and used
	OpAnalyzeUsagePatterns AnalyzeOperations = operations.AnalyzeUsagePatterns
)

// SystemOperations defines all system and administrative operations using clear names
type SystemOperations string

const (
	// OpCheckSystemHealth verifies system is functioning properly
	OpCheckSystemHealth SystemOperations = operations.CheckSystemHealth
	
	// OpExportProjectData exports all project data to external formats
	OpExportProjectData SystemOperations = operations.ExportProjectData
	
	// OpImportProjectData imports project data from external sources
	OpImportProjectData SystemOperations = operations.ImportProjectData
	
	// OpGenerateContentCitation creates properly formatted citations
	OpGenerateContentCitation SystemOperations = operations.GenerateContentCitation
	
	// OpValidateDataIntegrity checks data consistency and integrity
	OpValidateDataIntegrity SystemOperations = operations.ValidateDataIntegrity
	
	// OpCalculateSystemMetrics computes detailed system metrics
	OpCalculateSystemMetrics SystemOperations = operations.CalculateSystemMetrics
	
	// OpMonitorSystemPerformance tracks system performance
	OpMonitorSystemPerformance SystemOperations = operations.MonitorSystemPerformance
	
	// OpCreateUserSession starts new user session
	OpCreateUserSession SystemOperations = operations.CreateUserSession
	
	// OpUpdateSessionAccess updates session permissions
	OpUpdateSessionAccess SystemOperations = operations.UpdateSessionAccess
	
	// OpCleanupExpiredSessions removes old sessions
	OpCleanupExpiredSessions SystemOperations = operations.CleanupExpiredSessions
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
	// First, convert any deprecated operation names to clear names
	clearOp := operations.GetClearOperationName(operation)
	
	// Store operations
	storeOps := map[string]bool{
		string(OpStoreContent):      true,
		string(OpStoreDecision):     true,
		string(OpStoreInsight):      true,
		string(OpStorePattern):      true,
		string(OpStoreRelationship): true,
		string(OpUpdateContent):     true,
		string(OpUpdateMetadata):    true,
		string(OpDeleteContent):     true,
		string(OpDeleteRelationship): true,
		string(OpExpireContent):     true,
	}
	
	// Retrieve operations
	retrieveOps := map[string]bool{
		string(OpSearchContent):           true,
		string(OpGetContentByID):          true,
		string(OpGetContentByProject):     true,
		string(OpGetContentBySession):     true,
		string(OpFindSimilarContent):      true,
		string(OpFindRelatedContent):      true,
		string(OpGetContentHistory):       true,
		string(OpGetContentRelationships): true,
		string(OpExploreContentGraph):     true,
		string(OpListProjectContent):      true,
	}
	
	// Analyze operations
	analyzeOps := map[string]bool{
		string(OpDetectContentPatterns):    true,
		string(OpAnalyzeContentQuality):    true,
		string(OpFindContentRelationships): true,
		string(OpGenerateContentInsights):  true,
		string(OpRecommendRelatedContent):  true,
		string(OpDetectContentConflicts):   true,
		string(OpAnalyzeDecisionConflicts): true,
		string(OpPredictContentTrends):     true,
		string(OpIdentifyKnowledgeGaps):    true,
		string(OpAnalyzeUsagePatterns):     true,
	}
	
	// System operations
	systemOps := map[string]bool{
		string(OpCheckSystemHealth):         true,
		string(OpExportProjectData):         true,
		string(OpImportProjectData):         true,
		string(OpGenerateContentCitation):   true,
		string(OpValidateDataIntegrity):     true,
		string(OpCalculateSystemMetrics):    true,
		string(OpMonitorSystemPerformance):  true,
		string(OpCreateUserSession):         true,
		string(OpUpdateSessionAccess):       true,
		string(OpCleanupExpiredSessions):    true,
	}
	
	if storeOps[clearOp] {
		return ToolMemoryStore, nil
	}
	if retrieveOps[clearOp] {
		return ToolMemoryRetrieve, nil
	}
	if analyzeOps[clearOp] {
		return ToolMemoryAnalyze, nil
	}
	if systemOps[clearOp] {
		return ToolMemorySystem, nil
	}
	
	return "", fmt.Errorf("unknown operation: %s (clear name: %s)", operation, clearOp)
}

// GetOperationsForTool returns all operations handled by a tool
func GetOperationsForTool(tool ToolName) []string {
	switch tool {
	case ToolMemoryStore:
		return []string{
			string(OpStoreContent),
			string(OpStoreDecision),
			string(OpStoreInsight),
			string(OpStorePattern),
			string(OpStoreRelationship),
			string(OpUpdateContent),
			string(OpUpdateMetadata),
			string(OpDeleteContent),
			string(OpDeleteRelationship),
			string(OpExpireContent),
		}
	case ToolMemoryRetrieve:
		return []string{
			string(OpSearchContent),
			string(OpGetContentByID),
			string(OpGetContentByProject),
			string(OpGetContentBySession),
			string(OpFindSimilarContent),
			string(OpFindRelatedContent),
			string(OpGetContentHistory),
			string(OpGetContentRelationships),
			string(OpExploreContentGraph),
			string(OpListProjectContent),
		}
	case ToolMemoryAnalyze:
		return []string{
			string(OpDetectContentPatterns),
			string(OpAnalyzeContentQuality),
			string(OpFindContentRelationships),
			string(OpGenerateContentInsights),
			string(OpRecommendRelatedContent),
			string(OpDetectContentConflicts),
			string(OpAnalyzeDecisionConflicts),
			string(OpPredictContentTrends),
			string(OpIdentifyKnowledgeGaps),
			string(OpAnalyzeUsagePatterns),
		}
	case ToolMemorySystem:
		return []string{
			string(OpCheckSystemHealth),
			string(OpExportProjectData),
			string(OpImportProjectData),
			string(OpGenerateContentCitation),
			string(OpValidateDataIntegrity),
			string(OpCalculateSystemMetrics),
			string(OpMonitorSystemPerformance),
			string(OpCreateUserSession),
			string(OpUpdateSessionAccess),
			string(OpCleanupExpiredSessions),
		}
	default:
		return []string{}
	}
}

// IsWriteOperation returns true if the operation modifies data
func IsWriteOperation(operation string) bool {
	// Convert to clear operation name first
	clearOp := operations.GetClearOperationName(operation)
	
	writeOps := map[string]bool{
		string(OpStoreContent):      true,
		string(OpStoreDecision):     true,
		string(OpStoreInsight):      true,
		string(OpStorePattern):      true,
		string(OpStoreRelationship): true,
		string(OpUpdateContent):     true,
		string(OpUpdateMetadata):    true,
		string(OpDeleteContent):     true,
		string(OpDeleteRelationship): true,
		string(OpExpireContent):     true,
		string(OpImportProjectData): true,
		string(OpCreateUserSession): true,
		string(OpUpdateSessionAccess): true,
	}
	
	return writeOps[clearOp]
}

// GetOperationDescription returns a human-readable description of the operation
func GetOperationDescription(operation string) string {
	// Delegate to the operations package for consistent descriptions
	return operations.GetOperationDescription(operation)
}