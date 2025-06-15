// Package prompts provides functionality for loading and managing review prompts
package prompts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"lerian-mcp-memory-cli/internal/domain/entities"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PromptLoader loads review prompts from the filesystem
type PromptLoader struct {
	promptsDir string
	prompts    map[string]*entities.ReviewPrompt
}

// NewPromptLoader creates a new prompt loader
func NewPromptLoader(promptsDir string) *PromptLoader {
	return &PromptLoader{
		promptsDir: promptsDir,
		prompts:    make(map[string]*entities.ReviewPrompt),
	}
}

// LoadPrompts loads all prompts from the prompts directory
func (l *PromptLoader) LoadPrompts() error {
	// Define the review prompts directory
	reviewDir := filepath.Join(l.promptsDir, "2-code-review")

	// Check if directory exists
	if _, err := os.Stat(reviewDir); os.IsNotExist(err) {
		return fmt.Errorf("review prompts directory not found: %s", reviewDir)
	}

	// Load all .md and .mdc files
	err := filepath.Walk(reviewDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-prompt files
		if info.IsDir() || (!strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdc")) {
			return nil
		}

		// Skip the orchestrator file
		if strings.Contains(info.Name(), "orchestrator") {
			return nil
		}

		// Load the prompt
		prompt, err := l.loadPrompt(path)
		if err != nil {
			return fmt.Errorf("failed to load prompt %s: %w", path, err)
		}

		l.prompts[prompt.ID] = prompt
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load prompts: %w", err)
	}

	// Establish dependencies
	l.establishDependencies()

	return nil
}

// loadPrompt loads a single prompt from a file
func (l *PromptLoader) loadPrompt(path string) (*entities.ReviewPrompt, error) {
	// Validate path is within prompts directory
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, l.promptsDir) {
		return nil, errors.New("invalid path: outside of prompts directory")
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Extract prompt info from filename
	filename := filepath.Base(path)
	id, order, name := l.parseFilename(filename)

	// Parse phase from order
	phase := l.determinePhase(order)

	// Extract tags from content
	tags := l.extractTags(string(content))

	prompt := &entities.ReviewPrompt{
		ID:          id,
		Name:        name,
		Description: l.extractDescription(string(content)),
		Phase:       phase,
		Order:       order,
		FilePath:    path,
		Content:     string(content),
		Tags:        tags,
	}

	return prompt, nil
}

// parseFilename extracts ID, order, and name from filename
func (l *PromptLoader) parseFilename(filename string) (string, int, string) {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Extract order number (e.g., "01-codebase-overview" -> 1)
	re := regexp.MustCompile(`^(\d+)-(.+)$`)
	matches := re.FindStringSubmatch(name)

	if len(matches) == 3 {
		order, _ := strconv.Atoi(matches[1])
		cleanName := strings.ReplaceAll(matches[2], "-", " ")
		caser := cases.Title(language.English)
		cleanName = caser.String(cleanName)
		return fmt.Sprintf("prompt-%02d", order), order, cleanName
	}

	// Fallback
	return name, 99, name
}

// determinePhase determines the review phase based on prompt order
func (l *PromptLoader) determinePhase(order int) entities.ReviewPhase {
	switch {
	case order >= 1 && order <= 6:
		return entities.PhaseFoundation
	case order >= 7 && order <= 9:
		return entities.PhaseSecurity
	case order >= 10 && order <= 12:
		return entities.PhaseQuality
	case order >= 13 && order <= 15:
		return entities.PhaseDocumentation
	case order >= 16 && order <= 17:
		return entities.PhaseProduction
	case order >= 18:
		return entities.PhaseSynthesis
	default:
		return entities.PhaseFoundation
	}
}

// extractDescription extracts the description from prompt content
func (l *PromptLoader) extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			// Return first non-empty, non-header line
			if len(line) > 200 {
				return line[:200] + "..."
			}
			return line
		}
	}
	return "Code review prompt"
}

// extractTags extracts tags from prompt content
func (l *PromptLoader) extractTags(content string) []string {
	tags := []string{}

	// Look for explicit tags in content
	tagRe := regexp.MustCompile(`tags?:\s*\[(.*?)\]`)
	if matches := tagRe.FindStringSubmatch(content); len(matches) > 1 {
		tagStr := matches[1]
		for _, tag := range strings.Split(tagStr, ",") {
			tag = strings.TrimSpace(tag)
			tag = strings.Trim(tag, `"'`)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Add tags based on content keywords
	contentLower := strings.ToLower(content)
	keywordTags := map[string]string{
		"security":      "security",
		"vulnerability": "security",
		"api":           "api",
		"database":      "database",
		"test":          "testing",
		"coverage":      "testing",
		"performance":   "performance",
		"documentation": "documentation",
		"production":    "production",
		"deployment":    "deployment",
	}

	for keyword, tag := range keywordTags {
		if strings.Contains(contentLower, keyword) && !contains(tags, tag) {
			tags = append(tags, tag)
		}
	}

	return tags
}

// establishDependencies sets up dependencies between prompts
func (l *PromptLoader) establishDependencies() {
	// Based on the orchestrator documentation
	dependencies := map[int][]int{
		2:  {1},                                                         // Architecture depends on Overview
		3:  {1, 2},                                                      // API depends on Overview and Architecture
		4:  {1, 2},                                                      // Database depends on Overview and Architecture
		5:  {1, 2, 3, 4},                                                // Sequence diagrams depend on technical foundation
		6:  {1, 2, 3, 4, 5},                                             // Business analysis depends on all foundation
		7:  {1, 2, 3, 4},                                                // Security depends on technical foundation
		8:  {7},                                                         // Dependency security depends on security analysis
		9:  {7, 8},                                                      // Privacy depends on security analyses
		10: {1, 2, 7},                                                   // Test coverage depends on foundation and security
		11: {1, 2},                                                      // Observability depends on foundation
		12: {10},                                                        // Pre-commit depends on test coverage
		13: {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},                     // Documentation depends on all
		14: {3, 13},                                                     // API documentation depends on API analysis and general docs
		15: {5, 6},                                                      // Business workflow depends on sequence diagrams and business analysis
		16: {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},         // Production readiness depends on all
		17: {16},                                                        // Deployment depends on production readiness
		18: {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}, // Todo generation depends on all
	}

	for order, deps := range dependencies {
		promptID := fmt.Sprintf("prompt-%02d", order)
		if prompt, exists := l.prompts[promptID]; exists {
			for _, depOrder := range deps {
				depID := fmt.Sprintf("prompt-%02d", depOrder)
				prompt.DependsOn = append(prompt.DependsOn, depID)
			}
		}
	}
}

// GetPrompt returns a prompt by ID
func (l *PromptLoader) GetPrompt(id string) (*entities.ReviewPrompt, error) {
	prompt, exists := l.prompts[id]
	if !exists {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}
	return prompt, nil
}

// GetPromptsByPhase returns all prompts for a given phase
func (l *PromptLoader) GetPromptsByPhase(phase entities.ReviewPhase) []*entities.ReviewPrompt {
	var prompts []*entities.ReviewPrompt

	for _, prompt := range l.prompts {
		if prompt.Phase == phase {
			prompts = append(prompts, prompt)
		}
	}

	// Sort by order
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].Order < prompts[j].Order
	})

	return prompts
}

// GetAllPrompts returns all loaded prompts sorted by order
func (l *PromptLoader) GetAllPrompts() []*entities.ReviewPrompt {
	prompts := make([]*entities.ReviewPrompt, 0, len(l.prompts))

	for _, prompt := range l.prompts {
		prompts = append(prompts, prompt)
	}

	// Sort by order
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].Order < prompts[j].Order
	})

	return prompts
}

// GetPromptsForMode returns prompts based on review mode
func (l *PromptLoader) GetPromptsForMode(mode entities.ReviewMode) []*entities.ReviewPrompt {
	switch mode {
	case entities.ReviewModeQuick:
		// Quick mode: essential prompts only
		essentialOrders := []int{1, 2, 7, 10, 16, 18}
		return l.getPromptsByOrders(essentialOrders)

	case entities.ReviewModeSecurity:
		// Security focus
		securityOrders := []int{1, 7, 8, 9}
		return l.getPromptsByOrders(securityOrders)

	case entities.ReviewModeQuality:
		// Quality focus
		qualityOrders := []int{1, 10, 11, 12}
		return l.getPromptsByOrders(qualityOrders)

	default: // Full mode
		return l.GetAllPrompts()
	}
}

// getPromptsByOrders returns prompts matching specific order numbers
func (l *PromptLoader) getPromptsByOrders(orders []int) []*entities.ReviewPrompt {
	var prompts []*entities.ReviewPrompt

	for _, order := range orders {
		promptID := fmt.Sprintf("prompt-%02d", order)
		if prompt, exists := l.prompts[promptID]; exists {
			prompts = append(prompts, prompt)
		}
	}

	// Sort by order
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].Order < prompts[j].Order
	})

	return prompts
}

// ValidateDependencies checks if all dependencies are satisfied
func (l *PromptLoader) ValidateDependencies(completedPromptIDs []string) error {
	completedSet := make(map[string]bool)
	for _, id := range completedPromptIDs {
		completedSet[id] = true
	}

	for _, prompt := range l.prompts {
		for _, dep := range prompt.DependsOn {
			if !completedSet[dep] {
				return fmt.Errorf("prompt %s depends on %s which is not completed", prompt.ID, dep)
			}
		}
	}

	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
