// Package operations provides centralized operation name definitions
// for the refactored MCP Memory Server architecture.
//
// This package replaces cryptic operation names with clear, self-documenting
// names that immediately explain what each operation does.
package operations

import "fmt"

// Clear, descriptive operation names that explain what they do
// These replace the cryptic names from the old system with intuitive names.

// StoreOperations - All data persistence operations
// These operations clearly indicate that data is being stored or modified
const (
	// Content storage operations
	StoreContent      = "store_content"      // Store new content item
	StoreDecision     = "store_decision"     // Store architectural decision
	StoreInsight      = "store_insight"      // Store generated insight
	StorePattern      = "store_pattern"      // Store detected pattern
	StoreRelationship = "store_relationship" // Store content relationship

	// Content modification operations
	UpdateExistingContent        = "update_existing_content"        // Update existing content
	UpdateContentMetadata        = "update_content_metadata"        // Update content metadata only
	UpdateContentSummary         = "update_content_summary"         // Update content summary
	UpdateRelationshipConfidence = "update_relationship_confidence" // Update relationship confidence

	// Content removal operations
	DeleteOldContent       = "delete_old_content"        // Delete specific content
	DeleteContentByProject = "delete_content_by_project" // Delete all project content
	DeleteRelationship     = "delete_relationship"       // Delete content relationship

	// Maintenance operations
	ExpireStaleContent    = "expire_stale_content"    // Remove outdated content
	ArchiveOldContent     = "archive_old_content"     // Archive old but valuable content
	CompactContentStorage = "compact_content_storage" // Optimize storage efficiency
)

// RetrieveOperations - All data access and search operations
// These operations clearly indicate that data is being retrieved or searched
const (
	// Basic content retrieval
	SearchContent       = "search_content"         // Search content by query
	GetContentByID      = "get_content_by_id"      // Get specific content by ID
	GetContentByProject = "get_content_by_project" // Get all content in project
	GetContentBySession = "get_content_by_session" // Get session-specific content
	GetContentByType    = "get_content_by_type"    // Get content by type

	// Advanced search operations
	FindSimilarContent     = "find_similar_content"      // Find semantically similar content
	FindRelatedContent     = "find_related_content"      // Find connected content via relationships
	FindContentByTags      = "find_content_by_tags"      // Find content with specific tags
	FindContentByTimeRange = "find_content_by_timerange" // Find content in date range

	// Content history and versioning
	GetContentHistory  = "get_content_history"  // Get content version history
	GetContentVersions = "get_content_versions" // Get all versions of content
	GetLatestContent   = "get_latest_content"   // Get most recent content

	// Relationship and graph operations
	GetContentRelationships = "get_content_relationships" // Get relationships for content
	ExploreContentGraph     = "explore_content_graph"     // Traverse relationship graph
	GetRelationshipsByType  = "get_relationships_by_type" // Get specific relationship types

	// List and browse operations
	ListContentTypes    = "list_content_types"     // List available content types
	ListProjectContent  = "list_project_content"   // List content with pagination
	ListRecentContent   = "list_recent_content"    // List recently created/updated
	BrowseContentByDate = "browse_content_by_date" // Browse content chronologically
)

// AnalyzeOperations - All analysis and intelligence operations
// These operations clearly indicate the type of analysis being performed
const (
	// Pattern detection and analysis
	DetectContentPatterns   = "detect_content_patterns"   // Find patterns in content
	AnalyzePatternTrends    = "analyze_pattern_trends"    // Analyze how patterns change over time
	ComparePatternFrequency = "compare_pattern_frequency" // Compare pattern occurrence rates
	PredictPatternEvolution = "predict_pattern_evolution" // Predict how patterns will evolve

	// Content quality analysis
	AnalyzeContentQuality       = "analyze_content_quality"       // Assess content quality metrics
	EvaluateContentCompleteness = "evaluate_content_completeness" // Check if content is complete
	AssessContentClarity        = "assess_content_clarity"        // Evaluate how clear content is
	CheckContentRelevance       = "check_content_relevance"       // Verify content relevance
	ValidateContentAccuracy     = "validate_content_accuracy"     // Check content accuracy

	// Relationship and connection analysis
	FindContentRelationships    = "find_content_relationships"    // Discover relationships between content
	AnalyzeRelationshipStrength = "analyze_relationship_strength" // Measure relationship confidence
	MapContentDependencies      = "map_content_dependencies"      // Map content dependencies
	IdentifyContentClusters     = "identify_content_clusters"     // Find related content groups

	// Insight generation
	GenerateContentInsights    = "generate_content_insights"    // Create insights from content
	SuggestContentImprovements = "suggest_content_improvements" // Recommend content enhancements
	RecommendRelatedContent    = "recommend_related_content"    // Suggest related content
	IdentifyKnowledgeGaps      = "identify_knowledge_gaps"      // Find missing information

	// Conflict detection and resolution
	DetectContentConflicts      = "detect_content_conflicts"      // Find conflicting information
	AnalyzeDecisionConflicts    = "analyze_decision_conflicts"    // Check for conflicting decisions
	ValidateContentConsistency  = "validate_content_consistency"  // Ensure content consistency
	ResolveInformationConflicts = "resolve_information_conflicts" // Suggest conflict resolutions

	// Trend analysis and prediction
	AnalyzeUsagePatterns   = "analyze_usage_patterns"   // Study how content is used
	PredictContentTrends   = "predict_content_trends"   // Forecast content evolution
	IdentifyEmergingTopics = "identify_emerging_topics" // Find new topic trends
	TrackContentPopularity = "track_content_popularity" // Monitor content engagement
)

// SystemOperations - All administrative and system operations
// These operations clearly indicate system-level functionality
const (
	// Health and monitoring
	CheckSystemHealth        = "check_system_health"        // Verify system is healthy
	MonitorSystemPerformance = "monitor_system_performance" // Track system performance
	ValidateSystemIntegrity  = "validate_system_integrity"  // Check system data integrity
	DiagnoseSystemIssues     = "diagnose_system_issues"     // Identify system problems

	// Data management
	ExportProjectData     = "export_project_data"     // Export project to file
	ImportProjectData     = "import_project_data"     // Import project from file
	BackupProjectContent  = "backup_project_content"  // Create project backup
	RestoreProjectContent = "restore_project_content" // Restore from backup

	// Data integrity and maintenance
	ValidateDataIntegrity = "validate_data_integrity" // Check data consistency
	RepairDataCorruption  = "repair_data_corruption"  // Fix corrupted data
	OptimizeDataStorage   = "optimize_data_storage"   // Improve storage efficiency
	CleanupOrphanedData   = "cleanup_orphaned_data"   // Remove unused data

	// System configuration
	UpdateSystemConfiguration = "update_system_configuration" // Modify system settings
	ResetSystemSettings       = "reset_system_settings"       // Restore default settings
	ConfigureRetentionPolicy  = "configure_retention_policy"  // Set data retention rules
	ManageSystemPermissions   = "manage_system_permissions"   // Control access permissions

	// Citation and referencing
	GenerateContentCitation    = "generate_content_citation"    // Create citation for content
	FormatCitationStyle        = "format_citation_style"        // Format citation in specific style
	ValidateCitationFormat     = "validate_citation_format"     // Check citation formatting
	ExportCitationBibliography = "export_citation_bibliography" // Export bibliography

	// System metrics and reporting
	GenerateUsageReport      = "generate_usage_report"      // Create usage statistics
	CalculateSystemMetrics   = "calculate_system_metrics"   // Compute performance metrics
	ExportSystemLogs         = "export_system_logs"         // Export system log files
	AnalyzeSystemPerformance = "analyze_system_performance" // Analyze performance data

	// Session and access management
	CreateUserSession      = "create_user_session"      // Start new user session
	UpdateSessionAccess    = "update_session_access"    // Update session permissions
	ValidateSessionToken   = "validate_session_token"   // Verify session validity
	CleanupExpiredSessions = "cleanup_expired_sessions" // Remove old sessions
)

// DeprecatedOperations - Old cryptic operation names that are being replaced
// These serve as a mapping reference during the transition period
var DeprecatedOperations = map[string]string{
	// Old cryptic names → New clear names
	"decay_management":          ExpireStaleContent,
	"mark_refreshed":            UpdateContentMetadata,
	"traverse_graph":            ExploreContentGraph,
	"auto_detect_relationships": FindContentRelationships,
	"memory_create":             StoreContent,
	"memory_read":               SearchContent,
	"memory_update":             UpdateExistingContent,
	"memory_delete":             DeleteOldContent,
	"memory_search":             SearchContent,
	"memory_analyze":            AnalyzeContentQuality,
	"memory_intelligence":       GenerateContentInsights,
	"memory_system":             CheckSystemHealth,
	"memory_transfer":           ExportProjectData,
	"get_similar":               FindSimilarContent,
	"find_related":              FindRelatedContent,
	"quality_check":             AnalyzeContentQuality,
	"pattern_detect":            DetectContentPatterns,
	"insight_gen":               GenerateContentInsights,
	"conflict_check":            DetectContentConflicts,
	"health_status":             CheckSystemHealth,
	"export_data":               ExportProjectData,
	"import_data":               ImportProjectData,
	"cite_content":              GenerateContentCitation,
}

// GetClearOperationName returns the clear operation name for a given input
// If the input is already clear, it returns the input unchanged
// If the input is a deprecated cryptic name, it returns the clear equivalent
func GetClearOperationName(operation string) string {
	if clearName, exists := DeprecatedOperations[operation]; exists {
		return clearName
	}
	return operation
}

// IsValidOperation checks if an operation name is valid (either new clear name or deprecated)
func IsValidOperation(operation string) bool {
	// Check if it's a deprecated operation
	if _, exists := DeprecatedOperations[operation]; exists {
		return true
	}

	// Check if it's a valid clear operation name
	return isValidClearOperation(operation)
}

// isValidClearOperation checks if the operation is a valid clear operation name
func isValidClearOperation(operation string) bool {
	validOperations := []string{
		// Store operations
		StoreContent, StoreDecision, StoreInsight, StorePattern, StoreRelationship,
		UpdateExistingContent, UpdateContentMetadata, UpdateContentSummary, UpdateRelationshipConfidence,
		DeleteOldContent, DeleteContentByProject, DeleteRelationship,
		ExpireStaleContent, ArchiveOldContent, CompactContentStorage,

		// Retrieve operations
		SearchContent, GetContentByID, GetContentByProject, GetContentBySession, GetContentByType,
		FindSimilarContent, FindRelatedContent, FindContentByTags, FindContentByTimeRange,
		GetContentHistory, GetContentVersions, GetLatestContent,
		GetContentRelationships, ExploreContentGraph, GetRelationshipsByType,
		ListContentTypes, ListProjectContent, ListRecentContent, BrowseContentByDate,

		// Analyze operations
		DetectContentPatterns, AnalyzePatternTrends, ComparePatternFrequency, PredictPatternEvolution,
		AnalyzeContentQuality, EvaluateContentCompleteness, AssessContentClarity, CheckContentRelevance, ValidateContentAccuracy,
		FindContentRelationships, AnalyzeRelationshipStrength, MapContentDependencies, IdentifyContentClusters,
		GenerateContentInsights, SuggestContentImprovements, RecommendRelatedContent, IdentifyKnowledgeGaps,
		DetectContentConflicts, AnalyzeDecisionConflicts, ValidateContentConsistency, ResolveInformationConflicts,
		AnalyzeUsagePatterns, PredictContentTrends, IdentifyEmergingTopics, TrackContentPopularity,

		// System operations
		CheckSystemHealth, MonitorSystemPerformance, ValidateSystemIntegrity, DiagnoseSystemIssues,
		ExportProjectData, ImportProjectData, BackupProjectContent, RestoreProjectContent,
		ValidateDataIntegrity, RepairDataCorruption, OptimizeDataStorage, CleanupOrphanedData,
		UpdateSystemConfiguration, ResetSystemSettings, ConfigureRetentionPolicy, ManageSystemPermissions,
		GenerateContentCitation, FormatCitationStyle, ValidateCitationFormat, ExportCitationBibliography,
		GenerateUsageReport, CalculateSystemMetrics, ExportSystemLogs, AnalyzeSystemPerformance,
		CreateUserSession, UpdateSessionAccess, ValidateSessionToken, CleanupExpiredSessions,
	}

	for _, validOp := range validOperations {
		if operation == validOp {
			return true
		}
	}
	return false
}

// GetOperationDescription returns a human-readable description of what the operation does
func GetOperationDescription(operation string) string {
	// Convert to clear name first
	clearOp := GetClearOperationName(operation)

	descriptions := map[string]string{
		// Store operations
		StoreContent:                 "Store a new content item in the memory system",
		StoreDecision:                "Store an architectural or design decision with rationale",
		StoreInsight:                 "Store a generated insight or analysis result",
		StorePattern:                 "Store a detected pattern for future reference",
		StoreRelationship:            "Store a relationship between two content items",
		UpdateExistingContent:        "Update the content of an existing item",
		UpdateContentMetadata:        "Update only the metadata of existing content",
		UpdateContentSummary:         "Update the summary of existing content",
		UpdateRelationshipConfidence: "Update the confidence score of a relationship",
		DeleteOldContent:             "Delete a specific content item from storage",
		DeleteContentByProject:       "Delete all content associated with a project",
		DeleteRelationship:           "Remove a relationship between content items",
		ExpireStaleContent:           "Remove content that has become outdated",
		ArchiveOldContent:            "Move old but valuable content to archive",
		CompactContentStorage:        "Optimize storage by compacting data structures",

		// Retrieve operations
		SearchContent:           "Search for content using natural language queries",
		GetContentByID:          "Retrieve a specific content item by its unique ID",
		GetContentByProject:     "Get all content items belonging to a project",
		GetContentBySession:     "Get content items from a specific session",
		GetContentByType:        "Retrieve content items of a specific type",
		FindSimilarContent:      "Find content that is semantically similar to given content",
		FindRelatedContent:      "Find content connected through explicit relationships",
		FindContentByTags:       "Search for content items with specific tags",
		FindContentByTimeRange:  "Find content created or updated within a date range",
		GetContentHistory:       "Retrieve the version history of a content item",
		GetContentVersions:      "Get all available versions of content",
		GetLatestContent:        "Retrieve the most recently created or updated content",
		GetContentRelationships: "Get all relationships associated with content",
		ExploreContentGraph:     "Traverse the graph of content relationships",
		GetRelationshipsByType:  "Get relationships of a specific type",
		ListContentTypes:        "List all available content types in the system",
		ListProjectContent:      "List content items with pagination support",
		ListRecentContent:       "List recently created or modified content",
		BrowseContentByDate:     "Browse content organized by creation date",

		// Analyze operations
		DetectContentPatterns:       "Identify recurring patterns in content and behavior",
		AnalyzePatternTrends:        "Analyze how patterns change and evolve over time",
		ComparePatternFrequency:     "Compare how often different patterns occur",
		PredictPatternEvolution:     "Predict how patterns will develop in the future",
		AnalyzeContentQuality:       "Assess the quality metrics of content items",
		EvaluateContentCompleteness: "Check whether content provides complete information",
		AssessContentClarity:        "Evaluate how clear and understandable content is",
		CheckContentRelevance:       "Verify that content is relevant to its context",
		ValidateContentAccuracy:     "Check the accuracy of information in content",
		FindContentRelationships:    "Discover relationships between different content items",
		AnalyzeRelationshipStrength: "Measure the strength of relationships between content",
		MapContentDependencies:      "Create a map of how content items depend on each other",
		IdentifyContentClusters:     "Find groups of related content items",
		GenerateContentInsights:     "Generate insights and observations from content analysis",
		SuggestContentImprovements:  "Recommend ways to improve content quality",
		RecommendRelatedContent:     "Suggest content that might be of interest",
		IdentifyKnowledgeGaps:       "Find areas where information is missing",
		DetectContentConflicts:      "Identify conflicting information between content items",
		AnalyzeDecisionConflicts:    "Check for conflicts between different decisions",
		ValidateContentConsistency:  "Ensure content is consistent across the system",
		ResolveInformationConflicts: "Suggest ways to resolve information conflicts",
		AnalyzeUsagePatterns:        "Study how content is accessed and used",
		PredictContentTrends:        "Forecast how content topics will evolve",
		IdentifyEmergingTopics:      "Find new topics that are becoming popular",
		TrackContentPopularity:      "Monitor which content is most engaging",

		// System operations
		CheckSystemHealth:          "Verify that all system components are functioning properly",
		MonitorSystemPerformance:   "Track system performance metrics and statistics",
		ValidateSystemIntegrity:    "Check the integrity of system data and structures",
		DiagnoseSystemIssues:       "Identify and analyze system problems",
		ExportProjectData:          "Export all project data to external formats",
		ImportProjectData:          "Import project data from external sources",
		BackupProjectContent:       "Create a backup of project content",
		RestoreProjectContent:      "Restore project content from a backup",
		ValidateDataIntegrity:      "Check that stored data is consistent and uncorrupted",
		RepairDataCorruption:       "Fix any detected data corruption issues",
		OptimizeDataStorage:        "Improve the efficiency of data storage",
		CleanupOrphanedData:        "Remove data that is no longer referenced",
		UpdateSystemConfiguration:  "Modify system configuration settings",
		ResetSystemSettings:        "Restore system settings to default values",
		ConfigureRetentionPolicy:   "Set rules for how long data should be retained",
		ManageSystemPermissions:    "Control access permissions for system resources",
		GenerateContentCitation:    "Create properly formatted citations for content",
		FormatCitationStyle:        "Format citations in specific academic styles",
		ValidateCitationFormat:     "Check that citations follow proper formatting",
		ExportCitationBibliography: "Export a bibliography of content citations",
		GenerateUsageReport:        "Create reports on system and content usage",
		CalculateSystemMetrics:     "Compute detailed system performance metrics",
		ExportSystemLogs:           "Export system logs for analysis",
		AnalyzeSystemPerformance:   "Analyze system performance data for insights",
		CreateUserSession:          "Start a new user session for accessing content",
		UpdateSessionAccess:        "Update the access permissions for a session",
		ValidateSessionToken:       "Verify that a session token is valid",
		CleanupExpiredSessions:     "Remove sessions that have expired",
	}

	if desc, exists := descriptions[clearOp]; exists {
		return desc
	}

	return fmt.Sprintf("Perform operation: %s", clearOp)
}

// GetOperationCategory returns the category of the operation (store, retrieve, analyze, system)
func GetOperationCategory(operation string) string {
	clearOp := GetClearOperationName(operation)

	// Store operations
	storeOps := []string{
		StoreContent, StoreDecision, StoreInsight, StorePattern, StoreRelationship,
		UpdateExistingContent, UpdateContentMetadata, UpdateContentSummary, UpdateRelationshipConfidence,
		DeleteOldContent, DeleteContentByProject, DeleteRelationship,
		ExpireStaleContent, ArchiveOldContent, CompactContentStorage,
	}

	// Retrieve operations
	retrieveOps := []string{
		SearchContent, GetContentByID, GetContentByProject, GetContentBySession, GetContentByType,
		FindSimilarContent, FindRelatedContent, FindContentByTags, FindContentByTimeRange,
		GetContentHistory, GetContentVersions, GetLatestContent,
		GetContentRelationships, ExploreContentGraph, GetRelationshipsByType,
		ListContentTypes, ListProjectContent, ListRecentContent, BrowseContentByDate,
	}

	// Analyze operations
	analyzeOps := []string{
		DetectContentPatterns, AnalyzePatternTrends, ComparePatternFrequency, PredictPatternEvolution,
		AnalyzeContentQuality, EvaluateContentCompleteness, AssessContentClarity, CheckContentRelevance, ValidateContentAccuracy,
		FindContentRelationships, AnalyzeRelationshipStrength, MapContentDependencies, IdentifyContentClusters,
		GenerateContentInsights, SuggestContentImprovements, RecommendRelatedContent, IdentifyKnowledgeGaps,
		DetectContentConflicts, AnalyzeDecisionConflicts, ValidateContentConsistency, ResolveInformationConflicts,
		AnalyzeUsagePatterns, PredictContentTrends, IdentifyEmergingTopics, TrackContentPopularity,
	}

	// Check each category
	for _, op := range storeOps {
		if clearOp == op {
			return "store"
		}
	}
	for _, op := range retrieveOps {
		if clearOp == op {
			return "retrieve"
		}
	}
	for _, op := range analyzeOps {
		if clearOp == op {
			return "analyze"
		}
	}

	// Default to system if not found in other categories
	return "system"
}
