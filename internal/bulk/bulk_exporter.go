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
	"fmt"
	"log"
	"mcp-memory/internal/storage"
	"mcp-memory/pkg/types"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ExportFormat represents the format for exported data
type ExportFormat string

const (
	// ExportFormatJSON exports data in JSON format
	ExportFormatJSON ExportFormat = "json"
	// ExportFormatMarkdown exports data in Markdown format
	ExportFormatMarkdown ExportFormat = "markdown"
	// ExportFormatCSV exports data in CSV format
	ExportFormatCSV ExportFormat = "csv"
	// ExportFormatArchive exports data as an archive
	ExportFormatArchive ExportFormat = "archive"
)

// CompressionType represents compression options
type CompressionType string

const (
	// CompressionNone indicates no compression
	CompressionNone CompressionType = "none"
	// CompressionGzip indicates gzip compression
	CompressionGzip CompressionType = "gzip"
	// CompressionZip indicates zip compression
	CompressionZip CompressionType = "zip"
)

// ExportOptions configures export behavior
type ExportOptions struct {
	Format           ExportFormat     `json:"format"`
	Compression      CompressionType  `json:"compression"`
	IncludeVectors   bool             `json:"include_vectors"`
	IncludeMetadata  bool             `json:"include_metadata"`
	IncludeRelations bool             `json:"include_relations"`
	PrettyPrint      bool             `json:"pretty_print"`
	Filter           ExportFilter     `json:"filter"`
	Sorting          ExportSorting    `json:"sorting"`
	Pagination       ExportPagination `json:"pagination"`
}

// ExportFilter defines filtering criteria for export
type ExportFilter struct {
	Repository    *string            `json:"repository,omitempty"`
	SessionIDs    []string           `json:"session_ids,omitempty"`
	ChunkTypes    []types.ChunkType  `json:"chunk_types,omitempty"`
	DateRange     *ExportDateRange   `json:"date_range,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	Outcomes      []types.Outcome    `json:"outcomes,omitempty"`
	Difficulties  []types.Difficulty `json:"difficulties,omitempty"`
	MinRelevance  *float64           `json:"min_relevance,omitempty"`
	SearchQuery   *string            `json:"search_query,omitempty"`
	ContentFilter *string            `json:"content_filter,omitempty"` // Regex pattern
}

// ExportDateRange defines a date range for filtering
type ExportDateRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// ExportSorting defines sorting criteria
type ExportSorting struct {
	Field string `json:"field"` // timestamp, relevance, type, repository
	Order string `json:"order"` // asc, desc
}

// ExportPagination defines pagination options
type ExportPagination struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ExportResult represents the result of an export operation
type ExportResult struct {
	Format        ExportFormat `json:"format"`
	TotalItems    int          `json:"total_items"`
	ExportedItems int          `json:"exported_items"`
	DataSize      int64        `json:"data_size_bytes"`
	Data          string       `json:"data"` // Base64 encoded if binary
	Metadata      ExportMeta   `json:"metadata"`
	Warnings      []string     `json:"warnings,omitempty"`
	GeneratedAt   time.Time    `json:"generated_at"`
}

// ExportMeta contains metadata about the export
type ExportMeta struct {
	ExportID      string                 `json:"export_id"`
	Repository    string                 `json:"repository,omitempty"`
	SourceSystem  string                 `json:"source_system"`
	Version       string                 `json:"version"`
	TotalSessions int                    `json:"total_sessions"`
	DateRange     *ExportDateRange       `json:"date_range,omitempty"`
	ChunkTypes    map[string]int         `json:"chunk_types"`
	Tags          map[string]int         `json:"tags"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
}

// Exporter handles exporting memories to various formats
type Exporter struct {
	storage storage.VectorStore
	logger  *log.Logger
}

// NewExporter creates a new bulk exporter
func NewExporter(storage storage.VectorStore, logger *log.Logger) *Exporter {
	if logger == nil {
		logger = log.New(log.Writer(), "[BulkExporter] ", log.LstdFlags)
	}

	return &Exporter{
		storage: storage,
		logger:  logger,
	}
}

// Export exports memories based on the provided options
func (exp *Exporter) Export(ctx context.Context, options ExportOptions) (*ExportResult, error) {
	// Query chunks based on filter
	chunks, err := exp.queryChunks(ctx, options.Filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}

	// Apply sorting
	exp.sortChunks(chunks, options.Sorting)

	// Apply pagination
	chunks = exp.paginateChunks(chunks, options.Pagination)

	// Generate metadata
	metadata := exp.generateMetadata(chunks)

	// Export based on format
	var data string
	var dataSize int64

	switch options.Format {
	case ExportFormatJSON:
		data, dataSize, err = exp.exportJSON(chunks, options)
	case ExportFormatMarkdown:
		data, dataSize = exp.exportMarkdown(chunks, options)
	case ExportFormatCSV:
		data, dataSize, err = exp.exportCSV(chunks, options)
	case ExportFormatArchive:
		data, dataSize, err = exp.exportArchive(chunks, options, metadata)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", options.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to export data: %w", err)
	}

	// Apply compression if requested
	if options.Compression != CompressionNone {
		data, dataSize, err = exp.compressData(data, options.Compression)
		if err != nil {
			return nil, fmt.Errorf("failed to compress data: %w", err)
		}
	}

	result := &ExportResult{
		Format:        options.Format,
		TotalItems:    len(chunks),
		ExportedItems: len(chunks),
		DataSize:      dataSize,
		Data:          data,
		Metadata:      metadata,
		GeneratedAt:   time.Now().UTC(),
	}

	return result, nil
}

// queryChunks queries chunks based on the filter criteria
func (exp *Exporter) queryChunks(ctx context.Context, filter ExportFilter) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk

	// If repository is specified, query by repository
	if filter.Repository != nil {
		repoChunks, err := exp.storage.ListByRepository(ctx, *filter.Repository, 10000, 0)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, repoChunks...)
	} else {
		// Get all chunks
		allChunks, err := exp.storage.GetAllChunks(ctx)
		if err != nil {
			return nil, err
		}
		chunks = allChunks
	}

	// Apply filters
	filteredChunks := make([]types.ConversationChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if exp.matchesFilter(chunk, filter) {
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	return filteredChunks, nil
}

// matchesFilter checks if a chunk matches the filter criteria
func (exp *Exporter) matchesFilter(chunk types.ConversationChunk, filter ExportFilter) bool {
	// Session ID filter
	if len(filter.SessionIDs) > 0 {
		found := false
		for _, sessionID := range filter.SessionIDs {
			if chunk.SessionID == sessionID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Chunk type filter
	if len(filter.ChunkTypes) > 0 {
		found := false
		for _, chunkType := range filter.ChunkTypes {
			if chunk.Type == chunkType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Date range filter
	if filter.DateRange != nil {
		if filter.DateRange.Start != nil && chunk.Timestamp.Before(*filter.DateRange.Start) {
			return false
		}
		if filter.DateRange.End != nil && chunk.Timestamp.After(*filter.DateRange.End) {
			return false
		}
	}

	// Tags filter
	if len(filter.Tags) > 0 {
		for _, filterTag := range filter.Tags {
			found := false
			for _, chunkTag := range chunk.Metadata.Tags {
				if chunkTag == filterTag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Outcome filter
	if len(filter.Outcomes) > 0 {
		found := false
		for _, outcome := range filter.Outcomes {
			if chunk.Metadata.Outcome == outcome {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Difficulty filter
	if len(filter.Difficulties) > 0 {
		found := false
		for _, difficulty := range filter.Difficulties {
			if chunk.Metadata.Difficulty == difficulty {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Content filter (regex)
	if filter.ContentFilter != nil {
		if matched, _ := regexp.MatchString(*filter.ContentFilter, chunk.Content); !matched {
			return false
		}
	}

	return true
}

// sortChunks sorts chunks based on the sorting criteria
func (exp *Exporter) sortChunks(chunks []types.ConversationChunk, sorting ExportSorting) {
	if sorting.Field == "" {
		sorting.Field = "timestamp"
	}
	if sorting.Order == "" {
		sorting.Order = "desc"
	}

	sort.Slice(chunks, func(i, j int) bool {
		var less bool

		switch sorting.Field {
		case "timestamp":
			less = chunks[i].Timestamp.Before(chunks[j].Timestamp)
		case "type":
			less = string(chunks[i].Type) < string(chunks[j].Type)
		case "repository":
			less = chunks[i].Metadata.Repository < chunks[j].Metadata.Repository
		case "session_id":
			less = chunks[i].SessionID < chunks[j].SessionID
		default:
			less = chunks[i].Timestamp.Before(chunks[j].Timestamp)
		}

		if sorting.Order == "desc" {
			return !less
		}
		return less
	})
}

// paginateChunks applies pagination to the chunks
func (exp *Exporter) paginateChunks(chunks []types.ConversationChunk, pagination ExportPagination) []types.ConversationChunk {
	if pagination.Limit <= 0 && pagination.Offset <= 0 {
		return chunks
	}

	start := pagination.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(chunks) {
		return []types.ConversationChunk{}
	}

	end := len(chunks)
	if pagination.Limit > 0 {
		end = start + pagination.Limit
		if end > len(chunks) {
			end = len(chunks)
		}
	}

	return chunks[start:end]
}

// exportJSON exports chunks as JSON
func (exp *Exporter) exportJSON(chunks []types.ConversationChunk, options ExportOptions) (string, int64, error) {
	// Prepare chunks for export
	exportChunks := exp.prepareChunksForExport(chunks, options)

	var data []byte
	var err error

	if options.PrettyPrint {
		data, err = json.MarshalIndent(exportChunks, "", "  ")
	} else {
		data, err = json.Marshal(exportChunks)
	}

	if err != nil {
		return "", 0, err
	}

	return string(data), int64(len(data)), nil
}

// exportMarkdown exports chunks as markdown
func (exp *Exporter) exportMarkdown(chunks []types.ConversationChunk, options ExportOptions) (string, int64) {
	var builder strings.Builder

	// Write header
	builder.WriteString("# Memory Export\n\n")
	builder.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Total Chunks: %d\n\n", len(chunks)))

	// Group by session
	sessionChunks := make(map[string][]types.ConversationChunk)
	for _, chunk := range chunks {
		sessionChunks[chunk.SessionID] = append(sessionChunks[chunk.SessionID], chunk)
	}

	// Write sessions
	for sessionID, sessionChunks := range sessionChunks {
		builder.WriteString(fmt.Sprintf("## Session: %s\n\n", sessionID))

		for _, chunk := range sessionChunks {
			builder.WriteString(fmt.Sprintf("### %s - %s\n",
				chunk.Type, chunk.Timestamp.Format("2006-01-02 15:04:05")))

			if chunk.Summary != "" {
				builder.WriteString(fmt.Sprintf("**Summary:** %s\n\n", chunk.Summary))
			}

			builder.WriteString(chunk.Content)
			builder.WriteString("\n\n")

			if options.IncludeMetadata {
				builder.WriteString("**Metadata:**\n")
				builder.WriteString(fmt.Sprintf("- Repository: %s\n", chunk.Metadata.Repository))
				builder.WriteString(fmt.Sprintf("- Outcome: %s\n", chunk.Metadata.Outcome))
				builder.WriteString(fmt.Sprintf("- Difficulty: %s\n", chunk.Metadata.Difficulty))
				if len(chunk.Metadata.Tags) > 0 {
					builder.WriteString(fmt.Sprintf("- Tags: %s\n", strings.Join(chunk.Metadata.Tags, ", ")))
				}
				builder.WriteString("\n")
			}

			builder.WriteString("---\n\n")
		}
	}

	data := builder.String()
	return data, int64(len(data))
}

// exportCSV exports chunks as CSV
func (exp *Exporter) exportCSV(chunks []types.ConversationChunk, options ExportOptions) (string, int64, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Write header
	header := []string{
		"id", "session_id", "timestamp", "type", "content", "summary",
		"repository", "outcome", "difficulty", "tags",
	}

	if options.IncludeMetadata {
		header = append(header, "branch", "files_modified", "tools_used")
	}

	if err := writer.Write(header); err != nil {
		return "", 0, err
	}

	// Write data
	for _, chunk := range chunks {
		record := []string{
			chunk.ID,
			chunk.SessionID,
			chunk.Timestamp.Format(time.RFC3339),
			string(chunk.Type),
			chunk.Content,
			chunk.Summary,
			chunk.Metadata.Repository,
			string(chunk.Metadata.Outcome),
			string(chunk.Metadata.Difficulty),
			strings.Join(chunk.Metadata.Tags, ";"),
		}

		if options.IncludeMetadata {
			record = append(record,
				chunk.Metadata.Branch,
				strings.Join(chunk.Metadata.FilesModified, ";"),
				strings.Join(chunk.Metadata.ToolsUsed, ";"),
			)
		}

		if err := writer.Write(record); err != nil {
			return "", 0, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", 0, err
	}

	data := buffer.String()
	return data, int64(len(data)), nil
}

// exportArchive exports chunks as a compressed archive
func (exp *Exporter) exportArchive(chunks []types.ConversationChunk, options ExportOptions, metadata ExportMeta) (string, int64, error) {
	var buffer bytes.Buffer

	// Create tar writer
	tarWriter := tar.NewWriter(&buffer)
	defer func() { _ = tarWriter.Close() }()

	// Add metadata file
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", 0, err
	}

	if err := exp.addFileToTar(tarWriter, "metadata.json", metadataJSON); err != nil {
		return "", 0, err
	}

	// Add chunks as JSON
	chunksJSON, err := json.MarshalIndent(chunks, "", "  ")
	if err != nil {
		return "", 0, err
	}

	if err := exp.addFileToTar(tarWriter, "chunks.json", chunksJSON); err != nil {
		return "", 0, err
	}

	// Add chunks as markdown for human readability
	markdownData, _ := exp.exportMarkdown(chunks, options)

	if err := exp.addFileToTar(tarWriter, "export.md", []byte(markdownData)); err != nil {
		return "", 0, err
	}

	// Close tar writer to finalize
	if err := tarWriter.Close(); err != nil {
		return "", 0, err
	}

	// Encode as base64
	encoded := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return encoded, int64(len(encoded)), nil
}

// addFileToTar adds a file to the tar archive
func (exp *Exporter) addFileToTar(tarWriter *tar.Writer, filename string, data []byte) error {
	header := &tar.Header{
		Name:    filename,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err := tarWriter.Write(data)
	return err
}

// compressData compresses the data using the specified compression type
func (exp *Exporter) compressData(data string, compression CompressionType) (string, int64, error) {
	switch compression {
	case CompressionGzip:
		return exp.compressGzip(data)
	case CompressionZip:
		return exp.compressZip(data)
	case CompressionNone:
		return data, int64(len(data)), nil
	default:
		return data, int64(len(data)), nil
	}
}

// compressGzip compresses data using gzip
func (exp *Exporter) compressGzip(data string) (string, int64, error) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)

	if _, err := gzipWriter.Write([]byte(data)); err != nil {
		return "", 0, err
	}

	if err := gzipWriter.Close(); err != nil {
		return "", 0, err
	}

	compressed := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return compressed, int64(len(compressed)), nil
}

// compressZip compresses data using zip
func (exp *Exporter) compressZip(data string) (string, int64, error) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	writer, err := zipWriter.Create("export.txt")
	if err != nil {
		return "", 0, err
	}

	if _, err := writer.Write([]byte(data)); err != nil {
		return "", 0, err
	}

	if err := zipWriter.Close(); err != nil {
		return "", 0, err
	}

	compressed := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return compressed, int64(len(compressed)), nil
}

// prepareChunksForExport prepares chunks for export by filtering out unwanted fields
func (exp *Exporter) prepareChunksForExport(chunks []types.ConversationChunk, options ExportOptions) []map[string]interface{} {
	exportChunks := make([]map[string]interface{}, len(chunks))

	for i, chunk := range chunks {
		exportChunk := map[string]interface{}{
			"id":         chunk.ID,
			"session_id": chunk.SessionID,
			"timestamp":  chunk.Timestamp,
			"type":       chunk.Type,
			"content":    chunk.Content,
			"summary":    chunk.Summary,
		}

		if options.IncludeMetadata {
			exportChunk["metadata"] = chunk.Metadata
		}

		if options.IncludeVectors && len(chunk.Embeddings) > 0 {
			exportChunk["embeddings"] = chunk.Embeddings
		}

		if options.IncludeRelations && len(chunk.RelatedChunks) > 0 {
			exportChunk["related_chunks"] = chunk.RelatedChunks
		}

		exportChunks[i] = exportChunk
	}

	return exportChunks
}

// generateMetadata generates export metadata
func (exp *Exporter) generateMetadata(chunks []types.ConversationChunk) ExportMeta {
	metadata := ExportMeta{
		ExportID:     fmt.Sprintf("export_%d", time.Now().Unix()),
		SourceSystem: "mcp-memory",
		Version:      "2.0",
		ChunkTypes:   make(map[string]int),
		Tags:         make(map[string]int),
	}

	// Collect statistics
	sessions := make(map[string]bool)
	var earliest, latest time.Time

	for i, chunk := range chunks {
		// Session tracking
		sessions[chunk.SessionID] = true

		// Date range
		if i == 0 {
			earliest = chunk.Timestamp
			latest = chunk.Timestamp
		} else {
			if chunk.Timestamp.Before(earliest) {
				earliest = chunk.Timestamp
			}
			if chunk.Timestamp.After(latest) {
				latest = chunk.Timestamp
			}
		}

		// Chunk types
		metadata.ChunkTypes[string(chunk.Type)]++

		// Tags
		for _, tag := range chunk.Metadata.Tags {
			metadata.Tags[tag]++
		}

		// Repository (use first non-empty)
		if metadata.Repository == "" && chunk.Metadata.Repository != "" {
			metadata.Repository = chunk.Metadata.Repository
		}
	}

	metadata.TotalSessions = len(sessions)

	if len(chunks) > 0 {
		metadata.DateRange = &ExportDateRange{
			Start: &earliest,
			End:   &latest,
		}
	}

	return metadata
}
