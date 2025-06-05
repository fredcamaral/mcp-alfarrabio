package bulk

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lerian-mcp-memory/pkg/types"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ImportFormat represents the format of the import data
type ImportFormat string

const (
	// FormatJSON imports data in JSON format
	FormatJSON ImportFormat = "json"
	// FormatMarkdown imports data in Markdown format
	FormatMarkdown ImportFormat = "markdown"
	// FormatCSV imports data in CSV format
	FormatCSV ImportFormat = "csv"
	// FormatArchive imports data from archive format
	FormatArchive ImportFormat = "archive"
	// FormatAuto automatically detects the format
	FormatAuto ImportFormat = "auto" // Auto-detect format
)

// ImportOptions configures import behavior
type ImportOptions struct {
	Format           ImportFormat     `json:"format"`
	Repository       string           `json:"repository,omitempty"`
	DefaultSessionID string           `json:"default_session_id,omitempty"`
	DefaultTags      []string         `json:"default_tags,omitempty"`
	ChunkingStrategy ChunkingStrategy `json:"chunking_strategy"`
	ConflictPolicy   ConflictPolicy   `json:"conflict_policy"`
	ValidateChunks   bool             `json:"validate_chunks"`
	Metadata         ImportMetadata   `json:"metadata,omitempty"`
}

// ChunkingStrategy defines how to chunk imported data
type ChunkingStrategy string

const (
	// ChunkingAuto automatically determines the best chunking strategy
	ChunkingAuto ChunkingStrategy = "auto"
	// ChunkingParagraph chunks data by paragraphs
	ChunkingParagraph ChunkingStrategy = "paragraph"
	// ChunkingFixedSize chunks data into fixed-size pieces
	ChunkingFixedSize ChunkingStrategy = "fixed_size"
	// ChunkingConversationTurns chunks data by conversation turns
	ChunkingConversationTurns ChunkingStrategy = "conversation_turns"
)

// ImportMetadata contains metadata about the import
type ImportMetadata struct {
	SourceSystem string                 `json:"source_system,omitempty"`
	ImportDate   string                 `json:"import_date,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Custom       map[string]interface{} `json:"custom,omitempty"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	TotalItems      int                       `json:"total_items"`
	ProcessedItems  int                       `json:"processed_items"`
	SuccessfulItems int                       `json:"successful_items"`
	FailedItems     int                       `json:"failed_items"`
	SkippedItems    int                       `json:"skipped_items"`
	Chunks          []types.ConversationChunk `json:"chunks"`
	Errors          []ImportError             `json:"errors,omitempty"`
	Warnings        []ImportWarning           `json:"warnings,omitempty"`
	Summary         string                    `json:"summary"`
}

// ImportError represents an error during import
type ImportError struct {
	Line    int    `json:"line,omitempty"`
	Item    int    `json:"item,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// ImportWarning represents a warning during import
type ImportWarning struct {
	Line    int    `json:"line,omitempty"`
	Item    int    `json:"item,omitempty"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Importer handles importing memories from various formats
type Importer struct {
	logger *log.Logger
}

// NewImporter creates a new bulk importer
func NewImporter(logger *log.Logger) *Importer {
	if logger == nil {
		logger = log.New(log.Writer(), "[BulkImporter] ", log.LstdFlags)
	}

	return &Importer{
		logger: logger,
	}
}

// Import imports data from the provided source
func (imp *Importer) Import(ctx context.Context, data string, options *ImportOptions) (*ImportResult, error) {
	result := &ImportResult{
		Chunks: make([]types.ConversationChunk, 0),
	}

	// Auto-detect format if needed
	if options.Format == FormatAuto {
		options.Format = imp.detectFormat(data)
	}

	// Parse based on format
	switch options.Format {
	case FormatJSON:
		return imp.importJSON(ctx, data, options, result)
	case FormatMarkdown:
		return imp.importMarkdown(ctx, data, options, result)
	case FormatCSV:
		return imp.importCSV(ctx, data, options, result)
	case FormatArchive:
		return imp.importArchive(ctx, data, options, result)
	case FormatAuto:
		// Auto-detect format and retry
		options.Format = imp.detectFormat(data)
		return imp.Import(ctx, data, options)
	default:
		return nil, fmt.Errorf("unsupported format: %s", options.Format)
	}
}

// detectFormat auto-detects the format of the input data
func (imp *Importer) detectFormat(data string) ImportFormat {
	data = strings.TrimSpace(data)

	// Check for JSON
	if strings.HasPrefix(data, "{") || strings.HasPrefix(data, "[") {
		return FormatJSON
	}

	// Check for CSV (look for comma-separated values in first line)
	firstLine := strings.Split(data, "\n")[0]
	if strings.Count(firstLine, ",") >= 2 && !strings.HasPrefix(data, "#") {
		return FormatCSV
	}

	// Check for archive (base64 encoded)
	if imp.isBase64(data) {
		return FormatArchive
	}

	// Default to markdown
	return FormatMarkdown
}

// importJSON imports data from JSON format
func (imp *Importer) importJSON(_ context.Context, data string, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	// Try to parse as array of chunks first
	var chunks []types.ConversationChunk
	if err := json.Unmarshal([]byte(data), &chunks); err == nil {
		return imp.processChunks(chunks, options, result)
	}

	// Try to parse as single chunk
	var chunk types.ConversationChunk
	if err := json.Unmarshal([]byte(data), &chunk); err == nil {
		chunks = []types.ConversationChunk{chunk}
		return imp.processChunks(chunks, options, result)
	}

	// Try to parse as generic conversation data
	var conversationData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &conversationData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert generic data to chunks
	chunks, err := imp.convertConversationData(conversationData, options)
	if err != nil {
		return nil, fmt.Errorf("failed to convert conversation data: %w", err)
	}

	return imp.processChunks(chunks, options, result)
}

// importMarkdown imports data from markdown format
func (imp *Importer) importMarkdown(_ context.Context, data string, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	chunks, err := imp.parseMarkdown(data, options)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	return imp.processChunks(chunks, options, result)
}

// importCSV imports data from CSV format
func (imp *Importer) importCSV(_ context.Context, data string, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	reader := csv.NewReader(strings.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return result, nil
	}

	// Parse header
	header := records[0]
	chunks := make([]types.ConversationChunk, 0, len(records)-1)

	for i, record := range records[1:] {
		chunk, err := imp.parseCSVRecord(header, record, options, i+1)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Line:    i + 2, // +2 because we skip header and 0-indexed
				Message: err.Error(),
				Data:    strings.Join(record, ","),
			})
			result.FailedItems++
			continue
		}

		chunks = append(chunks, *chunk)
	}

	result.TotalItems = len(records) - 1
	return imp.processChunks(chunks, options, result)
}

// importArchive imports data from archive format (base64 encoded tar.gz or zip)
func (imp *Importer) importArchive(ctx context.Context, data string, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	// Decode base64 data
	archiveData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 archive: %w", err)
	}

	// Create a reader from the decoded data
	reader := bytes.NewReader(archiveData)

	// Try to detect and handle different archive formats
	if imp.isGzipArchive(archiveData) {
		return imp.extractAndImportTarGz(ctx, reader, options, result)
	} else if imp.isZipArchive(archiveData) {
		return imp.extractAndImportZip(ctx, archiveData, options, result)
	}

	return nil, errors.New("unsupported archive format - only tar.gz and zip are supported")
}

// parseMarkdown parses markdown content into conversation chunks
func (imp *Importer) parseMarkdown(data string, options *ImportOptions) ([]types.ConversationChunk, error) {
	_ = []types.ConversationChunk{}

	switch options.ChunkingStrategy {
	case ChunkingParagraph:
		return imp.chunkByParagraph(data, options)
	case ChunkingFixedSize:
		return imp.chunkByFixedSize(data, options)
	case ChunkingConversationTurns:
		return imp.chunkByConversationTurns(data, options)
	case ChunkingAuto:
		return imp.chunkMarkdownAuto(data, options)
	default:
		return imp.chunkMarkdownAuto(data, options)
	}
}

// chunkByParagraph splits markdown into chunks by paragraphs
func (imp *Importer) chunkByParagraph(data string, options *ImportOptions) ([]types.ConversationChunk, error) {
	paragraphs := strings.Split(data, "\n\n")
	chunks := make([]types.ConversationChunk, 0, len(paragraphs))

	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		chunk, err := imp.createChunkFromText(para, options, i)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// chunkByFixedSize splits content into fixed-size chunks
func (imp *Importer) chunkByFixedSize(data string, options *ImportOptions) ([]types.ConversationChunk, error) {
	const maxChunkSize = 2000 // characters
	chunks := make([]types.ConversationChunk, 0)

	for i := 0; i < len(data); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(data) {
			end = len(data)
		}

		chunkText := data[i:end]
		chunk, err := imp.createChunkFromText(chunkText, options, i/maxChunkSize)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// chunkByConversationTurns splits content by conversation turns (user/assistant pattern)
func (imp *Importer) chunkByConversationTurns(data string, options *ImportOptions) ([]types.ConversationChunk, error) {
	// Look for patterns like "User:", "Assistant:", "Human:", "AI:", etc.
	turnPattern := regexp.MustCompile(`(?m)^(User|Assistant|Human|AI|Claude):\s*(.*)`)
	matches := turnPattern.FindAllStringSubmatch(data, -1)

	if len(matches) == 0 {
		// No conversation turns found, treat as single chunk
		chunk, err := imp.createChunkFromText(data, options, 0)
		if err != nil {
			return nil, err
		}
		return []types.ConversationChunk{*chunk}, nil
	}

	chunks := make([]types.ConversationChunk, 0, len(matches))
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}
		speaker := match[1]
		content := match[2]

		// Determine chunk type based on speaker
		chunkType := types.ChunkTypeDiscussion
		if strings.Contains(strings.ToLower(speaker), "assistant") ||
			strings.Contains(strings.ToLower(speaker), "ai") ||
			strings.Contains(strings.ToLower(speaker), "claude") {
			chunkType = types.ChunkTypeSolution
		}

		chunk, err := imp.createChunkFromTextWithType(content, chunkType, options, i)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// chunkMarkdownAuto automatically detects the best chunking strategy for markdown
func (imp *Importer) chunkMarkdownAuto(data string, options *ImportOptions) ([]types.ConversationChunk, error) {
	// Check if it looks like a conversation
	turnPattern := regexp.MustCompile(`(?m)^(User|Assistant|Human|AI|Claude):\s*`)
	if turnPattern.MatchString(data) {
		return imp.chunkByConversationTurns(data, options)
	}

	// Check for clear paragraph structure
	paragraphs := strings.Split(data, "\n\n")
	if len(paragraphs) > 1 {
		return imp.chunkByParagraph(data, options)
	}

	// Default to fixed size
	return imp.chunkByFixedSize(data, options)
}

// parseCSVRecord parses a single CSV record into a conversation chunk
func (imp *Importer) parseCSVRecord(header, record []string, options *ImportOptions, lineNum int) (*types.ConversationChunk, error) {
	if len(record) != len(header) {
		return nil, fmt.Errorf("record length mismatch: expected %d fields, got %d", len(header), len(record))
	}

	// Create a map from the CSV data
	data := make(map[string]string)
	for i, value := range record {
		if i < len(header) {
			data[header[i]] = value
		}
	}

	// Extract required fields
	content := data["content"]
	if content == "" {
		return nil, errors.New("content field is required")
	}

	sessionID := data["session_id"]
	if sessionID == "" {
		sessionID = options.DefaultSessionID
		if sessionID == "" {
			sessionID = "imported_session"
		}
	}

	// Parse chunk type
	chunkTypeStr := data["type"]
	if chunkTypeStr == "" {
		chunkTypeStr = string(types.ChunkTypeDiscussion)
	}
	chunkType := types.ChunkType(chunkTypeStr)

	// Parse timestamp
	timestamp := time.Now().UTC()
	if timestampStr := data["timestamp"]; timestampStr != "" {
		if parsed, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			timestamp = parsed
		}
	}

	// Create metadata
	metadata := types.ChunkMetadata{
		Repository: options.Repository,
		Tags:       append(options.DefaultTags, options.Metadata.Tags...),
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultyModerate,
	}

	// Parse additional metadata fields
	if repository := data["repository"]; repository != "" {
		metadata.Repository = repository
	}
	if outcome := data["outcome"]; outcome != "" {
		metadata.Outcome = types.Outcome(outcome)
	}
	if difficulty := data["difficulty"]; difficulty != "" {
		metadata.Difficulty = types.Difficulty(difficulty)
	}
	if tags := data["tags"]; tags != "" {
		metadata.Tags = append(metadata.Tags, strings.Split(tags, ";")...)
	}

	// Create chunk
	chunk := &types.ConversationChunk{
		ID:        "imported_" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + strconv.Itoa(lineNum),
		SessionID: sessionID,
		Timestamp: timestamp,
		Type:      chunkType,
		Content:   content,
		Summary:   data["summary"],
		Metadata:  metadata,
	}

	return chunk, nil
}

// createChunkFromText creates a conversation chunk from raw text
func (imp *Importer) createChunkFromText(text string, options *ImportOptions, index int) (*types.ConversationChunk, error) {
	return imp.createChunkFromTextWithType(text, types.ChunkTypeDiscussion, options, index)
}

// createChunkFromTextWithType creates a conversation chunk from raw text with specified type
func (imp *Importer) createChunkFromTextWithType(text string, chunkType types.ChunkType, options *ImportOptions, index int) (*types.ConversationChunk, error) {
	sessionID := options.DefaultSessionID
	if sessionID == "" {
		sessionID = "imported_session"
	}

	metadata := types.ChunkMetadata{
		Repository: options.Repository,
		Tags:       append(options.DefaultTags, options.Metadata.Tags...),
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultyModerate,
	}

	// Add import metadata
	if options.Metadata.SourceSystem != "" {
		metadata.Tags = append(metadata.Tags, "source:"+options.Metadata.SourceSystem)
	}

	chunk := &types.ConversationChunk{
		ID:        "imported_" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + strconv.Itoa(index),
		SessionID: sessionID,
		Timestamp: time.Now().UTC(),
		Type:      chunkType,
		Content:   text,
		Summary:   imp.generateSummary(text),
		Metadata:  metadata,
	}

	return chunk, nil
}

// convertConversationData converts generic conversation data to chunks
func (imp *Importer) convertConversationData(data map[string]interface{}, options *ImportOptions) ([]types.ConversationChunk, error) {
	// Try messages format first (ChatML or similar)
	if messages, ok := data["messages"].([]interface{}); ok {
		return imp.convertMessagesFormat(messages, options)
	}

	// Try single content format
	if content, ok := data["content"].(string); ok {
		return imp.convertSingleContentFormat(content, options)
	}

	return nil, errors.New("unrecognized conversation data format")
}

// convertMessagesFormat handles message array format
func (imp *Importer) convertMessagesFormat(messages []interface{}, options *ImportOptions) ([]types.ConversationChunk, error) {
	chunks := make([]types.ConversationChunk, 0, len(messages))

	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue // Skip invalid message format
		}

		chunk, err := imp.convertMessageToChunk(msgMap, options, i)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// convertSingleContentFormat handles single content string format
func (imp *Importer) convertSingleContentFormat(content string, options *ImportOptions) ([]types.ConversationChunk, error) {
	chunk, err := imp.createChunkFromText(content, options, 0)
	if err != nil {
		return nil, err
	}
	return []types.ConversationChunk{*chunk}, nil
}

// convertMessageToChunk converts a message object to a conversation chunk
func (imp *Importer) convertMessageToChunk(message map[string]interface{}, options *ImportOptions, index int) (*types.ConversationChunk, error) {
	content, ok := message["content"].(string)
	if !ok {
		return nil, errors.New("message content must be a string")
	}

	role, _ := message["role"].(string)
	chunkType := types.ChunkTypeDiscussion

	// Map role to chunk type
	switch strings.ToLower(role) {
	case "user", "human":
		chunkType = types.ChunkTypeProblem
	case "assistant", "ai", "claude":
		chunkType = types.ChunkTypeSolution
	case "system":
		chunkType = types.ChunkTypeDiscussion
	}

	return imp.createChunkFromTextWithType(content, chunkType, options, index)
}

// processChunks processes the parsed chunks with validation and conflict resolution
func (imp *Importer) processChunks(chunks []types.ConversationChunk, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	result.TotalItems = len(chunks)

	for i := range chunks {
		chunk := &chunks[i]
		// Validate if requested
		if options.ValidateChunks {
			if err := chunk.Validate(); err != nil {
				result.Errors = append(result.Errors, ImportError{
					Item:    i,
					Message: fmt.Sprintf("validation failed: %v", err),
				})
				result.FailedItems++
				continue
			}
		}

		// Handle conflicts based on policy
		if err := imp.handleConflict(options.ConflictPolicy, result); err != nil {
			result.Errors = append(result.Errors, ImportError{
				Item:    i,
				Message: fmt.Sprintf("conflict resolution failed: %v", err),
			})
			result.FailedItems++
			continue
		}

		result.Chunks = append(result.Chunks, *chunk)
		result.SuccessfulItems++
		result.ProcessedItems++
	}

	// Generate summary
	result.Summary = imp.generateImportSummary(result)

	return result, nil
}

// handleConflict handles conflicts based on the specified policy
func (imp *Importer) handleConflict(policy ConflictPolicy, result *ImportResult) error {
	// This would check for existing chunks with same ID or content
	// For now, we'll assume no conflicts and just return nil
	// In a real implementation, this would query the storage to check for duplicates

	switch policy {
	case ConflictPolicySkip:
		// Skip if exists (would need to check storage)
	case ConflictPolicyOverwrite:
		// Overwrite existing (would need to update in storage)
	case ConflictPolicyMerge:
		// Merge with existing (would need complex merge logic)
	case ConflictPolicyFail:
		// For now, validate that we have valid chunks to import
		if len(result.Chunks) == 0 {
			return errors.New("no valid chunks to import")
		}
		// TODO: Check for actual conflicts in storage
	default:
		return fmt.Errorf("unknown conflict policy: %v", policy)
	}

	return nil
}

// generateSummary generates a brief summary of the content
func (imp *Importer) generateSummary(content string) string {
	// Simple summary generation - take first 100 characters
	summary := content
	if len(summary) > 100 {
		summary = summary[:100] + "..."
	}
	return strings.ReplaceAll(summary, "\n", " ")
}

// isGzipArchive checks if the data is a gzip archive
func (imp *Importer) isGzipArchive(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

// isZipArchive checks if the data is a zip archive
func (imp *Importer) isZipArchive(data []byte) bool {
	return len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4b && data[2] == 0x03 && data[3] == 0x04
}

// extractAndImportTarGz extracts and imports from tar.gz archive
func (imp *Importer) extractAndImportTarGz(_ context.Context, reader io.Reader, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if closeErr := gzipReader.Close(); closeErr != nil {
			imp.logger.Printf("Warning: failed to close gzip reader: %v", closeErr)
		}
	}()

	tarReader := tar.NewReader(gzipReader)

	allChunks := make([]types.ConversationChunk, 0)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Message: fmt.Sprintf("failed to read file %s: %v", header.Name, err),
				Data:    header.Name,
			})
			result.FailedItems++
			continue
		}

		// Process the file based on its extension
		chunks, err := imp.processArchiveFile(header.Name, string(content), options)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Message: fmt.Sprintf("failed to process file %s: %v", header.Name, err),
				Data:    header.Name,
			})
			result.FailedItems++
			continue
		}

		allChunks = append(allChunks, chunks...)
	}

	result.TotalItems = len(allChunks)
	return imp.processChunks(allChunks, options, result)
}

// extractAndImportZip extracts and imports from zip archive
func (imp *Importer) extractAndImportZip(_ context.Context, data []byte, options *ImportOptions, result *ImportResult) (*ImportResult, error) {
	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	allChunks := make([]types.ConversationChunk, 0)

	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Open file
		fileReader, err := file.Open()
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Message: fmt.Sprintf("failed to open file %s: %v", file.Name, err),
				Data:    file.Name,
			})
			result.FailedItems++
			continue
		}

		// Read content
		content, err := io.ReadAll(fileReader)
		if closeErr := fileReader.Close(); closeErr != nil {
			imp.logger.Printf("Warning: failed to close file reader: %v", closeErr)
		}
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Message: fmt.Sprintf("failed to read file %s: %v", file.Name, err),
				Data:    file.Name,
			})
			result.FailedItems++
			continue
		}

		// Process the file
		chunks, err := imp.processArchiveFile(file.Name, string(content), options)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Message: fmt.Sprintf("failed to process file %s: %v", file.Name, err),
				Data:    file.Name,
			})
			result.FailedItems++
			continue
		}

		allChunks = append(allChunks, chunks...)
	}

	result.TotalItems = len(allChunks)
	return imp.processChunks(allChunks, options, result)
}

// processArchiveFile processes a single file from an archive
func (imp *Importer) processArchiveFile(filename, content string, options *ImportOptions) ([]types.ConversationChunk, error) {
	// Determine format based on file extension
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".json":
		return imp.parseJSONContent(content, options, filename)
	case ".md", ".markdown":
		return imp.parseMarkdownContent(content, options, filename)
	case ".csv":
		return imp.parseCSVContent(content, options, filename)
	case ".txt":
		// Treat as markdown for simplicity
		return imp.parseMarkdownContent(content, options, filename)
	default:
		// Skip unsupported file types
		return []types.ConversationChunk{}, nil
	}
}

// parseJSONContent parses JSON content and returns chunks directly
func (imp *Importer) parseJSONContent(content string, _ *ImportOptions, filename string) ([]types.ConversationChunk, error) {
	// Try to parse as array of chunks first
	var chunks []types.ConversationChunk
	if err := json.Unmarshal([]byte(content), &chunks); err == nil {
		return chunks, nil
	}

	// Try to parse as single chunk
	var chunk types.ConversationChunk
	if err := json.Unmarshal([]byte(content), &chunk); err == nil {
		return []types.ConversationChunk{chunk}, nil
	}

	return nil, fmt.Errorf("failed to parse JSON from file %s", filename)
}

// parseMarkdownContent parses markdown content and returns chunks directly
func (imp *Importer) parseMarkdownContent(content string, options *ImportOptions, filename string) ([]types.ConversationChunk, error) {
	// Simple markdown parsing - create a single chunk from the content
	chunk := types.ConversationChunk{
		ID:        uuid.New().String(),
		SessionID: options.DefaultSessionID,
		Content:   content,
		Type:      types.ChunkTypeDiscussion,
		Timestamp: time.Now(),
		Metadata: types.ChunkMetadata{
			Repository: options.Repository,
			Tags:       []string{"archive-import", "file:" + filename},
		},
	}

	return []types.ConversationChunk{chunk}, nil
}

// parseCSVContent parses CSV content and returns chunks directly
func (imp *Importer) parseCSVContent(content string, options *ImportOptions, filename string) ([]types.ConversationChunk, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV from file %s: %w", filename, err)
	}

	if len(records) == 0 {
		return []types.ConversationChunk{}, nil
	}

	// Parse header and records
	header := records[0]
	chunks := make([]types.ConversationChunk, 0, len(records)-1)

	for i, record := range records[1:] {
		chunk, err := imp.parseCSVRecord(header, record, options, i+1)
		if err != nil {
			// Skip invalid records but continue processing
			continue
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// generateImportSummary generates a summary of the import operation
func (imp *Importer) generateImportSummary(result *ImportResult) string {
	return fmt.Sprintf("Imported %d/%d items successfully (%d failed, %d skipped)",
		result.SuccessfulItems, result.TotalItems, result.FailedItems, result.SkippedItems)
}

// isBase64 checks if a string is base64 encoded
func (imp *Importer) isBase64(s string) bool {
	// Simple heuristic - check if it's a reasonable length and contains only base64 characters
	if len(s) < 100 {
		return false
	}

	validChars := regexp.MustCompile(`^[A-Za-z0-9+/=]+$`)
	return validChars.MatchString(s) && len(s)%4 == 0
}
