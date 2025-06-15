// Package intelligence provides AI-powered pattern recognition, learning engines,
// conflict detection, and knowledge graph capabilities for the MCP Memory Server.
package intelligence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	internalAI "lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/embeddings"
	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/pkg/types"
)

// PatternType represents different types of patterns we can recognize
type PatternType string

const (
	PatternTypeCode          PatternType = "code"
	PatternTypeWorkflow      PatternType = "workflow"
	PatternTypeArchitectural PatternType = "architectural"
	PatternTypeBehavioral    PatternType = "behavioral"
	PatternTypeError         PatternType = "error"
	PatternTypeOptimization  PatternType = "optimization"
	PatternTypeRefactoring   PatternType = "refactoring"
)

// ConfidenceLevel represents pattern confidence levels
type ConfidenceLevel string

const (
	ConfidenceVeryLow  ConfidenceLevel = "very_low"
	ConfidenceLow      ConfidenceLevel = "low"
	ConfidenceMedium   ConfidenceLevel = "medium"
	ConfidenceHigh     ConfidenceLevel = "high"
	ConfidenceVeryHigh ConfidenceLevel = "very_high"
)

// ValidationStatus represents pattern validation status
type ValidationStatus string

const (
	ValidationUnvalidated ValidationStatus = "unvalidated"
	ValidationPending     ValidationStatus = "pending"
	ValidationValidated   ValidationStatus = "validated"
	ValidationInvalidated ValidationStatus = "invalidated"
	ValidationEvolved     ValidationStatus = "evolved"
)

// Pattern represents a recognized conversation pattern
type Pattern struct {
	ID               string                 `json:"id"`
	Type             PatternType            `json:"type"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	Signature        map[string]interface{} `json:"signature"`
	Keywords         []string               `json:"keywords"`
	RepositoryURL    string                 `json:"repository_url"`
	FilePatterns     []string               `json:"file_patterns"`
	Language         string                 `json:"language"`
	ConfidenceScore  float64                `json:"confidence_score"`
	ConfidenceLevel  ConfidenceLevel        `json:"confidence_level"`
	ValidationStatus ValidationStatus       `json:"validation_status"`
	OccurrenceCount  int                    `json:"occurrence_count"`
	PositiveFeedback int                    `json:"positive_feedback_count"`
	NegativeFeedback int                    `json:"negative_feedback_count"`
	LastSeenAt       *time.Time             `json:"last_seen_at"`
	ParentPatternID  *string                `json:"parent_pattern_id"`
	EvolutionReason  string                 `json:"evolution_reason"`
	Version          int                    `json:"version"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`

	// Runtime fields (not stored in DB)
	Steps           []PatternStep    `json:"steps,omitempty"`
	RelatedPatterns []string         `json:"related_patterns,omitempty"`
	Examples        []PatternExample `json:"examples,omitempty"`
	Embeddings      []float64        `json:"embeddings,omitempty"`
}

// PatternStep represents a step in a pattern sequence
type PatternStep struct {
	Order       int            `json:"order"`
	Action      string         `json:"action"`
	Description string         `json:"description"`
	Optional    bool           `json:"optional"`
	Confidence  float64        `json:"confidence"`
	Context     map[string]any `json:"context"`
}

// PatternExample represents an example instance of a pattern
type PatternExample struct {
	ID           string                    `json:"id"`
	ChunkIDs     []string                  `json:"chunk_ids"`
	Conversation []types.ConversationChunk `json:"conversation"`
	Outcome      PatternOutcome            `json:"outcome"`
	Confidence   float64                   `json:"confidence"`
	Timestamp    time.Time                 `json:"timestamp"`
}

// PatternOutcome represents the result of applying a pattern
type PatternOutcome string

const (
	OutcomeSuccess     PatternOutcome = "success"
	OutcomePartial     PatternOutcome = "partial"
	OutcomeFailure     PatternOutcome = "failure"
	OutcomeInterrupted PatternOutcome = "interrupted"
	OutcomeUnknown     PatternOutcome = "unknown"
)

// PatternMatcher defines the interface for pattern matching algorithms
type PatternMatcher interface {
	MatchPattern(chunks []types.ConversationChunk, pattern *Pattern) float64
	ExtractFeatures(chunks []types.ConversationChunk) map[string]any
	IdentifySequence(chunks []types.ConversationChunk) []PatternStep
}

// SequenceRecognizer recognizes sequential patterns in conversations
type SequenceRecognizer interface {
	RecognizeSequence(chunks []types.ConversationChunk) ([]Pattern, error)
	LearnFromSequence(chunks []types.ConversationChunk, outcome PatternOutcome) error
}

// PatternOccurrence represents where a pattern was detected
type PatternOccurrence struct {
	ID                 string                 `json:"id"`
	PatternID          string                 `json:"pattern_id"`
	RepositoryURL      string                 `json:"repository_url"`
	FilePath           string                 `json:"file_path"`
	LineStart          int                    `json:"line_start"`
	LineEnd            int                    `json:"line_end"`
	CodeSnippet        string                 `json:"code_snippet"`
	SurroundingContext string                 `json:"surrounding_context"`
	DetectionScore     float64                `json:"detection_score"`
	DetectionMethod    string                 `json:"detection_method"`
	SessionID          string                 `json:"session_id"`
	ChunkID            string                 `json:"chunk_id"`
	Metadata           map[string]interface{} `json:"metadata"`
	DetectedAt         time.Time              `json:"detected_at"`
}

// PatternRelationship represents relationships between patterns
type PatternRelationship struct {
	ID               string                 `json:"id"`
	SourcePatternID  string                 `json:"source_pattern_id"`
	TargetPatternID  string                 `json:"target_pattern_id"`
	RelationshipType string                 `json:"relationship_type"` // extends, conflicts_with, complements, alternative_to
	Strength         float64                `json:"strength"`
	Confidence       float64                `json:"confidence"`
	Context          string                 `json:"context"`
	Examples         []interface{}          `json:"examples"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
}

// AIService interface for AI operations needed by pattern engine
type AIService interface {
	ProcessRequest(ctx context.Context, req *internalAI.Request) (*internalAI.Response, error)
}

// PatternStorage interface for storing and retrieving patterns
type PatternStorage interface {
	StorePattern(ctx context.Context, pattern *Pattern) error
	GetPattern(ctx context.Context, id string) (*Pattern, error)
	ListPatterns(ctx context.Context, patternType *PatternType) ([]Pattern, error)
	UpdatePattern(ctx context.Context, pattern *Pattern) error
	DeletePattern(ctx context.Context, id string) error
	SearchPatterns(ctx context.Context, query string, limit int) ([]Pattern, error)

	// New methods for production
	StoreOccurrence(ctx context.Context, occurrence *PatternOccurrence) error
	GetOccurrences(ctx context.Context, patternID string, limit int) ([]PatternOccurrence, error)
	StoreRelationship(ctx context.Context, relationship *PatternRelationship) error
	GetRelationships(ctx context.Context, patternID string) ([]PatternRelationship, error)
	UpdateConfidence(ctx context.Context, patternID string, isPositive bool) error
	GetPatternStatistics(ctx context.Context) (map[string]interface{}, error)
}

// batchItem represents an item for batch processing
type batchItem struct {
	chunks   []types.ConversationChunk
	outcome  PatternOutcome
	callback func(error)
}

// patternMetrics tracks pattern engine metrics
type patternMetrics struct {
	patternsDetected int64
	patternsLearned  int64
	aiCalls          int64
	cacheHits        int64
	cacheMisses      int64
	processingTime   time.Duration
	mu               sync.RWMutex
}

// PatternEngineConfig holds configuration for the pattern engine
type PatternEngineConfig struct {
	MinConfidence      float64
	MaxPatterns        int
	LearningEnabled    bool
	EvolutionThreshold float64
	BatchSize          int
	BatchInterval      time.Duration
	EnableCaching      bool
}

// DefaultPatternEngineConfig returns default configuration
func DefaultPatternEngineConfig() *PatternEngineConfig {
	return &PatternEngineConfig{
		MinConfidence:      0.6,
		MaxPatterns:        1000,
		LearningEnabled:    true,
		EvolutionThreshold: 0.8,
		BatchSize:          10,
		BatchInterval:      5 * time.Second,
		EnableCaching:      true,
	}
}

// PatternEngine is the main engine for pattern recognition and learning
type PatternEngine struct {
	db               *sql.DB
	storage          PatternStorage
	aiService        AIService
	embeddingService embeddings.EmbeddingService
	matcher          PatternMatcher
	recognizer       SequenceRecognizer
	logger           logging.Logger

	// Caching
	embeddingCache sync.Map // map[string][]float64

	// Batch processing
	batchQueue chan *batchItem
	stopBatch  chan struct{}
	batchWg    sync.WaitGroup

	// Configuration
	config *PatternEngineConfig

	// Pattern matching regexes
	patternRegexes map[string]*regexp.Regexp

	// Metrics
	metrics *patternMetrics
}

// NewPatternEngineWithDependencies creates a new pattern recognition engine with all dependencies
func NewPatternEngineWithDependencies(
	db *sql.DB,
	storage PatternStorage,
	aiService AIService,
	embeddingService embeddings.EmbeddingService,
	logger logging.Logger,
	config *PatternEngineConfig,
) *PatternEngine {
	if config == nil {
		config = DefaultPatternEngineConfig()
	}

	engine := &PatternEngine{
		db:               db,
		storage:          storage,
		aiService:        aiService,
		embeddingService: embeddingService,
		logger:           logger,
		config:           config,
		batchQueue:       make(chan *batchItem, config.BatchSize*2),
		stopBatch:        make(chan struct{}),
		metrics:          &patternMetrics{},
	}

	// Initialize pattern regexes
	engine.patternRegexes = map[string]*regexp.Regexp{
		"problem":     regexp.MustCompile(`(?i)(error|issue|problem|bug|fail|broken|not working|doesn't work)`),
		"solution":    regexp.MustCompile(`(?i)(fix|solve|resolve|solution|fixed|resolved|working)`),
		"error":       regexp.MustCompile(`(?i)(error:|exception:|fatal:|panic:|warning:)`),
		"command":     regexp.MustCompile(`(?i)(run|execute|install|build|test|deploy)`),
		"code":        regexp.MustCompile("```[\\s\\S]*?```"),
		"import":      regexp.MustCompile(`(?i)(import|require|include|use)\s+[\w./"-]+`),
		"function":    regexp.MustCompile(`(?i)(func|function|def|method|class)\s+\w+`),
		"variable":    regexp.MustCompile(`(?i)(var|let|const|string|int|float|bool)\s+\w+`),
		"conditional": regexp.MustCompile(`(?i)(if|else|switch|case|when|unless)`),
		"loop":        regexp.MustCompile(`(?i)(for|while|foreach|loop|iterate)`),
		"async":       regexp.MustCompile(`(?i)(async|await|promise|future|goroutine|channel)`),
		"test":        regexp.MustCompile(`(?i)(test|assert|expect|should|describe|it\()`),
		"api":         regexp.MustCompile(`(?i)(api|endpoint|route|rest|graphql|grpc)`),
		"database":    regexp.MustCompile(`(?i)(select|insert|update|delete|create table|index|query)`),
		"config":      regexp.MustCompile(`(?i)(config|env|setting|parameter|option)`),
	}

	// Initialize matcher and recognizer
	engine.matcher = NewBasicPatternMatcher()
	engine.recognizer = NewSequenceRecognizer(engine)

	// Start batch processing goroutine
	if config.LearningEnabled {
		engine.startBatchProcessor()
	}

	return engine
}

// NewPatternEngine creates a new pattern recognition engine with minimal dependencies (for testing)
func NewPatternEngine(storage PatternStorage) *PatternEngine {
	config := DefaultPatternEngineConfig()
	config.LearningEnabled = false // Disable batch processing for tests

	engine := &PatternEngine{
		storage:    storage,
		config:     config,
		batchQueue: make(chan *batchItem, config.BatchSize*2),
		stopBatch:  make(chan struct{}),
		metrics:    &patternMetrics{},
		logger:     logging.NewNoOpLogger(), // No-op logger for tests
	}

	// Initialize pattern regexes
	engine.patternRegexes = map[string]*regexp.Regexp{
		"problem":  regexp.MustCompile(`(?i)(error|issue|problem|bug|fail|broken|not working|doesn't work)`),
		"solution": regexp.MustCompile(`(?i)(fix|solve|resolve|solution|fixed|resolved|working)`),
		"error":    regexp.MustCompile(`(?i)(error:|exception:|fatal:|panic:|warning:)`),
		"command":  regexp.MustCompile(`(?i)(run|execute|install|build|test|deploy)`),
		"code":     regexp.MustCompile("```[\\s\\S]*?```"),
	}

	// Initialize matcher and recognizer
	engine.matcher = NewBasicPatternMatcher()
	engine.recognizer = NewSequenceRecognizer(engine)

	return engine
}

// startBatchProcessor starts the background batch processor
func (pe *PatternEngine) startBatchProcessor() {
	pe.batchWg.Add(1)
	go func() {
		defer pe.batchWg.Done()

		ticker := time.NewTicker(pe.config.BatchInterval)
		defer ticker.Stop()

		batch := make([]*batchItem, 0, pe.config.BatchSize)

		for {
			select {
			case <-pe.stopBatch:
				// Process remaining items
				if len(batch) > 0 {
					pe.processBatch(batch)
				}
				return

			case item := <-pe.batchQueue:
				batch = append(batch, item)
				if len(batch) >= pe.config.BatchSize {
					pe.processBatch(batch)
					batch = batch[:0]
				}

			case <-ticker.C:
				if len(batch) > 0 {
					pe.processBatch(batch)
					batch = batch[:0]
				}
			}
		}
	}()
}

// processBatch processes a batch of pattern learning items
func (pe *PatternEngine) processBatch(batch []*batchItem) {
	ctx := context.Background()

	for _, item := range batch {
		err := pe.learnPatternInternal(ctx, item.chunks, item.outcome)
		if item.callback != nil {
			item.callback(err)
		}
	}
}

// Close cleanly shuts down the pattern engine
func (pe *PatternEngine) Close() error {
	if pe.config.LearningEnabled {
		close(pe.stopBatch)
		pe.batchWg.Wait()
	}
	return nil
}

// RecognizePatterns analyzes chunks and identifies patterns using AI
func (pe *PatternEngine) RecognizePatterns(ctx context.Context, chunks []types.ConversationChunk) ([]Pattern, error) {
	if len(chunks) == 0 {
		return []Pattern{}, nil
	}

	startTime := time.Now()
	defer func() {
		pe.metrics.mu.Lock()
		pe.metrics.processingTime += time.Since(startTime)
		pe.metrics.mu.Unlock()
	}()

	var recognizedPatterns []Pattern

	// Generate embeddings for the chunks
	text := extractText(chunks)
	textEmbeddings, err := pe.getOrGenerateEmbeddings(ctx, text)
	if err != nil {
		pe.logger.Error("Failed to generate embeddings", "error", err)
	}

	// Use AI to identify patterns
	aiPatterns, err := pe.identifyPatternsWithAI(ctx, chunks)
	if err == nil && len(aiPatterns) > 0 {
		recognizedPatterns = append(recognizedPatterns, aiPatterns...)
		pe.metrics.mu.Lock()
		pe.metrics.aiCalls++
		pe.metrics.mu.Unlock()
	}

	// Try sequence recognition
	sequencePatterns, err := pe.recognizer.RecognizeSequence(chunks)
	if err == nil {
		recognizedPatterns = append(recognizedPatterns, sequencePatterns...)
	}

	// Match against stored patterns using embeddings
	if len(textEmbeddings) > 0 {
		similarPatterns, err := pe.findSimilarPatterns(ctx, textEmbeddings, 10)
		if err == nil {
			for i := range similarPatterns {
				similarPatterns[i].ConfidenceScore = pe.calculateSimilarityScore(textEmbeddings, similarPatterns[i].Embeddings)
				if similarPatterns[i].ConfidenceScore >= pe.config.MinConfidence {
					recognizedPatterns = append(recognizedPatterns, similarPatterns[i])
				}
			}
		}
	}

	// Deduplicate and sort patterns
	recognizedPatterns = pe.deduplicatePatterns(recognizedPatterns)

	// Sort by confidence
	sort.Slice(recognizedPatterns, func(i, j int) bool {
		return recognizedPatterns[i].ConfidenceScore > recognizedPatterns[j].ConfidenceScore
	})

	// Limit results
	if len(recognizedPatterns) > 10 {
		recognizedPatterns = recognizedPatterns[:10]
	}

	pe.metrics.mu.Lock()
	pe.metrics.patternsDetected += int64(len(recognizedPatterns))
	pe.metrics.mu.Unlock()

	return recognizedPatterns, nil
}

// identifyPatternsWithAI uses AI to identify patterns in chunks
func (pe *PatternEngine) identifyPatternsWithAI(ctx context.Context, chunks []types.ConversationChunk) ([]Pattern, error) {
	// Return empty patterns if AI service is not available (testing scenario)
	if pe.aiService == nil {
		return []Pattern{}, nil
	}

	// Prepare context for AI
	prompt := pe.buildPatternIdentificationPrompt(chunks)

	request := &internalAI.Request{
		ID: uuid.New().String(),
		Messages: []internalAI.Message{
			{
				Role:    "system",
				Content: "You are an expert pattern recognition system. Analyze the conversation and identify recurring patterns, workflows, and architectural decisions.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: &internalAI.RequestMetadata{
			Repository: "pattern-engine",
			Tags:       []string{"pattern-detection"},
			CreatedAt:  time.Now(),
		},
	}

	response, err := pe.aiService.ProcessRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("AI pattern identification failed: %w", err)
	}

	// Parse AI response into patterns
	return pe.parseAIPatternResponse(response.Content)
}

// buildPatternIdentificationPrompt builds a prompt for AI pattern identification
func (pe *PatternEngine) buildPatternIdentificationPrompt(chunks []types.ConversationChunk) string {
	var sb strings.Builder

	sb.WriteString("Analyze the following conversation and identify patterns:\n\n")

	for i := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, chunks[i].Type, chunks[i].Content))
		if i >= 10 { // Limit context to avoid token limits
			sb.WriteString("...(truncated)...\n")
			break
		}
	}

	sb.WriteString("\nIdentify patterns in the following categories:\n")
	sb.WriteString("1. Code patterns (design patterns, idioms, anti-patterns)\n")
	sb.WriteString("2. Workflow patterns (development processes, task sequences)\n")
	sb.WriteString("3. Architectural patterns (system design, component relationships)\n")
	sb.WriteString("4. Behavioral patterns (user interactions, decision making)\n")
	sb.WriteString("5. Error patterns (common mistakes, debugging sequences)\n")
	sb.WriteString("\nReturn patterns in JSON format with: name, type, description, confidence, keywords.")

	return sb.String()
}

// parseAIPatternResponse parses AI response into Pattern structs
func (pe *PatternEngine) parseAIPatternResponse(content string) ([]Pattern, error) {
	// Extract JSON from AI response
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		// Try object format
		jsonStart = strings.Index(content, "{")
		jsonEnd = strings.LastIndex(content, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			return nil, errors.New("no JSON found in AI response")
		}
		content = "[" + content[jsonStart:jsonEnd+1] + "]"
	} else {
		content = content[jsonStart : jsonEnd+1]
	}

	var aiPatterns []struct {
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		Description string   `json:"description"`
		Confidence  float64  `json:"confidence"`
		Keywords    []string `json:"keywords"`
	}

	if err := json.Unmarshal([]byte(content), &aiPatterns); err != nil {
		return nil, fmt.Errorf("failed to parse AI patterns: %w", err)
	}

	patterns := make([]Pattern, 0, len(aiPatterns))
	for _, ap := range aiPatterns {
		pattern := Pattern{
			ID:               uuid.New().String(),
			Name:             ap.Name,
			Type:             pe.mapAIPatternType(ap.Type),
			Description:      ap.Description,
			ConfidenceScore:  ap.Confidence,
			Keywords:         ap.Keywords,
			ValidationStatus: ValidationUnvalidated,
			Version:          1,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// mapAIPatternType maps AI pattern type to PatternType
func (pe *PatternEngine) mapAIPatternType(aiType string) PatternType {
	typeMap := map[string]PatternType{
		"code":          PatternTypeCode,
		"workflow":      PatternTypeWorkflow,
		"architectural": PatternTypeArchitectural,
		"behavioral":    PatternTypeBehavioral,
		"error":         PatternTypeError,
		"optimization":  PatternTypeOptimization,
		"refactoring":   PatternTypeRefactoring,
	}

	aiTypeLower := strings.ToLower(aiType)
	for key, patternType := range typeMap {
		if strings.Contains(aiTypeLower, key) {
			return patternType
		}
	}

	return PatternTypeWorkflow // default
}

// LearnPattern creates or updates a pattern based on conversation examples
func (pe *PatternEngine) LearnPattern(ctx context.Context, chunks []types.ConversationChunk, outcome PatternOutcome) error {
	if !pe.config.LearningEnabled || len(chunks) < 2 {
		return nil
	}

	// Queue for batch processing
	if pe.config.BatchSize > 1 {
		callback := make(chan error, 1)
		pe.batchQueue <- &batchItem{
			chunks:   chunks,
			outcome:  outcome,
			callback: func(err error) { callback <- err },
		}

		select {
		case err := <-callback:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Second):
			return errors.New("pattern learning timeout")
		}
	}

	// Process immediately
	return pe.learnPatternInternal(ctx, chunks, outcome)
}

// learnPatternInternal performs the actual pattern learning
func (pe *PatternEngine) learnPatternInternal(ctx context.Context, chunks []types.ConversationChunk, outcome PatternOutcome) error {
	// Use AI to analyze and learn patterns
	pattern, err := pe.analyzeWithAI(ctx, chunks, outcome)
	if err != nil {
		return fmt.Errorf("AI pattern analysis failed: %w", err)
	}

	// Generate embeddings for the pattern
	text := pattern.Name + " " + pattern.Description + " " + strings.Join(pattern.Keywords, " ")
	patternEmbeddings, err := pe.getOrGenerateEmbeddings(ctx, text)
	if err == nil {
		pattern.Embeddings = patternEmbeddings
	}

	// Check for similar existing patterns
	existingPattern, exists, err := pe.findExistingPattern(ctx, pattern)
	if err != nil {
		pe.logger.Error("Failed to find existing pattern", "error", err)
	}

	var storeErr error
	if exists {
		storeErr = pe.updateExistingPattern(ctx, existingPattern, pattern, chunks, outcome)
	} else {
		storeErr = pe.storeNewPattern(ctx, pattern, chunks, outcome)
	}

	if storeErr != nil {
		return storeErr
	}

	pe.metrics.mu.Lock()
	pe.metrics.patternsLearned++
	pe.metrics.mu.Unlock()

	return nil
}

// updateExistingPattern updates an existing pattern with new data
func (pe *PatternEngine) updateExistingPattern(ctx context.Context, existingPattern, newPattern *Pattern, chunks []types.ConversationChunk, outcome PatternOutcome) error {
	// Update existing pattern
	existingPattern.OccurrenceCount++
	pe.updatePatternFeedback(existingPattern, outcome)

	// Update confidence using Bayesian approach
	existingPattern.ConfidenceScore = pe.calculateBayesianConfidence(
		existingPattern.PositiveFeedback,
		existingPattern.NegativeFeedback,
	)

	pe.updatePatternTimestamps(existingPattern)

	// Check if pattern needs evolution
	if err := pe.handlePatternEvolution(ctx, existingPattern, newPattern); err != nil {
		return err
	}

	if err := pe.storage.UpdatePattern(ctx, existingPattern); err != nil {
		return fmt.Errorf("failed to update pattern: %w", err)
	}

	// Store occurrence
	return pe.storePatternOccurrence(ctx, newPattern, chunks)
}

// storeNewPattern stores a new pattern
func (pe *PatternEngine) storeNewPattern(ctx context.Context, pattern *Pattern, chunks []types.ConversationChunk, outcome PatternOutcome) error {
	// Store new pattern
	pattern.OccurrenceCount = 1
	pe.updatePatternFeedback(pattern, outcome)
	pattern.ConfidenceScore = pe.calculateInitialConfidence(outcome)

	if err := pe.storage.StorePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to store pattern: %w", err)
	}

	// Store initial occurrence
	return pe.storePatternOccurrence(ctx, pattern, chunks)
}

// updatePatternFeedback updates pattern feedback based on outcome
func (pe *PatternEngine) updatePatternFeedback(pattern *Pattern, outcome PatternOutcome) {
	switch outcome {
	case OutcomeSuccess:
		pattern.PositiveFeedback++
	case OutcomeFailure:
		pattern.NegativeFeedback++
	}
}

// updatePatternTimestamps updates pattern timestamps
func (pe *PatternEngine) updatePatternTimestamps(pattern *Pattern) {
	now := time.Now()
	pattern.LastSeenAt = &now
	pattern.UpdatedAt = now
}

// handlePatternEvolution checks and handles pattern evolution
func (pe *PatternEngine) handlePatternEvolution(ctx context.Context, existingPattern, newPattern *Pattern) error {
	if !pe.shouldEvolvePattern(existingPattern, newPattern) {
		return nil
	}

	evolvedPattern := pe.evolvePattern(existingPattern, newPattern)
	if err := pe.storage.StorePattern(ctx, evolvedPattern); err != nil {
		return fmt.Errorf("failed to store evolved pattern: %w", err)
	}

	// Update parent pattern
	existingPattern.ValidationStatus = ValidationEvolved
	return nil
}

// storePatternOccurrence stores a pattern occurrence
func (pe *PatternEngine) storePatternOccurrence(ctx context.Context, pattern *Pattern, chunks []types.ConversationChunk) error {
	occurrence := pe.createOccurrence(pattern, chunks)
	if err := pe.storage.StoreOccurrence(ctx, occurrence); err != nil {
		pe.logger.Error("Failed to store occurrence", "error", err)
		return err
	}
	return nil
}

// analyzeWithAI uses AI to analyze chunks and create a pattern
func (pe *PatternEngine) analyzeWithAI(ctx context.Context, chunks []types.ConversationChunk, outcome PatternOutcome) (*Pattern, error) {
	// Return a basic pattern if AI service is not available (testing scenario)
	if pe.aiService == nil {
		return pe.createBasicPattern(chunks, outcome), nil
	}

	prompt := pe.buildLearningPrompt(chunks, outcome)

	request := &internalAI.Request{
		ID: uuid.New().String(),
		Messages: []internalAI.Message{
			{
				Role:    "system",
				Content: "You are an expert pattern learning system. Analyze the conversation and create a detailed pattern that can be reused.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: &internalAI.RequestMetadata{
			Repository: "pattern-engine",
			Tags:       []string{"pattern-learning"},
			CreatedAt:  time.Now(),
		},
	}

	response, err := pe.aiService.ProcessRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	pe.metrics.mu.Lock()
	pe.metrics.aiCalls++
	pe.metrics.mu.Unlock()

	return pe.parseAILearningResponse(response.Content, chunks)
}

// buildLearningPrompt builds a prompt for AI pattern learning
func (pe *PatternEngine) buildLearningPrompt(chunks []types.ConversationChunk, outcome PatternOutcome) string {
	var sb strings.Builder

	sb.WriteString("Learn a pattern from the following conversation:\n\n")

	for i := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, chunks[i].Type, chunks[i].Content))
	}

	sb.WriteString(fmt.Sprintf("\nOutcome: %s\n", outcome))
	sb.WriteString("\nCreate a detailed pattern with:\n")
	sb.WriteString("- name: descriptive pattern name\n")
	sb.WriteString("- type: code|workflow|architectural|behavioral|error|optimization|refactoring\n")
	sb.WriteString("- description: detailed description of the pattern\n")
	sb.WriteString("- category: specific subcategory within the type\n")
	sb.WriteString("- keywords: relevant keywords for searching\n")
	sb.WriteString("- signature: pattern matching rules (conditions, triggers, etc.)\n")
	sb.WriteString("- file_patterns: file types where this pattern applies\n")
	sb.WriteString("- language: programming language (if applicable)\n")
	sb.WriteString("\nReturn as JSON object.")

	return sb.String()
}

// parseAILearningResponse parses AI response for pattern learning
func (pe *PatternEngine) parseAILearningResponse(content string, chunks []types.ConversationChunk) (*Pattern, error) {
	// Extract JSON from response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, errors.New("no JSON found in AI response")
	}

	content = content[jsonStart : jsonEnd+1]

	var aiPattern struct {
		Name         string                 `json:"name"`
		Type         string                 `json:"type"`
		Description  string                 `json:"description"`
		Category     string                 `json:"category"`
		Keywords     []string               `json:"keywords"`
		Signature    map[string]interface{} `json:"signature"`
		FilePatterns []string               `json:"file_patterns"`
		Language     string                 `json:"language"`
	}

	if err := json.Unmarshal([]byte(content), &aiPattern); err != nil {
		return nil, fmt.Errorf("failed to parse AI pattern: %w", err)
	}

	// Extract repository from chunks metadata
	repository := ""
	if len(chunks) > 0 && chunks[0].Metadata.Repository != "" {
		repository = chunks[0].Metadata.Repository
	}

	pattern := &Pattern{
		ID:               uuid.New().String(),
		Name:             aiPattern.Name,
		Type:             pe.mapAIPatternType(aiPattern.Type),
		Description:      aiPattern.Description,
		Category:         aiPattern.Category,
		Keywords:         aiPattern.Keywords,
		Signature:        aiPattern.Signature,
		FilePatterns:     aiPattern.FilePatterns,
		Language:         aiPattern.Language,
		RepositoryURL:    repository,
		ValidationStatus: ValidationUnvalidated,
		Version:          1,
		Metadata:         make(map[string]interface{}),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return pattern, nil
}

// GetPatternSuggestions returns patterns that might be relevant to current context
func (pe *PatternEngine) GetPatternSuggestions(ctx context.Context, currentChunks []types.ConversationChunk, limit int) ([]Pattern, error) {
	if len(currentChunks) == 0 {
		return []Pattern{}, nil
	}

	// Generate embeddings for current context
	text := extractText(currentChunks)
	embeddingVec, err := pe.getOrGenerateEmbeddings(ctx, text)
	if err != nil {
		pe.logger.Error("Failed to generate embeddings for suggestions", "error", err)
		// Continue without embeddings
	}

	var suggestions []Pattern

	// Use AI to get pattern suggestions
	aiSuggestions, err := pe.getAISuggestions(ctx, currentChunks)
	if err == nil {
		suggestions = append(suggestions, aiSuggestions...)
	}

	// Find similar patterns using embeddings
	if len(embeddingVec) > 0 {
		similarPatterns, err := pe.findSimilarPatterns(ctx, embeddingVec, limit*2)
		if err == nil {
			suggestions = append(suggestions, similarPatterns...)
		}
	}

	// Deduplicate and score
	suggestions = pe.deduplicatePatterns(suggestions)

	// Calculate relevance scores
	for i := range suggestions {
		suggestions[i].ConfidenceScore = pe.calculateRelevance(currentChunks, &suggestions[i])
	}

	// Sort by relevance
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].ConfidenceScore > suggestions[j].ConfidenceScore
	})

	// Return top patterns
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

// Additional helper methods needed for the implementation

// getOrGenerateEmbeddings gets embeddings from cache or generates new ones
func (pe *PatternEngine) getOrGenerateEmbeddings(ctx context.Context, text string) ([]float64, error) {
	// Return empty embeddings if service is not available (testing scenario)
	if pe.embeddingService == nil {
		return []float64{}, nil
	}

	// Check cache
	if pe.config.EnableCaching {
		if cached, found := pe.embeddingCache.Load(text); found {
			pe.metrics.mu.Lock()
			pe.metrics.cacheHits++
			pe.metrics.mu.Unlock()
			return cached.([]float64), nil
		}
	}

	pe.metrics.mu.Lock()
	pe.metrics.cacheMisses++
	pe.metrics.mu.Unlock()

	// Generate embeddings
	embeddingVec, err := pe.embeddingService.Generate(ctx, text)
	if err != nil {
		return nil, err
	}

	// Cache embeddings
	if pe.config.EnableCaching {
		pe.embeddingCache.Store(text, embeddingVec)
	}

	return embeddingVec, nil
}

// findSimilarPatterns finds patterns similar to given embeddings
func (pe *PatternEngine) findSimilarPatterns(ctx context.Context, embeddingVec []float64, limit int) ([]Pattern, error) {
	// This would typically use a vector similarity search
	// For now, we'll use a simplified approach
	allPatterns, err := pe.storage.ListPatterns(ctx, nil)
	if err != nil {
		return nil, err
	}

	type scoredPattern struct {
		pattern Pattern
		score   float64
	}

	var scored []scoredPattern
	for i := range allPatterns {
		if len(allPatterns[i].Embeddings) > 0 {
			score := pe.calculateSimilarityScore(embeddingVec, allPatterns[i].Embeddings)
			scored = append(scored, scoredPattern{allPatterns[i], score})
		}
	}

	// Sort by similarity
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top patterns
	result := make([]Pattern, 0, limit)
	for i := range scored {
		if i >= limit {
			break
		}
		result = append(result, scored[i].pattern)
	}

	return result, nil
}

// calculateSimilarityScore calculates cosine similarity between embeddings
func (pe *PatternEngine) calculateSimilarityScore(embeddingVec1, embeddingVec2 []float64) float64 {
	if len(embeddingVec1) != len(embeddingVec2) || len(embeddingVec1) == 0 {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := range embeddingVec1 {
		dotProduct += embeddingVec1[i] * embeddingVec2[i]
		norm1 += embeddingVec1[i] * embeddingVec1[i]
		norm2 += embeddingVec2[i] * embeddingVec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// deduplicatePatterns removes duplicate patterns
func (pe *PatternEngine) deduplicatePatterns(patterns []Pattern) []Pattern {
	seen := make(map[string]bool)
	result := make([]Pattern, 0, len(patterns))

	for i := range patterns {
		if !seen[patterns[i].ID] {
			seen[patterns[i].ID] = true
			result = append(result, patterns[i])
		}
	}

	return result
}

// findExistingPattern finds an existing pattern similar to the given one
func (pe *PatternEngine) findExistingPattern(ctx context.Context, pattern *Pattern) (*Pattern, bool, error) {
	// Search by name and keywords
	candidates, err := pe.storage.SearchPatterns(ctx, pattern.Name, 10)
	if err != nil {
		return nil, false, err
	}

	// Check embeddings similarity
	for i := range candidates {
		candidate := &candidates[i]
		if candidate.Type != pattern.Type {
			continue
		}

		if pe.checkPatternSimilarity(pattern, candidate) {
			return candidate, true, nil
		}
	}

	return nil, false, nil
}

// checkPatternSimilarity checks if two patterns are similar
func (pe *PatternEngine) checkPatternSimilarity(pattern, candidate *Pattern) bool {
	// Check embedding similarity if available
	if len(pattern.Embeddings) > 0 && len(candidate.Embeddings) > 0 {
		similarity := pe.calculateSimilarityScore(pattern.Embeddings, candidate.Embeddings)
		return similarity > pe.config.EvolutionThreshold
	}

	// Fall back to keyword matching
	keywordOverlap := calculateOverlap(pattern.Keywords, candidate.Keywords)
	return keywordOverlap > 0.6
}

// calculateBayesianConfidence calculates confidence using Bayesian approach
func (pe *PatternEngine) calculateBayesianConfidence(positive, negative int) float64 {
	// Using Laplace smoothing
	return float64(positive+1) / float64(positive+negative+2)
}

// calculateInitialConfidence calculates initial confidence based on outcome
func (pe *PatternEngine) calculateInitialConfidence(outcome PatternOutcome) float64 {
	switch outcome {
	case OutcomeSuccess:
		return 0.8
	case OutcomePartial:
		return 0.6
	case OutcomeFailure:
		return 0.3
	case OutcomeInterrupted:
		return 0.4
	case OutcomeUnknown:
		return 0.5
	default:
		return 0.5
	}
}

// shouldEvolvePattern determines if a pattern should evolve
func (pe *PatternEngine) shouldEvolvePattern(existing, evolved *Pattern) bool {
	// Check if patterns are sufficiently different
	if len(existing.Embeddings) > 0 && len(evolved.Embeddings) > 0 {
		similarity := pe.calculateSimilarityScore(existing.Embeddings, evolved.Embeddings)
		return similarity < pe.config.EvolutionThreshold && similarity > 0.4
	}

	// Check keyword divergence
	keywordOverlap := calculateOverlap(existing.Keywords, evolved.Keywords)
	return keywordOverlap < 0.7 && keywordOverlap > 0.3
}

// evolvePattern creates an evolved version of a pattern
func (pe *PatternEngine) evolvePattern(parent, newPattern *Pattern) *Pattern {
	evolved := &Pattern{
		ID:               uuid.New().String(),
		Type:             parent.Type,
		Name:             newPattern.Name + " (Evolved)",
		Description:      newPattern.Description,
		Category:         newPattern.Category,
		Signature:        newPattern.Signature,
		Keywords:         append(parent.Keywords, newPattern.Keywords...),
		RepositoryURL:    newPattern.RepositoryURL,
		FilePatterns:     append(parent.FilePatterns, newPattern.FilePatterns...),
		Language:         newPattern.Language,
		ConfidenceScore:  parent.ConfidenceScore * 0.8, // Slightly lower confidence for evolved patterns
		ValidationStatus: ValidationUnvalidated,
		ParentPatternID:  &parent.ID,
		EvolutionReason:  "Pattern evolved due to significant variations",
		Version:          parent.Version + 1,
		Metadata:         newPattern.Metadata,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Embeddings:       newPattern.Embeddings,
	}

	// Deduplicate keywords and file patterns
	evolved.Keywords = unique(evolved.Keywords)
	evolved.FilePatterns = unique(evolved.FilePatterns)

	return evolved
}

// createOccurrence creates a pattern occurrence record
func (pe *PatternEngine) createOccurrence(pattern *Pattern, chunks []types.ConversationChunk) *PatternOccurrence {
	occurrence := &PatternOccurrence{
		ID:              uuid.New().String(),
		PatternID:       pattern.ID,
		RepositoryURL:   pattern.RepositoryURL,
		DetectionScore:  pattern.ConfidenceScore,
		DetectionMethod: "ai-assisted",
		Metadata:        make(map[string]interface{}),
		DetectedAt:      time.Now(),
	}

	// Extract code snippet if available
	for i := range chunks {
		if matches := pe.patternRegexes["code"].FindStringSubmatch(chunks[i].Content); len(matches) > 0 {
			occurrence.CodeSnippet = matches[0]
			break
		}
	}

	// Set session and chunk IDs
	if len(chunks) > 0 {
		occurrence.SessionID = chunks[0].SessionID
		occurrence.ChunkID = chunks[0].ID
	}

	return occurrence
}

// getAISuggestions gets pattern suggestions from AI
func (pe *PatternEngine) getAISuggestions(ctx context.Context, chunks []types.ConversationChunk) ([]Pattern, error) {
	// Return empty suggestions if AI service is not available
	if pe.aiService == nil {
		return []Pattern{}, nil
	}

	prompt := pe.buildSuggestionPrompt(chunks)

	request := &internalAI.Request{
		ID: uuid.New().String(),
		Messages: []internalAI.Message{
			{
				Role:    "system",
				Content: "You are a pattern suggestion system. Based on the current context, suggest relevant patterns that might help.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: &internalAI.RequestMetadata{
			Repository: "pattern-engine",
			Tags:       []string{"pattern-suggestions"},
			CreatedAt:  time.Now(),
		},
	}

	response, err := pe.aiService.ProcessRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	return pe.parseAIPatternResponse(response.Content)
}

// buildSuggestionPrompt builds a prompt for pattern suggestions
func (pe *PatternEngine) buildSuggestionPrompt(chunks []types.ConversationChunk) string {
	var sb strings.Builder

	sb.WriteString("Based on the current conversation context, suggest relevant patterns:\n\n")

	for i := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, chunks[i].Type, chunks[i].Content))
		if i >= 5 { // Limit context
			sb.WriteString("...(more context)...\n")
			break
		}
	}

	sb.WriteString("\nSuggest patterns that might be applicable or helpful.")
	sb.WriteString("\nReturn as JSON array with: name, type, description, confidence, keywords.")

	return sb.String()
}

// calculateRelevance calculates relevance of a pattern to current chunks
func (pe *PatternEngine) calculateRelevance(currentChunks []types.ConversationChunk, pattern *Pattern) float64 {
	// Extract features from current chunks
	text := extractText(currentChunks)
	currentKeywords := pe.extractKeywords(currentChunks)

	// Keyword overlap
	keywordScore := calculateOverlap(currentKeywords, pattern.Keywords)

	// Pattern type relevance
	typeScore := pe.calculateTypeRelevance(text, pattern.Type)

	// Repository match
	repoScore := 0.0
	if len(currentChunks) > 0 && currentChunks[0].Metadata.Repository == pattern.RepositoryURL {
		repoScore = 0.2
	}

	// Language match
	langScore := 0.0
	if pattern.Language != "" && strings.Contains(text, pattern.Language) {
		langScore = 0.1
	}

	// Combine scores
	relevance := keywordScore*0.4 + typeScore*0.3 + repoScore + langScore + pattern.ConfidenceScore*0.2

	return math.Min(relevance, 1.0)
}

// calculateTypeRelevance calculates relevance based on pattern type
func (pe *PatternEngine) calculateTypeRelevance(text string, patternType PatternType) float64 {
	textLower := strings.ToLower(text)

	switch patternType {
	case PatternTypeCode:
		if pe.patternRegexes["code"].MatchString(text) || pe.patternRegexes["function"].MatchString(text) {
			return 0.8
		}
	case PatternTypeError:
		if pe.patternRegexes["error"].MatchString(textLower) {
			return 0.9
		}
	case PatternTypeWorkflow:
		if pe.patternRegexes["command"].MatchString(textLower) {
			return 0.7
		}
	case PatternTypeArchitectural:
		if strings.Contains(textLower, "design") || strings.Contains(textLower, "architecture") {
			return 0.8
		}
	case PatternTypeOptimization:
		if strings.Contains(textLower, "performance") || strings.Contains(textLower, "optimize") {
			return 0.7
		}
	}

	return 0.3 // base relevance
}

// extractKeywords extracts keywords from chunks
func (pe *PatternEngine) extractKeywords(chunks []types.ConversationChunk) []string {
	text := extractText(chunks)
	words := strings.Fields(strings.ToLower(text))

	keywordCount := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 && !isStopWord(word) {
			keywordCount[word]++
		}
	}

	type wordCount struct {
		word  string
		count int
	}

	wordCounts := make([]wordCount, 0, len(keywordCount))
	for word, count := range keywordCount {
		wordCounts = append(wordCounts, wordCount{word, count})
	}

	sort.Slice(wordCounts, func(i, j int) bool {
		return wordCounts[i].count > wordCounts[j].count
	})

	// Pre-allocate for max 10 keywords
	maxKeywords := 10
	if len(wordCounts) < maxKeywords {
		maxKeywords = len(wordCounts)
	}
	keywords := make([]string, 0, maxKeywords)
	for i, wc := range wordCounts {
		if i >= 10 {
			break
		}
		keywords = append(keywords, wc.word)
	}

	return keywords
}

// GetMetrics returns pattern engine metrics
func (pe *PatternEngine) GetMetrics() map[string]interface{} {
	pe.metrics.mu.RLock()
	defer pe.metrics.mu.RUnlock()

	return map[string]interface{}{
		"patterns_detected":  pe.metrics.patternsDetected,
		"patterns_learned":   pe.metrics.patternsLearned,
		"ai_calls":           pe.metrics.aiCalls,
		"cache_hits":         pe.metrics.cacheHits,
		"cache_misses":       pe.metrics.cacheMisses,
		"processing_time_ms": pe.metrics.processingTime.Milliseconds(),
		"cache_hit_rate":     float64(pe.metrics.cacheHits) / float64(pe.metrics.cacheHits+pe.metrics.cacheMisses+1),
	}
}

// ValidatePattern validates a pattern with given feedback
func (pe *PatternEngine) ValidatePattern(ctx context.Context, patternID string, isValid bool) error {
	// Update pattern confidence based on validation
	return pe.storage.UpdateConfidence(ctx, patternID, isValid)
}

// GetPatternHierarchy gets the hierarchy of related patterns
func (pe *PatternEngine) GetPatternHierarchy(ctx context.Context, patternID string) ([]PatternRelationship, error) {
	return pe.storage.GetRelationships(ctx, patternID)
}

// ExportPatterns exports patterns in a specific format
func (pe *PatternEngine) ExportPatterns(ctx context.Context, patternType *PatternType) ([]byte, error) {
	patterns, err := pe.storage.ListPatterns(ctx, patternType)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(patterns, "", "  ")
}

// ImportPatterns imports patterns from exported data
func (pe *PatternEngine) ImportPatterns(ctx context.Context, data []byte) error {
	var patterns []Pattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		return fmt.Errorf("failed to unmarshal patterns: %w", err)
	}

	for i := range patterns {
		// Generate new ID to avoid conflicts
		patterns[i].ID = uuid.New().String()
		patterns[i].CreatedAt = time.Now()
		patterns[i].UpdatedAt = time.Now()

		if err := pe.storage.StorePattern(ctx, &patterns[i]); err != nil {
			pe.logger.Error("Failed to import pattern", "name", patterns[i].Name, "error", err)
		}
	}

	return nil
}

// Utility functions

func extractText(chunks []types.ConversationChunk) string {
	texts := make([]string, 0, len(chunks))
	for i := range chunks {
		chunk := &chunks[i]
		texts = append(texts, chunk.Content)
	}
	return strings.Join(texts, " ")
}

func extractChunkIDs(chunks []types.ConversationChunk) []string {
	ids := make([]string, 0, len(chunks))
	for i := range chunks {
		chunk := &chunks[i]
		ids = append(ids, chunk.ID)
	}
	return ids
}

func generatePatternID() string {
	return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "how": true,
		"man": true, "new": true, "now": true, "old": true, "see": true,
		"two": true, "way": true, "who": true, "boy": true, "did": true,
		"its": true, "let": true, "put": true, "say": true, "she": true,
		"too": true, "use": true,
	}
	return stopWords[word]
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func calculateOverlap(slice1, slice2 []string) float64 {
	if len(slice1) == 0 || len(slice2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, item := range slice1 {
		set1[item] = true
	}

	overlap := 0
	for _, item := range slice2 {
		if set1[item] {
			overlap++
		}
	}

	return float64(overlap) / math.Max(float64(len(slice1)), float64(len(slice2)))
}

// createBasicPattern creates a basic pattern from chunks when AI service is not available
func (pe *PatternEngine) createBasicPattern(chunks []types.ConversationChunk, outcome PatternOutcome) *Pattern {
	text := extractText(chunks)
	keywords := pe.extractKeywords(chunks)

	// Determine pattern type based on content
	var patternType PatternType
	switch {
	case pe.patternRegexes["error"].MatchString(strings.ToLower(text)):
		patternType = PatternTypeError
	case pe.patternRegexes["code"].MatchString(text):
		patternType = PatternTypeCode
	case pe.patternRegexes["command"].MatchString(strings.ToLower(text)):
		patternType = PatternTypeWorkflow
	default:
		patternType = PatternTypeBehavioral
	}

	// Extract repository from chunks metadata
	repository := ""
	if len(chunks) > 0 && chunks[0].Metadata.Repository != "" {
		repository = chunks[0].Metadata.Repository
	}

	return &Pattern{
		ID:               uuid.New().String(),
		Name:             fmt.Sprintf("Basic Pattern - %s", patternType),
		Type:             patternType,
		Description:      fmt.Sprintf("Auto-generated pattern from conversation (outcome: %s)", outcome),
		Category:         "basic",
		Keywords:         keywords,
		RepositoryURL:    repository,
		ConfidenceScore:  pe.calculateInitialConfidence(outcome),
		ValidationStatus: ValidationUnvalidated,
		Version:          1,
		Metadata:         make(map[string]interface{}),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}
