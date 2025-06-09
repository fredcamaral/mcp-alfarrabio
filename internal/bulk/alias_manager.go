// Package bulk provides functionality for bulk operations and memory aliasing.
// It includes alias management for flexible memory referencing and bulk data processing.
package bulk

import (
	"context"
	"errors"
	"fmt"
	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/pkg/types"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// AliasType represents the type of alias
type AliasType string

const (
	// AliasTypeTag represents tag-based aliases (e.g., "@bug-fixes")
	AliasTypeTag AliasType = "tag" // Tag-based alias (e.g., "@bug-fixes")
	// AliasTypeShortcut represents custom shortcuts (e.g., "#auth-module")
	AliasTypeShortcut   AliasType = "shortcut"   // Custom shortcut (e.g., "#auth-module")
	AliasTypeQuery      AliasType = "query"      // Saved query (e.g., "!recent-errors")
	AliasTypeCollection AliasType = "collection" // Named collection (e.g., "&deployment-notes")
)

// Alias represents a memory alias for flexible referencing
type Alias struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Type         AliasType     `json:"type"`
	Description  string        `json:"description,omitempty"`
	Target       AliasTarget   `json:"target"`
	Metadata     AliasMetadata `json:"metadata"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	AccessCount  int           `json:"access_count"`
	LastAccessed *time.Time    `json:"last_accessed,omitempty"`
}

// AliasTarget defines what the alias points to
type AliasTarget struct {
	Type       TargetType        `json:"type"`
	ChunkIDs   []string          `json:"chunk_ids,omitempty"`
	Query      *QueryTarget      `json:"query,omitempty"`
	Filter     *FilterTarget     `json:"filter,omitempty"`
	Collection *CollectionTarget `json:"collection,omitempty"`
}

// TargetType represents the type of alias target
type TargetType string

const (
	// TargetTypeChunks represents direct chunk references
	TargetTypeChunks TargetType = "chunks" // Direct chunk references
	// TargetTypeQuery represents dynamic query targets
	TargetTypeQuery      TargetType = "query"      // Dynamic query
	TargetTypeFilter     TargetType = "filter"     // Filter criteria
	TargetTypeCollection TargetType = "collection" // Named collection
)

// QueryTarget represents a dynamic query target
type QueryTarget struct {
	Query        string            `json:"query"`
	Repository   *string           `json:"repository,omitempty"`
	ChunkTypes   []types.ChunkType `json:"chunk_types,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	DateRange    *DateRange        `json:"date_range,omitempty"`
	MaxResults   int               `json:"max_results,omitempty"`
	MinRelevance float64           `json:"min_relevance,omitempty"`
}

// FilterTarget represents a filter-based target
type FilterTarget struct {
	Repository   *string            `json:"repository,omitempty"`
	SessionIDs   []string           `json:"session_ids,omitempty"`
	ChunkTypes   []types.ChunkType  `json:"chunk_types,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	Outcomes     []types.Outcome    `json:"outcomes,omitempty"`
	Difficulties []types.Difficulty `json:"difficulties,omitempty"`
	DateRange    *DateRange         `json:"date_range,omitempty"`
	ContentMatch *string            `json:"content_match,omitempty"` // Regex pattern
}

// CollectionTarget represents a named collection
type CollectionTarget struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	ChunkIDs    []string               `json:"chunk_ids"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DateRange represents a date range for filtering
type DateRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// AliasMetadata contains metadata about the alias
type AliasMetadata struct {
	CreatedBy  string                 `json:"created_by,omitempty"`
	Repository string                 `json:"repository,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	IsPublic   bool                   `json:"is_public"`
	IsFavorite bool                   `json:"is_favorite"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

// AliasResult represents the result of resolving an alias
type AliasResult struct {
	Alias      Alias                     `json:"alias"`
	Chunks     []types.ConversationChunk `json:"chunks"`
	TotalFound int                       `json:"total_found"`
	Message    string                    `json:"message,omitempty"`
	Warnings   []string                  `json:"warnings,omitempty"`
}

// AliasManager handles memory aliases for flexible referencing
type AliasManager struct {
	storage    storage.VectorStore
	aliases    map[string]*Alias
	aliasesMux sync.RWMutex
	logger     *log.Logger
}

// NewAliasManager creates a new alias manager
func NewAliasManager(vectorStore storage.VectorStore, logger *log.Logger) *AliasManager {
	if logger == nil {
		logger = log.New(log.Writer(), "[AliasManager] ", log.LstdFlags)
	}

	return &AliasManager{
		storage: vectorStore,
		aliases: make(map[string]*Alias),
		logger:  logger,
	}
}

// CreateAlias creates a new alias
func (am *AliasManager) CreateAlias(ctx context.Context, alias *Alias) (*Alias, error) {
	// Validate alias
	if err := am.validateAlias(alias); err != nil {
		return nil, errors.New("invalid alias: " + err.Error())
	}

	// Check for name conflicts
	if existing := am.findAliasByName(alias.Name); existing != nil {
		return nil, errors.New("alias with name '" + alias.Name + "' already exists")
	}

	// Generate ID if not provided
	if alias.ID == "" {
		alias.ID = am.generateAliasID(alias.Name)
	}

	// Set timestamps
	now := time.Now().UTC()
	alias.CreatedAt = now
	alias.UpdatedAt = now

	// Store alias
	am.aliasesMux.Lock()
	am.aliases[alias.ID] = alias
	am.aliasesMux.Unlock()

	// Persist to storage (would be implemented based on storage backend)
	if err := am.persistAlias(ctx, alias); err != nil {
		am.logger.Printf("Warning: failed to persist alias %s: %v", alias.ID, err)
	}

	return alias, nil
}

// UpdateAlias updates an existing alias
func (am *AliasManager) UpdateAlias(ctx context.Context, aliasID string, updates *Alias) (*Alias, error) {
	am.aliasesMux.Lock()
	existing, exists := am.aliases[aliasID]
	if !exists {
		am.aliasesMux.Unlock()
		return nil, errors.New("alias " + aliasID + " not found")
	}

	// Preserve creation info
	updates.ID = existing.ID
	updates.CreatedAt = existing.CreatedAt
	updates.AccessCount = existing.AccessCount
	updates.LastAccessed = existing.LastAccessed
	updates.UpdatedAt = time.Now().UTC()

	// Validate updates
	if err := am.validateAlias(updates); err != nil {
		am.aliasesMux.Unlock()
		return nil, errors.New("invalid alias update: " + err.Error())
	}

	// Check for name conflicts (exclude current alias)
	if existing.Name != updates.Name {
		if conflicting := am.findAliasByNameExcluding(updates.Name, aliasID); conflicting != nil {
			am.aliasesMux.Unlock()
			return nil, errors.New("alias with name '" + updates.Name + "' already exists")
		}
	}

	// Update alias
	am.aliases[aliasID] = updates
	am.aliasesMux.Unlock()

	// Persist to storage
	if err := am.persistAlias(ctx, updates); err != nil {
		am.logger.Printf("Warning: failed to persist alias update %s: %v", aliasID, err)
	}

	return updates, nil
}

// DeleteAlias deletes an alias
func (am *AliasManager) DeleteAlias(ctx context.Context, aliasID string) error {
	am.aliasesMux.Lock()
	_, exists := am.aliases[aliasID]
	if !exists {
		am.aliasesMux.Unlock()
		return errors.New("alias " + aliasID + " not found")
	}

	delete(am.aliases, aliasID)
	am.aliasesMux.Unlock()

	// Remove from persistent storage
	if err := am.removePersistedAlias(ctx, aliasID); err != nil {
		am.logger.Printf("Warning: failed to remove persisted alias %s: %v", aliasID, err)
	}

	return nil
}

// GetAlias retrieves an alias by ID
func (am *AliasManager) GetAlias(aliasID string) (*Alias, error) {
	am.aliasesMux.RLock()
	defer am.aliasesMux.RUnlock()

	alias, exists := am.aliases[aliasID]
	if !exists {
		return nil, errors.New("alias " + aliasID + " not found")
	}

	return alias, nil
}

// ResolveAlias resolves an alias reference and returns the matching chunks
func (am *AliasManager) ResolveAlias(ctx context.Context, aliasName string) (*AliasResult, error) {
	// Find alias by name or ID
	alias := am.findAliasByName(aliasName)
	if alias == nil {
		alias = am.findAliasByID(aliasName)
	}
	if alias == nil {
		return nil, errors.New("alias '" + aliasName + "' not found")
	}

	// Update access tracking
	am.trackAccess(alias.ID)

	// Resolve based on target type
	chunks, err := am.resolveTarget(ctx, alias.Target)
	if err != nil {
		return nil, errors.New("failed to resolve alias target: " + err.Error())
	}

	result := &AliasResult{
		Alias:      *alias,
		Chunks:     chunks,
		TotalFound: len(chunks),
	}

	// Add helpful message
	result.Message = am.generateResultMessage(alias, len(chunks))

	return result, nil
}

// ResolveAliasReference resolves alias references in text (e.g., "@bug-fixes", "#auth")
func (am *AliasManager) ResolveAliasReference(ctx context.Context, text string) (map[string]*AliasResult, error) {
	// Find alias references in text
	references := am.findAliasReferences(text)
	results := make(map[string]*AliasResult)

	for _, ref := range references {
		result, err := am.ResolveAlias(ctx, ref)
		if err != nil {
			am.logger.Printf("Failed to resolve alias reference '%s': %v", ref, err)
			continue
		}
		results[ref] = result
	}

	return results, nil
}

// ListAliases lists aliases with optional filtering
func (am *AliasManager) ListAliases(filter AliasListFilter) ([]*Alias, error) {
	am.aliasesMux.RLock()
	defer am.aliasesMux.RUnlock()

	var aliases []*Alias
	for _, alias := range am.aliases {
		if am.matchesFilter(alias, filter) {
			aliases = append(aliases, alias)
		}
	}

	// Sort by access count (most used first) or creation date
	sort.Slice(aliases, func(i, j int) bool {
		if filter.SortBy == "usage" {
			return aliases[i].AccessCount > aliases[j].AccessCount
		}
		return aliases[i].CreatedAt.After(aliases[j].CreatedAt)
	})

	// Apply limit
	if filter.Limit > 0 && len(aliases) > filter.Limit {
		aliases = aliases[:filter.Limit]
	}

	return aliases, nil
}

// AliasListFilter defines filtering options for listing aliases
type AliasListFilter struct {
	Type       *AliasType `json:"type,omitempty"`
	Repository *string    `json:"repository,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
	Query      *string    `json:"query,omitempty"` // Search in name/description
	SortBy     string     `json:"sort_by"`         // "usage", "created", "updated"
	Limit      int        `json:"limit,omitempty"`
}

// Helper methods

func (am *AliasManager) validateAlias(alias *Alias) error {
	if alias.Name == "" {
		return errors.New("alias name cannot be empty")
	}

	if !am.isValidAliasName(alias.Name) {
		return errors.New("invalid alias name format: " + alias.Name)
	}

	if alias.Type == "" {
		return errors.New("alias type cannot be empty")
	}

	return am.validateTarget(alias.Target)
}

func (am *AliasManager) isValidAliasName(name string) bool {
	// Allow alphanumeric, hyphens, underscores, and certain prefixes
	pattern := regexp.MustCompile(`^[@#!&]?[a-zA-Z0-9_-]+$`)
	return pattern.MatchString(name)
}

func (am *AliasManager) validateTarget(target AliasTarget) error {
	switch target.Type {
	case TargetTypeChunks:
		if len(target.ChunkIDs) == 0 {
			return errors.New("chunk target must have at least one chunk ID")
		}
	case TargetTypeQuery:
		if target.Query == nil || target.Query.Query == "" {
			return errors.New("query target must have a query")
		}
	case TargetTypeFilter:
		if target.Filter == nil {
			return errors.New("filter target must have filter criteria")
		}
	case TargetTypeCollection:
		if target.Collection == nil || target.Collection.Name == "" {
			return errors.New("collection target must have a name")
		}
	default:
		return errors.New("invalid target type: " + string(target.Type))
	}
	return nil
}

func (am *AliasManager) generateAliasID(name string) string {
	// Remove special characters and generate ID
	cleanName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(name, "")
	return fmt.Sprintf("alias_%s_%d", cleanName, time.Now().Unix())
}

func (am *AliasManager) findAliasByName(name string) *Alias {
	am.aliasesMux.RLock()
	defer am.aliasesMux.RUnlock()

	for _, alias := range am.aliases {
		if alias.Name == name {
			return alias
		}
	}
	return nil
}

func (am *AliasManager) findAliasByID(id string) *Alias {
	am.aliasesMux.RLock()
	defer am.aliasesMux.RUnlock()

	if alias, exists := am.aliases[id]; exists {
		return alias
	}
	return nil
}

func (am *AliasManager) findAliasByNameExcluding(name, excludeID string) *Alias {
	am.aliasesMux.RLock()
	defer am.aliasesMux.RUnlock()

	for id, alias := range am.aliases {
		if id != excludeID && alias.Name == name {
			return alias
		}
	}
	return nil
}

func (am *AliasManager) trackAccess(aliasID string) {
	am.aliasesMux.Lock()
	defer am.aliasesMux.Unlock()

	if alias, exists := am.aliases[aliasID]; exists {
		alias.AccessCount++
		now := time.Now().UTC()
		alias.LastAccessed = &now
	}
}

func (am *AliasManager) resolveTarget(ctx context.Context, target AliasTarget) ([]types.ConversationChunk, error) {
	switch target.Type {
	case TargetTypeChunks:
		return am.resolveChunkTarget(ctx, target.ChunkIDs)
	case TargetTypeQuery:
		return am.resolveQueryTarget(ctx, target.Query)
	case TargetTypeFilter:
		return am.resolveFilterTarget(ctx, target.Filter)
	case TargetTypeCollection:
		return am.resolveCollectionTarget(ctx, target.Collection)
	default:
		return nil, errors.New("unsupported target type: " + string(target.Type))
	}
}

func (am *AliasManager) resolveChunkTarget(ctx context.Context, chunkIDs []string) ([]types.ConversationChunk, error) {
	chunks := make([]types.ConversationChunk, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		chunk, err := am.storage.GetByID(ctx, id)
		if err != nil {
			am.logger.Printf("Warning: failed to retrieve chunk %s: %v", id, err)
			continue
		}
		chunks = append(chunks, *chunk)
	}
	return chunks, nil
}

func (am *AliasManager) resolveQueryTarget(_ context.Context, query *QueryTarget) ([]types.ConversationChunk, error) {
	// Convert QueryTarget to MemoryQuery
	_ = &types.MemoryQuery{
		Query:             query.Query,
		Repository:        query.Repository,
		Types:             query.ChunkTypes,
		MinRelevanceScore: query.MinRelevance,
		Limit:             query.MaxResults,
	}

	// This would use the search functionality - for now return empty
	// In real implementation, would call storage.Search or similar
	return []types.ConversationChunk{}, nil
}

func (am *AliasManager) resolveFilterTarget(_ context.Context, _ *FilterTarget) ([]types.ConversationChunk, error) {
	// This would apply the filter criteria to find matching chunks
	// For now, return empty - real implementation would query storage
	return []types.ConversationChunk{}, nil
}

func (am *AliasManager) resolveCollectionTarget(ctx context.Context, collection *CollectionTarget) ([]types.ConversationChunk, error) {
	return am.resolveChunkTarget(ctx, collection.ChunkIDs)
}

func (am *AliasManager) findAliasReferences(text string) []string {
	// Find alias references: @tag, #shortcut, !query, &collection
	patterns := []string{
		`@[a-zA-Z0-9_-]+`, // Tag aliases
		`#[a-zA-Z0-9_-]+`, // Shortcut aliases
		`![a-zA-Z0-9_-]+`, // Query aliases
		`&[a-zA-Z0-9_-]+`, // Collection aliases
	}

	var references []string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(text, -1)
		references = append(references, matches...)
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, ref := range references {
		if !seen[ref] {
			seen[ref] = true
			unique = append(unique, ref)
		}
	}

	return unique
}

func (am *AliasManager) matchesFilter(alias *Alias, filter AliasListFilter) bool {
	// Type filter
	if filter.Type != nil && alias.Type != *filter.Type {
		return false
	}

	// Repository filter
	if filter.Repository != nil && alias.Metadata.Repository != *filter.Repository {
		return false
	}

	// Tags filter
	if len(filter.Tags) > 0 {
		found := false
		for _, filterTag := range filter.Tags {
			for _, aliasTag := range alias.Metadata.Tags {
				if aliasTag == filterTag {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Query filter (search in name/description)
	if filter.Query != nil {
		query := strings.ToLower(*filter.Query)
		if !strings.Contains(strings.ToLower(alias.Name), query) &&
			!strings.Contains(strings.ToLower(alias.Description), query) {
			return false
		}
	}

	return true
}

func (am *AliasManager) generateResultMessage(alias *Alias, resultCount int) string {
	if resultCount == 0 {
		return fmt.Sprintf("No memories found for alias '%s'", alias.Name)
	}
	return fmt.Sprintf("Found %d memories for alias '%s' (%s)",
		resultCount, alias.Name, alias.Type)
}

// Persistence methods (would be implemented based on storage backend)

func (am *AliasManager) persistAlias(_ context.Context, _ *Alias) error {
	// This would persist the alias to storage
	// Implementation depends on the storage backend
	return nil
}

func (am *AliasManager) removePersistedAlias(_ context.Context, _ string) error {
	// This would remove the alias from persistent storage
	return nil
}
