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

// Metadata field constants
const (
	FieldDescription = "description"
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
		return nil, errors.New("failed to load embedded rules: " + err.Error())
	}

	// Load custom rules from directory if it exists
	if rulesDir != "" {
		if err := rm.loadCustomRules(rulesDir); err != nil {
			return nil, errors.New("failed to load custom rules: " + err.Error())
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
			return errors.New("failed to read embedded rule " + path + ": " + err.Error())
		}

		rule, err := rm.parseRuleFile(path, content)
		if err != nil {
			return errors.New("failed to parse embedded rule " + path + ": " + err.Error())
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

		// Clean and validate the path
		cleanPath := filepath.Clean(path)

		// Security check: ensure path is within the rules directory
		if !strings.HasPrefix(cleanPath, filepath.Clean(dir)) {
			return errors.New("invalid rule path: path traversal not allowed")
		}

		content, err := os.ReadFile(cleanPath) // #nosec G304 -- Path is cleaned and validated above
		if err != nil {
			return errors.New("failed to read custom rule " + path + ": " + err.Error())
		}

		rule, err := rm.parseRuleFile(path, content)
		if err != nil {
			return errors.New("failed to parse custom rule " + path + ": " + err.Error())
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

	rule := rm.createBaseRule(nameWithoutExt, string(content))
	rm.determineRuleType(rule, nameWithoutExt)

	rm.parseYAMLFrontMatter(rule, content)

	if err := rule.Validate(); err != nil {
		return nil, err
	}

	return rule, nil
}

// createBaseRule creates a rule with default values
func (rm *RuleManager) createBaseRule(name, content string) *Rule {
	return &Rule{
		Name:      name,
		Content:   content,
		Version:   "1.0.0",
		Active:    true,
		Priority:  50, // Default priority
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// determineRuleType sets rule type and description based on filename
func (rm *RuleManager) determineRuleType(rule *Rule, nameWithoutExt string) {
	ruleTypes := map[string]struct {
		ruleType    RuleType
		description string
	}{
		"create-prd":          {RulePRDGeneration, "Rule for generating Product Requirements Documents"},
		"create-trd":          {RuleTRDGeneration, "Rule for generating Technical Requirements Documents"},
		"generate-main-tasks": {RuleTaskGeneration, "Rule for generating main tasks from PRD/TRD"},
		"generate-sub-tasks":  {RuleSubTaskGeneration, "Rule for generating sub-tasks from main tasks"},
		"complexity":          {RuleComplexityAnalysis, "Rule for analyzing task complexity"},
		"validation":          {RuleValidation, "Rule for validating documents"},
	}

	for key, config := range ruleTypes {
		if strings.Contains(nameWithoutExt, key) {
			rule.Type = config.ruleType
			rule.Description = config.description
			return
		}
	}

	// Default type
	rule.Type = RulePRDGeneration
}

// parseYAMLFrontMatter extracts YAML front matter from content
func (rm *RuleManager) parseYAMLFrontMatter(rule *Rule, content []byte) {
	if !strings.HasPrefix(string(content), "---") {
		return
	}

	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		return
	}

	var metadata map[string]interface{}
	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		return // Continue with default values if YAML parsing fails
	}

	rm.extractMetadataFields(rule, metadata)
	rule.Content = strings.TrimSpace(parts[2])
}

// extractMetadataFields extracts known fields from metadata
func (rm *RuleManager) extractMetadataFields(rule *Rule, metadata map[string]interface{}) {
	metadataFields := map[string]func(interface{}){
		"name": func(v interface{}) {
			if name, ok := v.(string); ok {
				rule.Name = name
			}
		},
		FieldDescription: func(v interface{}) {
			if desc, ok := v.(string); ok {
				rule.Description = desc
			}
		},
		"type": func(v interface{}) {
			if typeStr, ok := v.(string); ok {
				rule.Type = RuleType(typeStr)
			}
		},
		"priority": func(v interface{}) {
			if priority, ok := v.(int); ok {
				rule.Priority = priority
			}
		},
		"version": func(v interface{}) {
			if version, ok := v.(string); ok {
				rule.Version = version
			}
		},
	}

	// Extract known fields
	for field, extractor := range metadataFields {
		if value, exists := metadata[field]; exists {
			extractor(value)
		}
	}

	// Store remaining fields as custom metadata
	for k, v := range metadata {
		if _, isKnown := metadataFields[k]; !isKnown {
			rule.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}
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
		return nil, errors.New("rule not found: " + id)
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

	return nil, errors.New("rule not found: " + name)
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
		return errors.New("rule not found: " + id)
	}

	// Apply updates using field updaters
	for key, value := range updates {
		rm.updateRuleField(rule, key, value)
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
	if err := os.MkdirAll(rm.rulesDir, 0o750); err != nil {
		return errors.New("failed to create rules directory: " + err.Error())
	}

	// Generate filename
	filename := strings.ReplaceAll(rule.Name, " ", "-") + ".yaml"
	filePath := filepath.Join(rm.rulesDir, filename)

	// Prepare rule data with metadata
	data := map[string]interface{}{
		"name":           rule.Name,
		"type":           string(rule.Type),
		FieldDescription: rule.Description,
		"priority":       rule.Priority,
		"version":        rule.Version,
		"active":         rule.Active,
		"metadata":       rule.Metadata,
		"content":        rule.Content,
	}

	// Marshal to YAML
	content, err := yaml.Marshal(data)
	if err != nil {
		return errors.New("failed to marshal rule: " + err.Error())
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		return errors.New("failed to write rule file: " + err.Error())
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
		return errors.New("rule not found: " + id)
	}

	// Only delete custom rules, not embedded ones
	if rm.rulesDir != "" {
		filename := strings.ReplaceAll(rule.Name, " ", "-") + ".yaml"
		filePath := filepath.Join(rm.rulesDir, filename)

		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				return errors.New("failed to delete rule file: " + err.Error())
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
		return errors.New("failed to unmarshal rules: " + err.Error())
	}

	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			return errors.New("invalid rule " + rule.Name + ": " + err.Error())
		}
		rm.addRule(rule)
	}

	return nil
}

// GetRuleContent returns the processed content of a rule
func (rm *RuleManager) GetRuleContent(ruleType RuleType) (string, error) {
	rules := rm.GetRulesByType(ruleType)
	if len(rules) == 0 {
		return "", errors.New("no rules found for type: " + string(ruleType))
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
			"id":             rule.ID,
			"name":           rule.Name,
			"type":           rule.Type,
			FieldDescription: rule.Description,
			"priority":       rule.Priority,
			"active":         rule.Active,
			"version":        rule.Version,
			"updated_at":     rule.UpdatedAt,
		})
	}

	return list
}

// updateRuleField updates a specific field of a rule
func (rm *RuleManager) updateRuleField(rule *Rule, key string, value interface{}) {
	switch key {
	case "name":
		rm.updateStringField(&rule.Name, value)
	case FieldDescription:
		rm.updateStringField(&rule.Description, value)
	case "content":
		rm.updateStringField(&rule.Content, value)
	case "active":
		rm.updateBoolField(&rule.Active, value)
	case "priority":
		rm.updateIntField(&rule.Priority, value)
	case "metadata":
		rm.updateMetadataField(rule, value)
	}
}

// updateStringField updates a string field if value is valid
func (rm *RuleManager) updateStringField(field *string, value interface{}) {
	if str, ok := value.(string); ok {
		*field = str
	}
}

// updateBoolField updates a boolean field if value is valid
func (rm *RuleManager) updateBoolField(field *bool, value interface{}) {
	if b, ok := value.(bool); ok {
		*field = b
	}
}

// updateIntField updates an integer field if value is valid
func (rm *RuleManager) updateIntField(field *int, value interface{}) {
	if i, ok := value.(int); ok {
		*field = i
	}
}

// updateMetadataField updates metadata if value is valid
func (rm *RuleManager) updateMetadataField(rule *Rule, value interface{}) {
	if metadata, ok := value.(map[string]string); ok {
		for k, v := range metadata {
			rule.Metadata[k] = v
		}
	}
}
