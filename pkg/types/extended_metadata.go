package types

// ExtendedMetadataKeys defines standard keys for extended metadata
type ExtendedMetadataKeys struct {
	// Location Context
	WorkingDirectory string
	RelativePath     string
	GitBranch        string
	GitCommit        string
	ProjectType      string

	// Client Context
	ClientType    string
	ClientVersion string
	Platform      string
	Environment   map[string]string

	// Enhanced Metadata
	LanguageVersions map[string]string
	Dependencies     map[string]string
	ErrorSignatures  []string
	StackTraces      []string
	CommandResults   []CommandResult
	FileOperations   []FileOperation

	// Relationships
	ParentChunkID   string
	ChildChunkIDs   []string
	RelatedChunkIDs []string
	SupersededByID  string
	SupersedesID    string

	// Search & Analytics
	AutoTags           []string
	ProblemDomain      string
	SemanticCategories []string
	ConfidenceScore    float64

	// Usage Analytics
	AccessCount        int
	LastAccessedAt     string
	SuccessRate        float64
	EffectivenessScore float64
	IsObsolete         bool
	ArchivedAt         string
}

// CommandResult captures the result of a command execution
type CommandResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// FileOperation captures file operations performed
type FileOperation struct {
	Type     string `json:"type"` // create, edit, delete, rename
	FilePath string `json:"file_path"`
	OldPath  string `json:"old_path,omitempty"` // for rename
}

// Standard keys for extended metadata
const (
	// Location Context Keys
	EMKeyWorkingDir   = "working_directory"
	EMKeyRelativePath = "relative_path"
	EMKeyGitBranch    = "git_branch"
	EMKeyGitCommit    = "git_commit"
	EMKeyProjectType  = "project_type"

	// Client Context Keys
	EMKeyClientType    = "client_type"
	EMKeyClientVersion = "client_version"
	EMKeyPlatform      = "platform"
	EMKeyEnvironment   = "environment"

	// Enhanced Metadata Keys
	EMKeyLanguageVersions = "language_versions"
	EMKeyDependencies     = "dependencies"
	EMKeyErrorSignatures  = "error_signatures"
	EMKeyStackTraces      = "stack_traces"
	EMKeyCommandResults   = "command_results"
	EMKeyFileOperations   = "file_operations"

	// Relationship Keys
	EMKeyParentChunk   = "parent_chunk_id"
	EMKeyChildChunks   = "child_chunk_ids"
	EMKeyRelatedChunks = "related_chunk_ids"
	EMKeySupersededBy  = "superseded_by_id"
	EMKeySupersedes    = "supersedes_id"

	// Search & Analytics Keys
	EMKeyAutoTags           = "auto_tags"
	EMKeyProblemDomain      = "problem_domain"
	EMKeySemanticCategories = "semantic_categories"
	EMKeyConfidenceScore    = "confidence_score"

	// Usage Analytics Keys
	EMKeyAccessCount        = "access_count"
	EMKeyLastAccessed       = "last_accessed_at"
	EMKeySuccessRate        = "success_rate"
	EMKeyEffectivenessScore = "effectiveness_score"
	EMKeyIsObsolete         = "is_obsolete"
	EMKeyArchivedAt         = "archived_at"
)

// Client types
const (
	ClientTypeCLI     = "claude-cli"
	ClientTypeChatGPT = "chatgpt"
	ClientTypeVSCode  = "vscode"
	ClientTypeWeb     = "web"
	ClientTypeAPI     = "api"
)

// Project types
const (
	ProjectTypeGo         = "go"
	ProjectTypePython     = "python"
	ProjectTypeJavaScript = "javascript"
	ProjectTypeTypeScript = "typescript"
	ProjectTypeRust       = "rust"
	ProjectTypeJava       = "java"
	ProjectTypeUnknown    = "unknown"
)

// Problem domains
const (
	DomainFrontend    = "frontend"
	DomainBackend     = "backend"
	DomainDatabase    = "database"
	DomainCICD        = "ci-cd"
	DomainSecurity    = "security"
	DomainPerformance = "performance"
	DomainTesting     = "testing"
	DomainDocs        = "documentation"
)
