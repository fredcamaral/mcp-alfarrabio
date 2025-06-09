// Package documents provides data structures and processing for PRD/TRD document management.
package documents

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed rules/*.mdc rules/*.yaml
var embeddedRules embed.FS

// RuleManager manages document generation rules
type RuleManager struct {
	rules       map[string]*Rule
	rulesByType map[RuleType][]*Rule
	mu          sync.RWMutex
	rulesDir    string
}

// NewRuleManager creates a new rule manager
func NewRuleManager(rulesDir string) (*RuleManager, error) {
	rm := &RuleManager{
		rules:       make(map[string]*Rule),
		rulesByType: make(map[RuleType][]*Rule),
		rulesDir:    rulesDir,
	}

	// Load embedded rules first
	if err := rm.loadEmbeddedRules(); err != nil {
		return nil, fmt.Errorf("failed to load embedded rules: %w", err)
	}

	// Load custom rules from directory if it exists
	if rulesDir != "" {
		if err := rm.loadCustomRules(rulesDir); err != nil {
			return nil, fmt.Errorf("failed to load custom rules: %w", err)
		}
	}

	return rm, nil
}

// loadEmbeddedRules loads rules from embedded files
func (rm *RuleManager) loadEmbeddedRules() error {
	return fs.WalkDir(embeddedRules, "rules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !rm.isRuleFile(path) {
			return nil
		}

		content, err := embeddedRules.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded rule %s: %w", path, err)
		}

		rule, err := rm.parseRuleFile(path, content)
		if err != nil {
			return fmt.Errorf("failed to parse embedded rule %s: %w", path, err)
		}

		rm.addRule(rule)
		return nil
	})
}

// loadCustomRules loads rules from a directory
func (rm *RuleManager) loadCustomRules(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !rm.isRuleFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read custom rule %s: %w", path, err)
		}

		rule, err := rm.parseRuleFile(path, content)
		if err != nil {
			return fmt.Errorf("failed to parse custom rule %s: %w", path, err)
		}

		// Custom rules override embedded rules with same name
		rm.addRule(rule)
		return nil
	})
}

// isRuleFile checks if a file is a rule file
func (rm *RuleManager) isRuleFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mdc" || ext == ".yaml" || ext == ".yml"
}

// parseRuleFile parses a rule file based on its extension
func (rm *RuleManager) parseRuleFile(path string, content []byte) (*Rule, error) {
	filename := filepath.Base(path)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	rule := &Rule{
		Name:      nameWithoutExt,
		Content:   string(content),
		Version:   "1.0.0",
		Active:    true,
		Priority:  50, // Default priority
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Determine rule type from filename
	switch {
	case strings.Contains(nameWithoutExt, "create-prd"):
		rule.Type = RulePRDGeneration
		rule.Description = "Rule for generating Product Requirements Documents"
	case strings.Contains(nameWithoutExt, "create-trd"):
		rule.Type = RuleTRDGeneration
		rule.Description = "Rule for generating Technical Requirements Documents"
	case strings.Contains(nameWithoutExt, "generate-main-tasks"):
		rule.Type = RuleTaskGeneration
		rule.Description = "Rule for generating main tasks from PRD/TRD"
	case strings.Contains(nameWithoutExt, "generate-sub-tasks"):
		rule.Type = RuleSubTaskGeneration
		rule.Description = "Rule for generating sub-tasks from main tasks"
	case strings.Contains(nameWithoutExt, "complexity"):
		rule.Type = RuleComplexityAnalysis
		rule.Description = "Rule for analyzing task complexity"
	case strings.Contains(nameWithoutExt, "validation"):
		rule.Type = RuleValidation
		rule.Description = "Rule for validating documents"
	default:
		rule.Type = RulePRDGeneration // Default type
	}

	// Parse YAML front matter if present
	if strings.HasPrefix(string(content), "---") {
		parts := strings.SplitN(string(content), "---", 3)
		if len(parts) >= 3 {
			var metadata map[string]interface{}
			if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err == nil {
				// Extract metadata
				if name, ok := metadata["name"].(string); ok {
					rule.Name = name
				}
				if desc, ok := metadata["description"].(string); ok {
					rule.Description = desc
				}
				if typeStr, ok := metadata["type"].(string); ok {
					rule.Type = RuleType(typeStr)
				}
				if priority, ok := metadata["priority"].(int); ok {
					rule.Priority = priority
				}
				if version, ok := metadata["version"].(string); ok {
					rule.Version = version
				}

				// Store other metadata
				for k, v := range metadata {
					if k != "name" && k != "description" && k != "type" && k != "priority" && k != "version" {
						rule.Metadata[k] = fmt.Sprintf("%v", v)
					}
				}
			}

			// Update content to exclude front matter
			rule.Content = strings.TrimSpace(parts[2])
		}
	}

	// Validate rule
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	return rule, nil
}

// addRule adds a rule to the manager
func (rm *RuleManager) addRule(rule *Rule) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.rules[rule.ID] = rule
	rm.rulesByType[rule.Type] = append(rm.rulesByType[rule.Type], rule)
}

// GetRule retrieves a rule by ID
func (rm *RuleManager) GetRule(id string) (*Rule, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rule, ok := rm.rules[id]
	if !ok {
		return nil, fmt.Errorf("rule not found: %s", id)
	}

	return rule, nil
}

// GetRuleByName retrieves a rule by name
func (rm *RuleManager) GetRuleByName(name string) (*Rule, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	for _, rule := range rm.rules {
		if rule.Name == name {
			return rule, nil
		}
	}

	return nil, fmt.Errorf("rule not found: %s", name)
}

// GetRulesByType retrieves all rules of a specific type
func (rm *RuleManager) GetRulesByType(ruleType RuleType) []*Rule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rules := rm.rulesByType[ruleType]

	// Sort by priority (higher priority first)
	sorted := make([]*Rule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	return sorted
}

// GetActiveRules retrieves all active rules
func (rm *RuleManager) GetActiveRules() []*Rule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	activeRules := []*Rule{}
	for _, rule := range rm.rules {
		if rule.Active {
			activeRules = append(activeRules, rule)
		}
	}

	// Sort by type and priority
	sort.Slice(activeRules, func(i, j int) bool {
		if activeRules[i].Type != activeRules[j].Type {
			return activeRules[i].Type < activeRules[j].Type
		}
		return activeRules[i].Priority > activeRules[j].Priority
	})

	return activeRules
}

// UpdateRule updates an existing rule
func (rm *RuleManager) UpdateRule(id string, updates map[string]interface{}) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, ok := rm.rules[id]
	if !ok {
		return fmt.Errorf("rule not found: %s", id)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "name":
			if name, ok := value.(string); ok {
				rule.Name = name
			}
		case "description":
			if desc, ok := value.(string); ok {
				rule.Description = desc
			}
		case "content":
			if content, ok := value.(string); ok {
				rule.Content = content
			}
		case "active":
			if active, ok := value.(bool); ok {
				rule.Active = active
			}
		case "priority":
			if priority, ok := value.(int); ok {
				rule.Priority = priority
			}
		case "metadata":
			if metadata, ok := value.(map[string]string); ok {
				for k, v := range metadata {
					rule.Metadata[k] = v
				}
			}
		}
	}

	rule.UpdatedAt = time.Now()
	return nil
}

// SaveCustomRule saves a custom rule to the rules directory
func (rm *RuleManager) SaveCustomRule(rule *Rule) error {
	if rm.rulesDir == "" {
		return errors.New("custom rules directory not configured")
	}

	// Ensure directory exists
	if err := os.MkdirAll(rm.rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
	}

	// Generate filename
	filename := strings.ReplaceAll(rule.Name, " ", "-") + ".yaml"
	filepath := filepath.Join(rm.rulesDir, filename)

	// Prepare rule data with metadata
	data := map[string]interface{}{
		"name":        rule.Name,
		"type":        string(rule.Type),
		"description": rule.Description,
		"priority":    rule.Priority,
		"version":     rule.Version,
		"active":      rule.Active,
		"metadata":    rule.Metadata,
		"content":     rule.Content,
	}

	// Marshal to YAML
	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	// Write file
	if err := os.WriteFile(filepath, content, 0644); err != nil {
		return fmt.Errorf("failed to write rule file: %w", err)
	}

	// Add to manager
	rm.addRule(rule)
	return nil
}

// DeleteCustomRule deletes a custom rule
func (rm *RuleManager) DeleteCustomRule(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, ok := rm.rules[id]
	if !ok {
		return fmt.Errorf("rule not found: %s", id)
	}

	// Only delete custom rules, not embedded ones
	if rm.rulesDir != "" {
		filename := strings.ReplaceAll(rule.Name, " ", "-") + ".yaml"
		filepath := filepath.Join(rm.rulesDir, filename)

		if _, err := os.Stat(filepath); err == nil {
			if err := os.Remove(filepath); err != nil {
				return fmt.Errorf("failed to delete rule file: %w", err)
			}
		}
	}

	// Remove from manager
	delete(rm.rules, id)

	// Remove from type index
	typeRules := rm.rulesByType[rule.Type]
	for i, r := range typeRules {
		if r.ID == id {
			rm.rulesByType[rule.Type] = append(typeRules[:i], typeRules[i+1:]...)
			break
		}
	}

	return nil
}

// ExportRules exports all rules to JSON
func (rm *RuleManager) ExportRules() ([]byte, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rules := make([]*Rule, 0, len(rm.rules))
	for _, rule := range rm.rules {
		rules = append(rules, rule)
	}

	return json.MarshalIndent(rules, "", "  ")
}

// ImportRules imports rules from JSON
func (rm *RuleManager) ImportRules(data []byte) error {
	var rules []*Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule %s: %w", rule.Name, err)
		}
		rm.addRule(rule)
	}

	return nil
}

// GetRuleContent returns the processed content of a rule
func (rm *RuleManager) GetRuleContent(ruleType RuleType) (string, error) {
	rules := rm.GetRulesByType(ruleType)
	if len(rules) == 0 {
		return "", fmt.Errorf("no rules found for type: %s", ruleType)
	}

	// Return the highest priority rule content
	return rules[0].Content, nil
}

// ListRules returns a summary of all rules
func (rm *RuleManager) ListRules() []map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	list := []map[string]interface{}{}
	for _, rule := range rm.rules {
		list = append(list, map[string]interface{}{
			"id":          rule.ID,
			"name":        rule.Name,
			"type":        rule.Type,
			"description": rule.Description,
			"priority":    rule.Priority,
			"active":      rule.Active,
			"version":     rule.Version,
			"updated_at":  rule.UpdatedAt,
		})
	}

	return list
}
