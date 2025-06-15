// Package constants provides shared string constants for the CLI application
package constants

// Output format constants
const (
	OutputFormatJSON  = "json"
	OutputFormatTable = "table"
	OutputFormatPlain = "plain"
)

// Export format constants
const (
	FormatYAML     = "yaml"
	FormatCSV      = "csv"
	FormatTSV      = "tsv"
	FormatXML      = "xml"
	FormatPDF      = "pdf"
	FormatHTML     = "html"
	FormatMarkdown = "markdown"
	FormatZip      = "zip"
)

// Workflow status constants
const (
	WorkflowStatusReadyToStart           = "ready_to_start"
	WorkflowStatusReadyForTRD            = "ready_for_trd"
	WorkflowStatusReadyForTasks          = "ready_for_tasks"
	WorkflowStatusReadyForSubtasks       = "ready_for_subtasks"
	WorkflowStatusReadyForImplementation = "ready_for_implementation"
)

// Common field constants
const (
	FieldPriority  = "priority"
	FieldAnalytics = "analytics"
)

// Project type constants
const (
	ProjectTypeAPI     = "api"
	ProjectTypeWebApp  = "web-app"
	ProjectTypeGeneral = "general"
)

// Repository constants
const (
	RepositoryGlobal = "global"
	RepositoryLocal  = "local"
)

// Boolean string constants
const (
	BoolStringTrue = "true"
)

// Analytics constants
const (
	OutlierTypePositive = "positive"
	StatusStable        = "stable"
)

// Severity constants
const (
	SeverityHigh    = "high"
	SeverityMedium  = "medium"
	SeverityLow     = "low"
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

// Time of day constants
const (
	TimeOfDayMorning   = "morning"
	TimeOfDayAfternoon = "afternoon"
	TimeOfDayEvening   = "evening"
	TimeOfDayNight     = "night"
)

// Task type constants
const (
	TaskTypeImplementation = "implementation"
	TaskTypeResearch       = "research"
	TaskTypeTesting        = "testing"
	TaskTypeDefault        = "default"
)

// Language constants
const (
	LanguageJavaScript = "javascript"
	LanguagePython     = "python"
	LanguageCSharp     = "csharp"
	LanguagePostgreSQL = "postgresql"
)

// MCP constants
const (
	MCPMethodMemorySystem = "memory_system"
	MCPOperationHealth    = "health"
)

// Status string constants
const (
	StatusStringDone      = "done"
	StatusStringCompleted = "completed"
)

// Directory constants
const (
	DefaultPreDevelopmentDir = "docs/pre-development"
)
