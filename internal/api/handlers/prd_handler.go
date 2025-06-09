// Package handlers provides HTTP request handlers for PRD import and processing.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/prd"
	"lerian-mcp-memory/pkg/types"
)

// Content type constants
const (
	ContentTypeText = "text"
)

// PRDHandler handles PRD import and processing requests
type PRDHandler struct {
	parser   *prd.Parser
	analyzer *prd.Analyzer
	storage  PRDStorage
	config   PRDHandlerConfig
}

// PRDStorage defines the interface for PRD storage operations
type PRDStorage interface {
	Store(ctx context.Context, doc *types.PRDDocument) error
	Get(ctx context.Context, id string) (*types.PRDDocument, error)
	List(ctx context.Context, filters PRDFilters) ([]*types.PRDDocument, error)
	Update(ctx context.Context, doc *types.PRDDocument) error
	Delete(ctx context.Context, id string) error
}

// PRDFilters represents filters for PRD listing
type PRDFilters struct {
	Status      types.PRDStatus   `json:"status,omitempty"`
	Priority    types.PRDPriority `json:"priority,omitempty"`
	ProjectType types.ProjectType `json:"project_type,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
}

// PRDHandlerConfig represents configuration for PRD handler
type PRDHandlerConfig struct {
	MaxFileSize       int64         `json:"max_file_size"`
	AllowedFormats    []string      `json:"allowed_formats"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`
	EnableAnalysis    bool          `json:"enable_analysis"`
	AutoProcess       bool          `json:"auto_process"`
}

// DefaultPRDHandlerConfig returns default configuration
func DefaultPRDHandlerConfig() PRDHandlerConfig {
	return PRDHandlerConfig{
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		AllowedFormats:    []string{"markdown", "md", ContentTypeText, "txt", "plain"},
		ProcessingTimeout: 60 * time.Second,
		EnableAnalysis:    true,
		AutoProcess:       true,
	}
}

// NewPRDHandler creates a new PRD handler
func NewPRDHandler(storage PRDStorage, config PRDHandlerConfig) *PRDHandler {
	parserConfig := prd.DefaultParserConfig()
	parserConfig.MaxFileSize = config.MaxFileSize

	analyzerConfig := prd.DefaultAnalyzerConfig()
	analyzerConfig.EnableAIAnalysis = config.EnableAnalysis

	return &PRDHandler{
		parser:   prd.NewParser(parserConfig),
		analyzer: prd.NewAnalyzer(analyzerConfig),
		storage:  storage,
		config:   config,
	}
}

// ImportPRD handles PRD import requests
func (h *PRDHandler) ImportPRD(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.config.ProcessingTimeout)
	defer cancel()

	// Parse multipart form or JSON request
	var req types.PRDImportRequest
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		if err := h.parseMultipartRequest(r, &req); err != nil {
			response.WriteError(w, http.StatusBadRequest, "Invalid multipart request", err.Error())
			return
		}
	} else if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
			return
		}
	} else {
		response.WriteError(w, http.StatusBadRequest, "Unsupported content type", "Use multipart/form-data or application/json")
		return
	}

	// Validate request
	if err := h.validateImportRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Process the PRD
	doc, err := h.processPRD(ctx, &req)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to process PRD", err.Error())
		return
	}

	// Store the document
	if err := h.storage.Store(ctx, doc); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to store PRD", err.Error())
		return
	}

	// Create response
	resp := types.PRDImportResponse{
		DocumentID: doc.ID,
		Status:     doc.Status,
		Message:    "PRD imported successfully",
		Processing: doc.Processing,
		NextSteps:  h.generateNextSteps(doc),
	}

	response.WriteSuccess(w, resp)
}

// GetPRD handles individual PRD retrieval
func (h *PRDHandler) GetPRD(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract document ID from URL path
	id := h.extractIDFromPath(r.URL.Path)
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "Missing document ID", "Document ID is required")
		return
	}

	// Retrieve document
	doc, err := h.storage.Get(ctx, id)
	if err != nil {
		response.WriteError(w, http.StatusNotFound, "Document not found", err.Error())
		return
	}

	response.WriteSuccess(w, doc)
}

// ListPRDs handles PRD listing with filters
func (h *PRDHandler) ListPRDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := h.parseListFilters(r)

	// Retrieve documents
	docs, err := h.storage.List(ctx, filters)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to list PRDs", err.Error())
		return
	}

	// Create response with metadata
	resp := map[string]interface{}{
		"prds":    docs,
		"count":   len(docs),
		"filters": filters,
	}

	response.WriteSuccess(w, resp)
}

// UpdatePRD handles PRD updates
func (h *PRDHandler) UpdatePRD(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract document ID
	id := h.extractIDFromPath(r.URL.Path)
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "Missing document ID", "Document ID is required")
		return
	}

	// Parse update request
	var updateReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Get existing document
	doc, err := h.storage.Get(ctx, id)
	if err != nil {
		response.WriteError(w, http.StatusNotFound, "Document not found", err.Error())
		return
	}

	// Apply updates
	h.applyUpdates(doc, updateReq)

	// Update document
	if err := h.storage.Update(ctx, doc); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to update PRD", err.Error())
		return
	}

	response.WriteSuccess(w, doc)
}

// DeletePRD handles PRD deletion
func (h *PRDHandler) DeletePRD(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract document ID
	id := h.extractIDFromPath(r.URL.Path)
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "Missing document ID", "Document ID is required")
		return
	}

	// Delete document
	if err := h.storage.Delete(ctx, id); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to delete PRD", err.Error())
		return
	}

	response.WriteSuccess(w, map[string]string{
		"message": "PRD deleted successfully",
		"id":      id,
	})
}

// parseMultipartRequest parses multipart form data
func (h *PRDHandler) parseMultipartRequest(r *http.Request, req *types.PRDImportRequest) error {
	if err := r.ParseMultipartForm(h.config.MaxFileSize); err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	// Check file size
	if header.Size > h.config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", header.Size, h.config.MaxFileSize)
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Detect format from filename
	format := h.detectFormatFromFilename(header.Filename)

	// Populate request
	req.Name = r.FormValue("name")
	if req.Name == "" {
		req.Name = header.Filename
	}
	req.Content = string(content)
	req.Format = format
	req.Encoding = "utf-8"

	// Parse options
	if optionsStr := r.FormValue("options"); optionsStr != "" {
		if err := json.Unmarshal([]byte(optionsStr), &req.Options); err != nil {
			return fmt.Errorf("invalid options JSON: %w", err)
		}
	} else {
		req.Options = types.ImportOptions{
			AutoProcess:        h.config.AutoProcess,
			AutoAnalyze:        h.config.EnableAnalysis,
			ExtractUserStories: true,
			GenerateTasks:      true,
			AIProcessing:       true,
			ValidationLevel:    "standard",
		}
	}

	return nil
}

// validateImportRequest validates the import request
func (h *PRDHandler) validateImportRequest(req *types.PRDImportRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if req.Content == "" {
		return fmt.Errorf("content is required")
	}

	if len(req.Content) > int(h.config.MaxFileSize) {
		return fmt.Errorf("content size exceeds maximum allowed size")
	}

	// Validate format
	formatValid := false
	for _, allowedFormat := range h.config.AllowedFormats {
		if strings.EqualFold(req.Format, allowedFormat) {
			formatValid = true
			break
		}
	}

	if !formatValid {
		return fmt.Errorf("format '%s' is not supported. Allowed formats: %v", req.Format, h.config.AllowedFormats)
	}

	return nil
}

// processPRD processes the PRD document
func (h *PRDHandler) processPRD(ctx context.Context, req *types.PRDImportRequest) (*types.PRDDocument, error) {
	_ = ctx // unused parameter, kept for future use
	// Validate content first
	if err := h.parser.ValidateContent(req.Content, req.Format, req.Encoding); err != nil {
		return nil, fmt.Errorf("content validation failed: %w", err)
	}

	// Parse document
	doc, err := h.parser.ParseDocument(req.Content, req.Format, req.Encoding)
	if err != nil {
		return nil, fmt.Errorf("document parsing failed: %w", err)
	}

	// Set document name and metadata
	doc.Name = req.Name
	if req.Metadata != nil {
		// Apply metadata from request
		for key, value := range req.Metadata {
			switch key {
			case "author":
				doc.Metadata.Author = value
			case "owner":
				doc.Metadata.Owner = value
			case "priority":
				if priority := types.PRDPriority(value); priority != "" {
					doc.Metadata.Priority = priority
				}
			case "domain":
				doc.Metadata.Domain = value
			}
		}
	}

	// Analyze document if enabled
	if req.Options.AutoAnalyze && h.config.EnableAnalysis {
		if err := h.analyzer.AnalyzeDocument(doc); err != nil {
			// Don't fail on analysis errors, just log them
			doc.Processing.Warnings = append(doc.Processing.Warnings,
				fmt.Sprintf("Analysis failed: %v", err))
		} else {
			now := time.Now()
			doc.Timestamps.Analyzed = &now
			doc.Status = types.PRDStatusAnalyzed
		}
	}

	return doc, nil
}

// generateNextSteps generates next steps based on document state
func (h *PRDHandler) generateNextSteps(doc *types.PRDDocument) []string {
	steps := []string{}

	switch doc.Status {
	case types.PRDStatusImported:
		steps = append(steps, "Process the document to extract structure", "Analyze content for completeness")
	case types.PRDStatusProcessed:
		steps = append(steps, "Analyze document quality", "Extract user stories and requirements")
	case types.PRDStatusAnalyzed:
		steps = append(steps, "Generate tasks from requirements", "Create project timeline")
	}

	// Add specific recommendations based on analysis
	if doc.Analysis.QualityScore < 0.7 {
		steps = append(steps, "Review and improve document quality")
	}

	if len(doc.Analysis.MissingElements) > 0 {
		steps = append(steps, "Add missing required sections")
	}

	return steps
}

// Helper functions

func (h *PRDHandler) extractIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (h *PRDHandler) parseListFilters(r *http.Request) PRDFilters {
	filters := PRDFilters{}

	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = types.PRDStatus(status)
	}

	if priority := r.URL.Query().Get("priority"); priority != "" {
		filters.Priority = types.PRDPriority(priority)
	}

	if projectType := r.URL.Query().Get("project_type"); projectType != "" {
		filters.ProjectType = types.ProjectType(projectType)
	}

	if tags := r.URL.Query().Get("tags"); tags != "" {
		filters.Tags = strings.Split(tags, ",")
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filters.Offset = offset
		}
	}

	// Set defaults
	if filters.Limit == 0 {
		filters.Limit = 20
	}

	return filters
}

func (h *PRDHandler) detectFormatFromFilename(filename string) string {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])

	switch ext {
	case "md":
		return "markdown"
	case "txt":
		return ContentTypeText
	case ContentTypeText:
		return ContentTypeText
	default:
		return ContentTypeText
	}
}

func (h *PRDHandler) applyUpdates(doc *types.PRDDocument, updates map[string]interface{}) {
	// Apply updates to document fields
	if name, ok := updates["name"].(string); ok {
		doc.Name = name
	}

	if status, ok := updates["status"].(string); ok {
		doc.Status = types.PRDStatus(status)
	}

	// Update metadata if provided
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		if author, ok := metadata["author"].(string); ok {
			doc.Metadata.Author = author
		}
		if owner, ok := metadata["owner"].(string); ok {
			doc.Metadata.Owner = owner
		}
		if priority, ok := metadata["priority"].(string); ok {
			doc.Metadata.Priority = types.PRDPriority(priority)
		}
	}

	// Update timestamp
	doc.Timestamps.Updated = time.Now()
}
