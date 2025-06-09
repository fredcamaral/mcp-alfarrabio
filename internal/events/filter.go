// Package events provides event filtering and routing logic
package events

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FilterEngine provides advanced event filtering capabilities
type FilterEngine struct {
	compiledRegexes map[string]*regexp.Regexp
	filterRules     map[string]*FilterRule
}

// FilterRule represents a complex filtering rule
type FilterRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  []*FilterCondition     `json:"conditions"`
	Logic       FilterLogic            `json:"logic"`
	Actions     []*FilterAction        `json:"actions"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// FilterCondition represents a single filtering condition
type FilterCondition struct {
	Field         string      `json:"field"`
	Operator      FilterOp    `json:"operator"`
	Value         interface{} `json:"value"`
	CaseSensitive bool        `json:"case_sensitive"`
}

// FilterAction represents an action to take when a filter matches
type FilterAction struct {
	Type       FilterActionType       `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// FilterLogic defines how multiple conditions are combined
type FilterLogic string

const (
	FilterLogicAND FilterLogic = "AND"
	FilterLogicOR  FilterLogic = "OR"
	FilterLogicNOT FilterLogic = "NOT"
)

// FilterOp defines filter operators
type FilterOp string

const (
	FilterOpEquals      FilterOp = "equals"
	FilterOpNotEquals   FilterOp = "not_equals"
	FilterOpContains    FilterOp = "contains"
	FilterOpNotContains FilterOp = "not_contains"
	FilterOpStartsWith  FilterOp = "starts_with"
	FilterOpEndsWith    FilterOp = "ends_with"
	FilterOpRegex       FilterOp = "regex"
	FilterOpIn          FilterOp = "in"
	FilterOpNotIn       FilterOp = "not_in"
	FilterOpGreaterThan FilterOp = "greater_than"
	FilterOpLessThan    FilterOp = "less_than"
	FilterOpBetween     FilterOp = "between"
	FilterOpExists      FilterOp = "exists"
	FilterOpNotExists   FilterOp = "not_exists"
	FilterOpEmpty       FilterOp = "empty"
	FilterOpNotEmpty    FilterOp = "not_empty"
)

// FilterActionType defines types of filter actions
type FilterActionType string

const (
	FilterActionAllow     FilterActionType = "allow"
	FilterActionDeny      FilterActionType = "deny"
	FilterActionTransform FilterActionType = "transform"
	FilterActionRoute     FilterActionType = "route"
	FilterActionTag       FilterActionType = "tag"
	FilterActionPriority  FilterActionType = "priority"
	FilterActionDelay     FilterActionType = "delay"
	FilterActionDuplicate FilterActionType = "duplicate"
)

// FilterResult represents the result of applying filters
type FilterResult struct {
	Allowed      bool                   `json:"allowed"`
	Transformed  *Event                 `json:"transformed"`
	Actions      []*FilterAction        `json:"actions"`
	MatchedRules []string               `json:"matched_rules"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewFilterEngine creates a new filter engine
func NewFilterEngine() *FilterEngine {
	return &FilterEngine{
		compiledRegexes: make(map[string]*regexp.Regexp),
		filterRules:     make(map[string]*FilterRule),
	}
}

// AddRule adds a filter rule to the engine
func (fe *FilterEngine) AddRule(rule *FilterRule) error {
	if rule.ID == "" {
		return errors.New("filter rule ID cannot be empty")
	}

	// Validate and compile regex patterns
	for _, condition := range rule.Conditions {
		if condition.Operator == FilterOpRegex {
			pattern, ok := condition.Value.(string)
			if !ok {
				return errors.New("regex condition value must be a string")
			}

			compiledRegex, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern: %v", err)
			}

			fe.compiledRegexes[rule.ID+"_"+condition.Field] = compiledRegex
		}
	}

	fe.filterRules[rule.ID] = rule
	return nil
}

// RemoveRule removes a filter rule from the engine
func (fe *FilterEngine) RemoveRule(ruleID string) {
	delete(fe.filterRules, ruleID)

	// Clean up compiled regexes for this rule
	for key := range fe.compiledRegexes {
		if strings.HasPrefix(key, ruleID+"_") {
			delete(fe.compiledRegexes, key)
		}
	}
}

// GetRule returns a filter rule by ID
func (fe *FilterEngine) GetRule(ruleID string) (*FilterRule, bool) {
	rule, exists := fe.filterRules[ruleID]
	return rule, exists
}

// GetAllRules returns all filter rules
func (fe *FilterEngine) GetAllRules() map[string]*FilterRule {
	// Return a copy to prevent external modification
	rules := make(map[string]*FilterRule)
	for id, rule := range fe.filterRules {
		rules[id] = rule
	}
	return rules
}

// ApplyFilters applies all filter rules to an event
func (fe *FilterEngine) ApplyFilters(event *Event) *FilterResult {
	result := &FilterResult{
		Allowed:      true,
		Transformed:  event,
		Actions:      make([]*FilterAction, 0),
		MatchedRules: make([]string, 0),
		Metadata:     make(map[string]interface{}),
	}

	// Get rules sorted by priority
	sortedRules := fe.getSortedRules()

	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		if fe.ruleMatches(event, rule) {
			result.MatchedRules = append(result.MatchedRules, rule.ID)

			// Apply rule actions
			for _, action := range rule.Actions {
				result.Actions = append(result.Actions, action)

				switch action.Type {
				case FilterActionDeny:
					result.Allowed = false
					return result // Early exit for deny actions

				case FilterActionTransform:
					result.Transformed = fe.applyTransformation(result.Transformed, action)

				case FilterActionTag:
					fe.applyTagAction(result.Transformed, action)

				case FilterActionPriority:
					fe.applyPriorityAction(result.Transformed, action)

				case FilterActionRoute:
					fe.applyRouteAction(result, action)
				}
			}
		}
	}

	return result
}

// QuickFilter applies a simple filter to an event
func (fe *FilterEngine) QuickFilter(event *Event, filter *EventFilter) bool {
	if filter == nil {
		return true
	}

	return event.Matches(filter)
}

// CreateSimpleFilter creates a simple event filter from parameters
func CreateSimpleFilter(eventTypes []EventType, actions []string, sources []string) *EventFilter {
	return &EventFilter{
		Types:   eventTypes,
		Actions: actions,
		Sources: sources,
	}
}

// CreateTimeRangeFilter creates a filter for events within a time range
func CreateTimeRangeFilter(after, before *time.Time) *EventFilter {
	return &EventFilter{
		After:  after,
		Before: before,
	}
}

// CreateRepositoryFilter creates a filter for specific repositories
func CreateRepositoryFilter(repositories []string) *EventFilter {
	return &EventFilter{
		Repositories: repositories,
	}
}

// CreateUserFilter creates a filter for specific users
func CreateUserFilter(userIDs []string) *EventFilter {
	return &EventFilter{
		UserIDs: userIDs,
	}
}

// CreateSessionFilter creates a filter for specific sessions
func CreateSessionFilter(sessionIDs []string) *EventFilter {
	return &EventFilter{
		SessionIDs: sessionIDs,
	}
}

// CreateTagFilter creates a filter for events with specific tags
func CreateTagFilter(tags []string) *EventFilter {
	return &EventFilter{
		Tags: tags,
	}
}

// CreateMetadataFilter creates a filter for events with specific metadata
func CreateMetadataFilter(metadata map[string]interface{}) *EventFilter {
	return &EventFilter{
		Metadata: metadata,
	}
}

// CombineFilters combines multiple filters with AND logic
func CombineFilters(filters ...*EventFilter) *EventFilter {
	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return filters[0]
	}

	combined := &EventFilter{}

	// Combine all filter criteria
	for _, filter := range filters {
		if filter == nil {
			continue
		}

		combined.Types = append(combined.Types, filter.Types...)
		combined.Actions = append(combined.Actions, filter.Actions...)
		combined.Sources = append(combined.Sources, filter.Sources...)
		combined.Repositories = append(combined.Repositories, filter.Repositories...)
		combined.SessionIDs = append(combined.SessionIDs, filter.SessionIDs...)
		combined.UserIDs = append(combined.UserIDs, filter.UserIDs...)
		combined.ClientIDs = append(combined.ClientIDs, filter.ClientIDs...)
		combined.Tags = append(combined.Tags, filter.Tags...)

		// Use the most restrictive time range
		if filter.After != nil && (combined.After == nil || filter.After.After(*combined.After)) {
			combined.After = filter.After
		}
		if filter.Before != nil && (combined.Before == nil || filter.Before.Before(*combined.Before)) {
			combined.Before = filter.Before
		}

		// Combine metadata
		if combined.Metadata == nil {
			combined.Metadata = make(map[string]interface{})
		}
		for key, value := range filter.Metadata {
			combined.Metadata[key] = value
		}
	}

	return combined
}

// ruleMatches checks if an event matches a filter rule
func (fe *FilterEngine) ruleMatches(event *Event, rule *FilterRule) bool {
	if len(rule.Conditions) == 0 {
		return true
	}

	switch rule.Logic {
	case FilterLogicAND:
		for _, condition := range rule.Conditions {
			if !fe.conditionMatches(event, condition) {
				return false
			}
		}
		return true

	case FilterLogicOR:
		for _, condition := range rule.Conditions {
			if fe.conditionMatches(event, condition) {
				return true
			}
		}
		return false

	case FilterLogicNOT:
		for _, condition := range rule.Conditions {
			if fe.conditionMatches(event, condition) {
				return false
			}
		}
		return true

	default:
		// Default to AND logic
		for _, condition := range rule.Conditions {
			if !fe.conditionMatches(event, condition) {
				return false
			}
		}
		return true
	}
}

// conditionMatches checks if an event field matches a filter condition
func (fe *FilterEngine) conditionMatches(event *Event, condition *FilterCondition) bool {
	fieldValue := fe.getFieldValue(event, condition.Field)

	switch condition.Operator {
	case FilterOpEquals:
		return fe.compareValues(fieldValue, condition.Value, condition.CaseSensitive) == 0

	case FilterOpNotEquals:
		return fe.compareValues(fieldValue, condition.Value, condition.CaseSensitive) != 0

	case FilterOpContains:
		return fe.stringContains(fieldValue, condition.Value, condition.CaseSensitive)

	case FilterOpNotContains:
		return !fe.stringContains(fieldValue, condition.Value, condition.CaseSensitive)

	case FilterOpStartsWith:
		return fe.stringStartsWith(fieldValue, condition.Value, condition.CaseSensitive)

	case FilterOpEndsWith:
		return fe.stringEndsWith(fieldValue, condition.Value, condition.CaseSensitive)

	case FilterOpRegex:
		if regex, exists := fe.compiledRegexes[condition.Field]; exists {
			return regex.MatchString(fmt.Sprintf("%v", fieldValue))
		}
		return false

	case FilterOpIn:
		return fe.valueInSlice(fieldValue, condition.Value)

	case FilterOpNotIn:
		return !fe.valueInSlice(fieldValue, condition.Value)

	case FilterOpExists:
		return fieldValue != nil

	case FilterOpNotExists:
		return fieldValue == nil

	case FilterOpEmpty:
		return fe.isEmpty(fieldValue)

	case FilterOpNotEmpty:
		return !fe.isEmpty(fieldValue)

	default:
		return false
	}
}

// getFieldValue extracts a field value from an event
func (fe *FilterEngine) getFieldValue(event *Event, field string) interface{} {
	switch field {
	case "id":
		return event.ID
	case "type":
		return event.Type
	case "action":
		return event.Action
	case "version":
		return event.Version
	case "timestamp":
		return event.Timestamp
	case "source":
		return event.Source
	case "repository":
		return event.Repository
	case "session_id":
		return event.SessionID
	case "user_id":
		return event.UserID
	case "client_id":
		return event.ClientID
	case "tags":
		return event.Tags
	case "correlation_id":
		return event.CorrelationID
	case "causation_id":
		return event.CausationID
	case "parent_id":
		return event.ParentID
	case "payload":
		return event.Payload
	default:
		// Check metadata
		if strings.HasPrefix(field, "metadata.") {
			metadataKey := strings.TrimPrefix(field, "metadata.")
			return event.Metadata[metadataKey]
		}
		// Check payload fields
		if strings.HasPrefix(field, "payload.") && event.Payload != nil {
			if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
				payloadKey := strings.TrimPrefix(field, "payload.")
				return payloadMap[payloadKey]
			}
		}
		return nil
	}
}

// compareValues compares two values with optional case sensitivity
func (fe *FilterEngine) compareValues(a, b interface{}, caseSensitive bool) int {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}

	return strings.Compare(aStr, bStr)
}

// stringContains checks if string a contains string b
func (fe *FilterEngine) stringContains(a, b interface{}, caseSensitive bool) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}

	return strings.Contains(aStr, bStr)
}

// stringStartsWith checks if string a starts with string b
func (fe *FilterEngine) stringStartsWith(a, b interface{}, caseSensitive bool) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}

	return strings.HasPrefix(aStr, bStr)
}

// stringEndsWith checks if string a ends with string b
func (fe *FilterEngine) stringEndsWith(a, b interface{}, caseSensitive bool) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}

	return strings.HasSuffix(aStr, bStr)
}

// valueInSlice checks if a value exists in a slice
func (fe *FilterEngine) valueInSlice(value, slice interface{}) bool {
	if sliceValue, ok := slice.([]interface{}); ok {
		for _, item := range sliceValue {
			if fe.compareValues(value, item, true) == 0 {
				return true
			}
		}
	}
	return false
}

// isEmpty checks if a value is empty
func (fe *FilterEngine) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	str := fmt.Sprintf("%v", value)
	return strings.TrimSpace(str) == ""
}

// getSortedRules returns filter rules sorted by priority (higher priority first)
func (fe *FilterEngine) getSortedRules() []*FilterRule {
	rules := make([]*FilterRule, 0, len(fe.filterRules))
	for _, rule := range fe.filterRules {
		rules = append(rules, rule)
	}

	// Simple insertion sort by priority (descending)
	for i := 1; i < len(rules); i++ {
		key := rules[i]
		j := i - 1
		for j >= 0 && rules[j].Priority < key.Priority {
			rules[j+1] = rules[j]
			j--
		}
		rules[j+1] = key
	}

	return rules
}

// applyTransformation applies a transformation action to an event
func (fe *FilterEngine) applyTransformation(event *Event, action *FilterAction) *Event {
	// Clone the event to avoid modifying the original
	transformed := event.Clone()

	// Apply transformations based on action parameters
	if newType, exists := action.Parameters["type"]; exists {
		if eventType, ok := newType.(EventType); ok {
			transformed.Type = eventType
		}
	}

	if newAction, exists := action.Parameters["action"]; exists {
		if actionStr, ok := newAction.(string); ok {
			transformed.Action = actionStr
		}
	}

	if newSource, exists := action.Parameters["source"]; exists {
		if sourceStr, ok := newSource.(string); ok {
			transformed.Source = sourceStr
		}
	}

	if addTags, exists := action.Parameters["add_tags"]; exists {
		if tags, ok := addTags.([]string); ok {
			transformed.Tags = append(transformed.Tags, tags...)
		}
	}

	if addMetadata, exists := action.Parameters["add_metadata"]; exists {
		if metadata, ok := addMetadata.(map[string]interface{}); ok {
			for key, value := range metadata {
				transformed.Metadata[key] = value
			}
		}
	}

	return transformed
}

// applyTagAction applies a tag action to an event
func (fe *FilterEngine) applyTagAction(event *Event, action *FilterAction) {
	if tags, exists := action.Parameters["tags"]; exists {
		if tagSlice, ok := tags.([]string); ok {
			event.Tags = append(event.Tags, tagSlice...)
		}
	}
}

// applyPriorityAction applies a priority action to an event
func (fe *FilterEngine) applyPriorityAction(event *Event, action *FilterAction) {
	if priority, exists := action.Parameters["priority"]; exists {
		event.Metadata["filter_priority"] = priority
	}
}

// applyRouteAction applies a route action to the filter result
func (fe *FilterEngine) applyRouteAction(result *FilterResult, action *FilterAction) {
	if route, exists := action.Parameters["route"]; exists {
		result.Metadata["route"] = route
	}

	if subscribers, exists := action.Parameters["subscribers"]; exists {
		result.Metadata["target_subscribers"] = subscribers
	}
}
