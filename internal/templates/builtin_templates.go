// Package templates provides built-in template library for task management
package templates

import (
	"lerian-mcp-memory/pkg/types"
	"time"
)

// BuiltinTemplate represents a built-in task template
type BuiltinTemplate struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	ProjectType   types.ProjectType      `json:"project_type"`
	Tasks         []TemplateTask         `json:"tasks"`
	Variables     []TemplateVariable     `json:"variables"`
	Prerequisites []string               `json:"prerequisites"`
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	Version       string                 `json:"version"`
	Author        string                 `json:"author"`
	CreatedAt     time.Time              `json:"created_at"`
}

// TemplateTask represents a task within a template
type TemplateTask struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Type          string                 `json:"type"`           // "implementation", "testing", "documentation", "review"
	Priority      string                 `json:"priority"`       // "critical", "high", "medium", "low"
	EstimatedTime string                 `json:"estimated_time"` // "30m", "2h", "1d"
	Dependencies  []string               `json:"dependencies"`   // Names of other tasks in template
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	Template      string                 `json:"template"` // Template string with variables
}

// TemplateVariable represents a variable in a template
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // "string", "number", "boolean", "choice"
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Options      []string    `json:"options,omitempty"`    // For choice type
	Validation   string      `json:"validation,omitempty"` // Regex pattern
}

// GetBuiltinTemplates returns all built-in templates
func GetBuiltinTemplates() []BuiltinTemplate {
	return []BuiltinTemplate{
		createWebAppFeatureTemplate(),
		createAPIEndpointTemplate(),
		createBugFixTemplate(),
		createDatabaseMigrationTemplate(),
		createTestingSuiteTemplate(),
		createDocumentationTemplate(),
		createDeploymentTemplate(),
		createSecurityReviewTemplate(),
		createPerformanceOptimizationTemplate(),
		createCodeRefactorTemplate(),
	}
}

// createWebAppFeatureTemplate creates a template for implementing web application features
func createWebAppFeatureTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-webapp-feature",
		Name:        "Web Application Feature",
		Description: "Template for implementing new features in web applications",
		Category:    "feature",
		ProjectType: types.ProjectTypeWeb,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"web", "feature", "development"},
		Variables: []TemplateVariable{
			{
				Name:        "feature_name",
				Type:        "string",
				Description: "Name of the feature to implement",
				Required:    true,
				Validation:  "^[a-zA-Z][a-zA-Z0-9_-]*$",
			},
			{
				Name:        "feature_description",
				Type:        "string",
				Description: "Detailed description of the feature",
				Required:    true,
			},
			{
				Name:         "has_api",
				Type:         "boolean",
				Description:  "Whether this feature requires new API endpoints",
				Required:     false,
				DefaultValue: true,
			},
			{
				Name:         "has_database",
				Type:         "boolean",
				Description:  "Whether this feature requires database changes",
				Required:     false,
				DefaultValue: false,
			},
			{
				Name:         "frontend_framework",
				Type:         "choice",
				Description:  "Frontend framework being used",
				Required:     false,
				DefaultValue: "react",
				Options:      []string{"react", "vue", "angular", "vanilla"},
			},
		},
		Tasks: []TemplateTask{
			{
				Name:          "Create feature specification",
				Description:   "Define detailed requirements and specifications for {{.feature_name}}",
				Type:          "documentation",
				Priority:      "high",
				EstimatedTime: "1h",
				Dependencies:  []string{},
				Tags:          []string{"specification", "planning"},
				Template:      "Create comprehensive specification for {{.feature_name}} feature:\n\n{{.feature_description}}\n\nInclude:\n- User stories\n- Acceptance criteria\n- Technical requirements\n- Edge cases",
			},
			{
				Name:          "Design database schema",
				Description:   "Design database changes required for {{.feature_name}}",
				Type:          "implementation",
				Priority:      "high",
				EstimatedTime: "2h",
				Dependencies:  []string{"Create feature specification"},
				Tags:          []string{"database", "design"},
				Template:      "{{if .has_database}}Design database schema changes for {{.feature_name}}:\n\n- Create migration files\n- Define new tables/columns\n- Add necessary indexes\n- Consider data relationships{{else}}No database changes required for this feature{{end}}",
			},
			{
				Name:          "Implement backend API",
				Description:   "Create backend API endpoints for {{.feature_name}}",
				Type:          "implementation",
				Priority:      "high",
				EstimatedTime: "4h",
				Dependencies:  []string{"Design database schema"},
				Tags:          []string{"backend", "api"},
				Template:      "{{if .has_api}}Implement backend API for {{.feature_name}}:\n\n- Create REST endpoints\n- Add input validation\n- Implement business logic\n- Add error handling\n- Write API documentation{{else}}No API changes required for this feature{{end}}",
			},
			{
				Name:          "Implement frontend components",
				Description:   "Create frontend UI components for {{.feature_name}}",
				Type:          "implementation",
				Priority:      "high",
				EstimatedTime: "6h",
				Dependencies:  []string{"Implement backend API"},
				Tags:          []string{"frontend", "{{.frontend_framework}}"},
				Template:      "Implement {{.frontend_framework}} components for {{.feature_name}}:\n\n- Create reusable components\n- Add state management\n- Implement user interactions\n- Add responsive design\n- Follow UI/UX guidelines",
			},
			{
				Name:          "Write unit tests",
				Description:   "Create comprehensive unit tests for {{.feature_name}}",
				Type:          "testing",
				Priority:      "medium",
				EstimatedTime: "3h",
				Dependencies:  []string{"Implement frontend components"},
				Tags:          []string{"testing", "unit-tests"},
				Template:      "Write unit tests for {{.feature_name}}:\n\n- Backend API tests\n- Frontend component tests\n- Edge case validation\n- Mock external dependencies\n- Aim for >90% code coverage",
			},
			{
				Name:          "Write integration tests",
				Description:   "Create integration tests for {{.feature_name}}",
				Type:          "testing",
				Priority:      "medium",
				EstimatedTime: "2h",
				Dependencies:  []string{"Write unit tests"},
				Tags:          []string{"testing", "integration-tests"},
				Template:      "Write integration tests for {{.feature_name}}:\n\n- End-to-end user workflows\n- API integration tests\n- Database integration tests\n- Cross-browser testing\n- Performance testing",
			},
			{
				Name:          "Update documentation",
				Description:   "Update project documentation for {{.feature_name}}",
				Type:          "documentation",
				Priority:      "low",
				EstimatedTime: "1h",
				Dependencies:  []string{"Write integration tests"},
				Tags:          []string{"documentation"},
				Template:      "Update documentation for {{.feature_name}}:\n\n- Update README.md\n- Add feature documentation\n- Update API documentation\n- Add usage examples\n- Update changelog",
			},
			{
				Name:          "Code review and testing",
				Description:   "Review code and test {{.feature_name}} thoroughly",
				Type:          "review",
				Priority:      "high",
				EstimatedTime: "1h",
				Dependencies:  []string{"Update documentation"},
				Tags:          []string{"review", "qa"},
				Template:      "Review and test {{.feature_name}}:\n\n- Code review checklist\n- Manual testing scenarios\n- Security review\n- Performance review\n- Documentation review",
			},
		},
		Metadata: map[string]interface{}{
			"complexity":       "medium",
			"team_size":        "1-3",
			"typical_duration": "2-5 days",
		},
	}
}

// createAPIEndpointTemplate creates a template for implementing API endpoints
func createAPIEndpointTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-api-endpoint",
		Name:        "REST API Endpoint",
		Description: "Template for implementing new REST API endpoints",
		Category:    "api",
		ProjectType: types.ProjectTypeAPI,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"api", "backend", "rest"},
		Variables: []TemplateVariable{
			{
				Name:        "endpoint_name",
				Type:        "string",
				Description: "Name of the API endpoint (e.g., users, orders)",
				Required:    true,
				Validation:  "^[a-z][a-z0-9_-]*$",
			},
			{
				Name:         "http_methods",
				Type:         "choice",
				Description:  "HTTP methods to implement",
				Required:     true,
				DefaultValue: "crud",
				Options:      []string{"get", "post", "put", "delete", "crud"},
			},
			{
				Name:         "requires_auth",
				Type:         "boolean",
				Description:  "Whether the endpoint requires authentication",
				Required:     false,
				DefaultValue: true,
			},
			{
				Name:         "has_validation",
				Type:         "boolean",
				Description:  "Whether input validation is required",
				Required:     false,
				DefaultValue: true,
			},
		},
		Tasks: []TemplateTask{
			{
				Name:          "Design API contract",
				Description:   "Define API contract for {{.endpoint_name}} endpoint",
				Type:          "documentation",
				Priority:      "high",
				EstimatedTime: "1h",
				Dependencies:  []string{},
				Tags:          []string{"api-design", "specification"},
				Template:      "Design API contract for {{.endpoint_name}} endpoint:\n\n- Define request/response schemas\n- Specify HTTP methods: {{.http_methods}}\n- Define error responses\n- Add OpenAPI specification\n{{if .requires_auth}}- Define authentication requirements{{end}}\n{{if .has_validation}}- Define validation rules{{end}}",
			},
			{
				Name:          "Implement endpoint handlers",
				Description:   "Create HTTP handlers for {{.endpoint_name}}",
				Type:          "implementation",
				Priority:      "high",
				EstimatedTime: "3h",
				Dependencies:  []string{"Design API contract"},
				Tags:          []string{"backend", "handlers"},
				Template:      "Implement {{.endpoint_name}} endpoint handlers:\n\n{{if eq .http_methods \"crud\"}}Create CRUD operations:\n- GET /{{.endpoint_name}} (list)\n- GET /{{.endpoint_name}}/:id (get by ID)\n- POST /{{.endpoint_name}} (create)\n- PUT /{{.endpoint_name}}/:id (update)\n- DELETE /{{.endpoint_name}}/:id (delete){{else}}Create {{.http_methods}} handler for /{{.endpoint_name}}{{end}}\n\n{{if .requires_auth}}- Add authentication middleware{{end}}\n{{if .has_validation}}- Add input validation{{end}}",
			},
			{
				Name:          "Add error handling",
				Description:   "Implement proper error handling for {{.endpoint_name}}",
				Type:          "implementation",
				Priority:      "medium",
				EstimatedTime: "1h",
				Dependencies:  []string{"Implement endpoint handlers"},
				Tags:          []string{"error-handling", "robustness"},
				Template:      "Add comprehensive error handling for {{.endpoint_name}}:\n\n- Handle validation errors (400)\n- Handle authentication errors (401)\n- Handle authorization errors (403)\n- Handle not found errors (404)\n- Handle server errors (500)\n- Add proper error logging\n- Return consistent error format",
			},
			{
				Name:          "Write API tests",
				Description:   "Create comprehensive tests for {{.endpoint_name}} API",
				Type:          "testing",
				Priority:      "high",
				EstimatedTime: "2h",
				Dependencies:  []string{"Add error handling"},
				Tags:          []string{"testing", "api-tests"},
				Template:      "Write comprehensive API tests for {{.endpoint_name}}:\n\n- Unit tests for handlers\n- Integration tests for endpoints\n- Authentication tests\n- Validation tests\n- Error scenario tests\n- Performance tests\n- Load testing scenarios",
			},
		},
		Metadata: map[string]interface{}{
			"complexity":       "low-medium",
			"team_size":        "1",
			"typical_duration": "1-2 days",
		},
	}
}

// createBugFixTemplate creates a template for fixing bugs
func createBugFixTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-bug-fix",
		Name:        "Bug Fix",
		Description: "Template for systematically fixing bugs",
		Category:    "maintenance",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"bug", "maintenance", "debugging"},
		Variables: []TemplateVariable{
			{
				Name:        "bug_title",
				Type:        "string",
				Description: "Brief title describing the bug",
				Required:    true,
			},
			{
				Name:        "bug_description",
				Type:        "string",
				Description: "Detailed description of the bug",
				Required:    true,
			},
			{
				Name:         "severity",
				Type:         "choice",
				Description:  "Severity level of the bug",
				Required:     true,
				Options:      []string{"critical", "high", "medium", "low"},
				DefaultValue: "medium",
			},
			{
				Name:        "affected_component",
				Type:        "string",
				Description: "Component or module affected by the bug",
				Required:    false,
			},
		},
		Tasks: []TemplateTask{
			{
				Name:          "Reproduce the bug",
				Description:   "Reproduce and document the bug: {{.bug_title}}",
				Type:          "investigation",
				Priority:      "{{.severity}}",
				EstimatedTime: "30m",
				Dependencies:  []string{},
				Tags:          []string{"reproduction", "investigation"},
				Template:      "Reproduce the bug: {{.bug_title}}\n\nDescription: {{.bug_description}}\n\nSteps to reproduce:\n1. \n2. \n3. \n\nExpected behavior:\n\nActual behavior:\n\nEnvironment details:\n- OS:\n- Browser/Version:\n- Other relevant info:\n\n{{if .affected_component}}Affected component: {{.affected_component}}{{end}}",
			},
			{
				Name:          "Investigate root cause",
				Description:   "Investigate the root cause of {{.bug_title}}",
				Type:          "investigation",
				Priority:      "{{.severity}}",
				EstimatedTime: "1h",
				Dependencies:  []string{"Reproduce the bug"},
				Tags:          []string{"debugging", "analysis"},
				Template:      "Investigate root cause of {{.bug_title}}:\n\n- Review error logs\n- Check recent code changes\n- Debug step by step\n- Identify exact failure point\n- Determine impact scope\n- Consider similar issues",
			},
			{
				Name:          "Implement fix",
				Description:   "Implement fix for {{.bug_title}}",
				Type:          "implementation",
				Priority:      "{{.severity}}",
				EstimatedTime: "2h",
				Dependencies:  []string{"Investigate root cause"},
				Tags:          []string{"fix", "implementation"},
				Template:      "Implement fix for {{.bug_title}}:\n\n- Apply minimal necessary changes\n- Ensure fix addresses root cause\n- Consider edge cases\n- Maintain backward compatibility\n- Add defensive programming measures\n- Document the fix approach",
			},
			{
				Name:          "Write regression tests",
				Description:   "Write tests to prevent regression of {{.bug_title}}",
				Type:          "testing",
				Priority:      "high",
				EstimatedTime: "1h",
				Dependencies:  []string{"Implement fix"},
				Tags:          []string{"testing", "regression"},
				Template:      "Write regression tests for {{.bug_title}}:\n\n- Test the specific bug scenario\n- Test edge cases that led to the bug\n- Add integration tests if needed\n- Verify fix works in different environments\n- Ensure existing functionality still works",
			},
			{
				Name:          "Verify fix and deploy",
				Description:   "Verify the fix and deploy for {{.bug_title}}",
				Type:          "review",
				Priority:      "{{.severity}}",
				EstimatedTime: "30m",
				Dependencies:  []string{"Write regression tests"},
				Tags:          []string{"verification", "deployment"},
				Template:      "Verify and deploy fix for {{.bug_title}}:\n\n- Verify bug is fixed in test environment\n- Run full test suite\n- Code review\n- Deploy to staging\n- Verify in staging environment\n- Deploy to production\n- Monitor for any new issues",
			},
		},
		Metadata: map[string]interface{}{
			"complexity":       "variable",
			"team_size":        "1",
			"typical_duration": "0.5-2 days",
		},
	}
}

// Additional builtin templates for completeness
func createDatabaseMigrationTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-database-migration",
		Name:        "Database Migration",
		Description: "Template for implementing database schema changes",
		Category:    "infrastructure",
		ProjectType: types.ProjectTypeBackend,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"database", "migration", "schema"},
		Variables: []TemplateVariable{
			{
				Name:        "migration_name",
				Type:        "string",
				Description: "Name of the migration",
				Required:    true,
			},
			{
				Name:         "migration_type",
				Type:         "choice",
				Description:  "Type of migration",
				Required:     true,
				Options:      []string{"add_table", "modify_table", "add_index", "data_migration"},
				DefaultValue: "add_table",
			},
		},
		Tasks: []TemplateTask{
			{
				Name:          "Design migration",
				Description:   "Design database migration: {{.migration_name}}",
				Type:          "design",
				Priority:      "high",
				EstimatedTime: "1h",
				Dependencies:  []string{},
				Tags:          []string{"design", "database"},
				Template:      "Design migration {{.migration_name}} ({{.migration_type}}):\n\n- Define schema changes\n- Plan rollback strategy\n- Consider data impact\n- Review dependencies",
			},
		},
		Metadata: map[string]interface{}{
			"complexity": "medium",
		},
	}
}

func createTestingSuiteTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-testing-suite",
		Name:        "Testing Suite Setup",
		Description: "Template for setting up comprehensive testing",
		Category:    "testing",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"testing", "qa", "automation"},
		Variables:   []TemplateVariable{},
		Tasks:       []TemplateTask{},
		Metadata:    map[string]interface{}{"complexity": "medium"},
	}
}

func createDocumentationTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-documentation",
		Name:        "Documentation Update",
		Description: "Template for updating project documentation",
		Category:    "documentation",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"documentation", "knowledge"},
		Variables:   []TemplateVariable{},
		Tasks:       []TemplateTask{},
		Metadata:    map[string]interface{}{"complexity": "low"},
	}
}

func createDeploymentTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-deployment",
		Name:        "Deployment Setup",
		Description: "Template for setting up deployment pipeline",
		Category:    "deployment",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"deployment", "devops", "ci-cd"},
		Variables:   []TemplateVariable{},
		Tasks:       []TemplateTask{},
		Metadata:    map[string]interface{}{"complexity": "high"},
	}
}

func createSecurityReviewTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-security-review",
		Name:        "Security Review",
		Description: "Template for conducting security reviews",
		Category:    "security",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"security", "review", "audit"},
		Variables:   []TemplateVariable{},
		Tasks:       []TemplateTask{},
		Metadata:    map[string]interface{}{"complexity": "high"},
	}
}

func createPerformanceOptimizationTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-performance-optimization",
		Name:        "Performance Optimization",
		Description: "Template for optimizing application performance",
		Category:    "optimization",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"performance", "optimization", "monitoring"},
		Variables:   []TemplateVariable{},
		Tasks:       []TemplateTask{},
		Metadata:    map[string]interface{}{"complexity": "high"},
	}
}

func createCodeRefactorTemplate() BuiltinTemplate {
	return BuiltinTemplate{
		ID:          "builtin-code-refactor",
		Name:        "Code Refactoring",
		Description: "Template for refactoring existing code",
		Category:    "refactoring",
		ProjectType: types.ProjectTypeAny,
		Version:     "1.0.0",
		Author:      "Lerian MCP Memory",
		CreatedAt:   time.Now(),
		Tags:        []string{"refactor", "cleanup", "maintainability"},
		Variables:   []TemplateVariable{},
		Tasks:       []TemplateTask{},
		Metadata:    map[string]interface{}{"complexity": "medium"},
	}
}
