package entities

import (
	"time"
)

// ProjectType represents different types of projects
type ProjectType string

const (
	ProjectTypeWebApp       ProjectType = "web_app"
	ProjectTypeCLI          ProjectType = "cli"
	ProjectTypeAPI          ProjectType = "api"
	ProjectTypeLibrary      ProjectType = "library"
	ProjectTypeMicroservice ProjectType = "microservice"
	ProjectTypeDataPipeline ProjectType = "data_pipeline"
	ProjectTypeMobile       ProjectType = "mobile"
	ProjectTypeDesktop      ProjectType = "desktop"
	ProjectTypeGame         ProjectType = "game"
	ProjectTypeUnknown      ProjectType = "unknown"
)

// TaskTemplate represents a reusable template for creating tasks
type TaskTemplate struct {
	ID            string                 `json:"id" validate:"required,uuid"`
	Name          string                 `json:"name" validate:"required,min=1,max=200"`
	Description   string                 `json:"description"`
	ProjectType   ProjectType            `json:"project_type" validate:"required"`
	Category      string                 `json:"category"` // "initialization", "feature", "maintenance"
	Version       string                 `json:"version"`
	Author        string                 `json:"author"`
	Tasks         []TemplateTask         `json:"tasks" validate:"required,min=1"`
	Variables     []TemplateVariable     `json:"variables"`
	Prerequisites []string               `json:"prerequisites"` // Other template IDs
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	UsageCount    int                    `json:"usage_count"`
	SuccessRate   float64                `json:"success_rate"`
	LastUsed      *time.Time             `json:"last_used,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	IsBuiltIn     bool                   `json:"is_built_in"`
	IsPublic      bool                   `json:"is_public"`
}

// TemplateTask represents a task within a template
type TemplateTask struct {
	Order          int                    `json:"order"`
	Content        string                 `json:"content" validate:"required"`
	Description    string                 `json:"description"`
	Priority       string                 `json:"priority"`
	Type           string                 `json:"type"`
	EstimatedHours float64                `json:"estimated_hours"`
	Dependencies   []int                  `json:"dependencies"` // Orders of dependent tasks
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// TemplateVariable represents a configurable variable in a template
type TemplateVariable struct {
	Name            string                 `json:"name" validate:"required"`
	Description     string                 `json:"description"`
	Type            string                 `json:"type"` // "string", "number", "boolean", "choice"
	Default         interface{}            `json:"default"`
	Required        bool                   `json:"required"`
	Options         []string               `json:"options,omitempty"` // For choice type
	ValidationRegex string                 `json:"validation_regex,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateMatch represents a template matched to a project
type TemplateMatch struct {
	Template  *TaskTemplate          `json:"template"`
	Score     float64                `json:"score"`               // Match score 0-1
	Reason    string                 `json:"reason"`              // Why this template matches
	Variables map[string]interface{} `json:"variables,omitempty"` // Suggested variable values
}

// ProjectCharacteristics represents analyzed project characteristics
type ProjectCharacteristics struct {
	Languages      map[string]int         `json:"languages"` // Language -> file count
	Frameworks     []string               `json:"frameworks"`
	Dependencies   []string               `json:"dependencies"`
	FilePatterns   map[string]int         `json:"file_patterns"` // Pattern -> count
	HasTests       bool                   `json:"has_tests"`
	HasCI          bool                   `json:"has_ci"`
	HasDocker      bool                   `json:"has_docker"`
	HasDatabase    bool                   `json:"has_database"`
	HasAPI         bool                   `json:"has_api"`
	HasFrontend    bool                   `json:"has_frontend"`
	HasBackend     bool                   `json:"has_backend"`
	DirectoryDepth int                    `json:"directory_depth"`
	TotalFiles     int                    `json:"total_files"`
	ConfigFiles    []string               `json:"config_files"`
	BuildFiles     []string               `json:"build_files"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// TemplateInstantiation represents an instantiated template
type TemplateInstantiation struct {
	ID           string                 `json:"id"`
	TemplateID   string                 `json:"template_id"`
	Repository   string                 `json:"repository"`
	Variables    map[string]interface{} `json:"variables"`
	CreatedTasks []string               `json:"created_tasks"` // Task IDs
	Status       string                 `json:"status"`        // "pending", "in_progress", "completed", "failed"
	Progress     float64                `json:"progress"`      // 0-1
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Helper methods for ProjectCharacteristics

// HasFramework checks if the project uses any of the specified frameworks
func (pc *ProjectCharacteristics) HasFramework(frameworks ...string) bool {
	for _, framework := range frameworks {
		for _, projectFramework := range pc.Frameworks {
			if framework == projectFramework {
				return true
			}
		}
	}
	return false
}

// HasMainFile checks if the project has a main entry point file
func (pc *ProjectCharacteristics) HasMainFile() bool {
	patterns := []string{"main.", "index.", "app.", "cli.", "cmd/"}
	for _, pattern := range patterns {
		if count, exists := pc.FilePatterns[pattern]; exists && count > 0 {
			return true
		}
	}
	return false
}

// HasKubernetes checks if the project has Kubernetes configuration
func (pc *ProjectCharacteristics) HasKubernetes() bool {
	kubernetesIndicators := []string{"k8s", "kubernetes", "helm", "kustomize"}
	for _, indicator := range kubernetesIndicators {
		if count, exists := pc.FilePatterns[indicator]; exists && count > 0 {
			return true
		}
	}
	return false
}

// HasServiceMesh checks if the project has service mesh configuration
func (pc *ProjectCharacteristics) HasServiceMesh() bool {
	serviceMeshIndicators := []string{"istio", "envoy", "linkerd", "consul"}
	for _, indicator := range serviceMeshIndicators {
		if count, exists := pc.FilePatterns[indicator]; exists && count > 0 {
			return true
		}
	}
	return false
}

// GetPrimaryLanguage returns the most used language in the project
func (pc *ProjectCharacteristics) GetPrimaryLanguage() string {
	if len(pc.Languages) == 0 {
		return "unknown"
	}

	maxCount := 0
	primaryLang := "unknown"

	for lang, count := range pc.Languages {
		if count > maxCount {
			maxCount = count
			primaryLang = lang
		}
	}

	return primaryLang
}

// IsMonorepo checks if the project appears to be a monorepo
func (pc *ProjectCharacteristics) IsMonorepo() bool {
	return pc.DirectoryDepth > 4 && pc.TotalFiles > 500
}

// GetComplexityScore returns a complexity score based on characteristics
func (pc *ProjectCharacteristics) GetComplexityScore() float64 {
	score := 0.0

	// Language diversity
	score += float64(len(pc.Languages)) * 0.1

	// Framework count
	score += float64(len(pc.Frameworks)) * 0.15

	// File count
	if pc.TotalFiles > 1000 {
		score += 0.3
	} else if pc.TotalFiles > 100 {
		score += 0.2
	} else {
		score += 0.1
	}

	// Directory depth
	if pc.DirectoryDepth > 5 {
		score += 0.2
	} else if pc.DirectoryDepth > 3 {
		score += 0.1
	}

	// Technology indicators
	if pc.HasDocker {
		score += 0.1
	}
	if pc.HasKubernetes() {
		score += 0.2
	}
	if pc.HasDatabase {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// Helper methods for TaskTemplate

// GetTaskByOrder returns a task by its order
func (tt *TaskTemplate) GetTaskByOrder(order int) *TemplateTask {
	for i := range tt.Tasks {
		if tt.Tasks[i].Order == order {
			return &tt.Tasks[i]
		}
	}
	return nil
}

// GetDependentTasks returns tasks that depend on the given task order
func (tt *TaskTemplate) GetDependentTasks(order int) []TemplateTask {
	var dependents []TemplateTask
	for i := range tt.Tasks {
		task := &tt.Tasks[i]
		for _, dep := range task.Dependencies {
			if dep == order {
				dependents = append(dependents, *task)
				break
			}
		}
	}
	return dependents
}

// GetEstimatedTotalHours returns the total estimated hours for all tasks
func (tt *TaskTemplate) GetEstimatedTotalHours() float64 {
	total := 0.0
	for i := range tt.Tasks {
		total += tt.Tasks[i].EstimatedHours
	}
	return total
}

// ValidateTemplate validates the template structure
func (tt *TaskTemplate) ValidateTemplate() []string {
	var errors []string

	// Check for duplicate orders
	orders := make(map[int]bool)
	for _, task := range tt.Tasks {
		if orders[task.Order] {
			errors = append(errors, "duplicate task order: "+string(rune(task.Order)))
		}
		orders[task.Order] = true
	}

	// Check dependency validity
	for _, task := range tt.Tasks {
		for _, dep := range task.Dependencies {
			if !orders[dep] {
				errors = append(errors, "invalid dependency order: "+string(rune(dep)))
			}
			if dep >= task.Order {
				errors = append(errors, "circular or forward dependency detected")
			}
		}
	}

	// Check required variables
	for _, variable := range tt.Variables {
		if variable.Required && variable.Default == nil {
			errors = append(errors, "required variable without default: "+variable.Name)
		}
	}

	return errors
}

// GetVariableByName returns a variable by name
func (tt *TaskTemplate) GetVariableByName(name string) *TemplateVariable {
	for _, variable := range tt.Variables {
		if variable.Name == name {
			return &variable
		}
	}
	return nil
}

// UpdateUsageStats updates usage statistics for the template
func (tt *TaskTemplate) UpdateUsageStats(success bool) {
	tt.UsageCount++
	now := time.Now()
	tt.LastUsed = &now
	tt.UpdatedAt = now

	if success {
		// Calculate new success rate using exponential moving average
		alpha := 0.1 // Smoothing factor
		if tt.UsageCount == 1 {
			tt.SuccessRate = 1.0
		} else {
			tt.SuccessRate = alpha*1.0 + (1-alpha)*tt.SuccessRate
		}
	} else {
		// Failure case
		alpha := 0.1
		if tt.UsageCount == 1 {
			tt.SuccessRate = 0.0
		} else {
			tt.SuccessRate = alpha*0.0 + (1-alpha)*tt.SuccessRate
		}
	}
}
