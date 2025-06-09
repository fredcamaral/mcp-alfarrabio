package documents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PRDEntity represents a Product Requirements Document
type PRDEntity struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	Content           string            `json:"content"`
	Metadata          map[string]string `json:"metadata"`
	Sections          []Section         `json:"sections"`
	GeneratedAt       time.Time         `json:"generated_at"`
	LastModified      time.Time         `json:"last_modified"`
	Version           string            `json:"version"`
	Status            DocumentStatus    `json:"status"`
	ComplexityScore   int               `json:"complexity_score"`
	EstimatedDuration string            `json:"estimated_duration"`
	Author            string            `json:"author"`
	Repository        string            `json:"repository"`
	ParsedContent     ParsedContent     `json:"parsed_content,omitempty"`
}

// TRDEntity represents a Technical Requirements Document
type TRDEntity struct {
	ID             string            `json:"id"`
	PRDID          string            `json:"prd_id"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	Metadata       map[string]string `json:"metadata"`
	Sections       []Section         `json:"sections"`
	GeneratedAt    time.Time         `json:"generated_at"`
	LastModified   time.Time         `json:"last_modified"`
	Version        string            `json:"version"`
	Status         DocumentStatus    `json:"status"`
	TechnicalStack []string          `json:"technical_stack"`
	Architecture   string            `json:"architecture"`
	Dependencies   []string          `json:"dependencies"`
	Repository     string            `json:"repository"`
}

// MainTask represents a main task document
type MainTask struct {
	ID                 string         `json:"id"`
	PRDID              string         `json:"prd_id"`
	TRDID              string         `json:"trd_id"`
	TaskID             string         `json:"task_id"` // e.g., MT-001
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Phase              string         `json:"phase"`
	DurationEstimate   string         `json:"duration_estimate"`
	Dependencies       []string       `json:"dependencies"`
	Deliverables       []string       `json:"deliverables"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	ComplexityScore    int            `json:"complexity_score"`
	Status             DocumentStatus `json:"status"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	Repository         string         `json:"repository"`
}

// SubTask represents a sub-task document
type SubTask struct {
	ID                 string            `json:"id"`
	MainTaskID         string            `json:"main_task_id"`
	SubTaskID          string            `json:"sub_task_id"` // e.g., ST-MT-001-001
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	EstimatedHours     int               `json:"estimated_hours"`
	ImplementationType string            `json:"implementation_type"`
	Dependencies       []string          `json:"dependencies"`
	AcceptanceCriteria []string          `json:"acceptance_criteria"`
	TechnicalDetails   map[string]string `json:"technical_details"`
	Status             DocumentStatus    `json:"status"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Repository         string            `json:"repository"`
}

// Rule represents a document generation rule
type Rule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        RuleType          `json:"type"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	Version     string            `json:"version"`
	Active      bool              `json:"active"`
	Priority    int               `json:"priority"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Section represents a document section
type Section struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Level    int      `json:"level"`
	Order    int      `json:"order"`
	Keywords []string `json:"keywords,omitempty"`
}

// ParsedContent holds extracted information from document parsing
type ParsedContent struct {
	ProjectName    string            `json:"project_name"`
	Summary        string            `json:"summary"`
	Goals          []string          `json:"goals"`
	Requirements   []string          `json:"requirements"`
	UserStories    []string          `json:"user_stories"`
	TechnicalNotes []string          `json:"technical_notes"`
	Constraints    []string          `json:"constraints"`
	Keywords       []string          `json:"keywords"`
	ExtractedData  map[string]string `json:"extracted_data"`
}

// DocumentStatus represents the status of a document
type DocumentStatus string

const (
	StatusDraft      DocumentStatus = "draft"
	StatusInReview   DocumentStatus = "in_review"
	StatusApproved   DocumentStatus = "approved"
	StatusGenerated  DocumentStatus = "generated"
	StatusProcessing DocumentStatus = "processing"
	StatusCompleted  DocumentStatus = "completed"
	StatusError      DocumentStatus = "error"
)

// RuleType represents the type of generation rule
type RuleType string

const (
	RulePRDGeneration      RuleType = "prd_generation"
	RuleTRDGeneration      RuleType = "trd_generation"
	RuleTaskGeneration     RuleType = "task_generation"
	RuleSubTaskGeneration  RuleType = "subtask_generation"
	RuleComplexityAnalysis RuleType = "complexity_analysis"
	RuleValidation         RuleType = "validation"
)

// Document interface for common document operations
type Document interface {
	GetID() string
	GetTitle() string
	GetContent() string
	GetStatus() DocumentStatus
	SetStatus(status DocumentStatus)
	Validate() error
}

// GetID returns the document ID
func (p *PRDEntity) GetID() string { return p.ID }
func (t *TRDEntity) GetID() string { return t.ID }
func (m *MainTask) GetID() string  { return m.ID }
func (s *SubTask) GetID() string   { return s.ID }

// GetTitle returns the document title
func (p *PRDEntity) GetTitle() string { return p.Title }
func (t *TRDEntity) GetTitle() string { return t.Title }
func (m *MainTask) GetTitle() string  { return m.Name }
func (s *SubTask) GetTitle() string   { return s.Name }

// GetContent returns the document content
func (p *PRDEntity) GetContent() string { return p.Content }
func (t *TRDEntity) GetContent() string { return t.Content }
func (m *MainTask) GetContent() string  { return m.Description }
func (s *SubTask) GetContent() string   { return s.Description }

// GetStatus returns the document status
func (p *PRDEntity) GetStatus() DocumentStatus { return p.Status }
func (t *TRDEntity) GetStatus() DocumentStatus { return t.Status }
func (m *MainTask) GetStatus() DocumentStatus  { return m.Status }
func (s *SubTask) GetStatus() DocumentStatus   { return s.Status }

// SetStatus sets the document status
func (p *PRDEntity) SetStatus(status DocumentStatus) { p.Status = status }
func (t *TRDEntity) SetStatus(status DocumentStatus) { t.Status = status }
func (m *MainTask) SetStatus(status DocumentStatus)  { m.Status = status }
func (s *SubTask) SetStatus(status DocumentStatus)   { s.Status = status }

// Validate validates the PRD entity
func (p *PRDEntity) Validate() error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Title == "" {
		return fmt.Errorf("PRD title is required")
	}
	if p.Content == "" {
		return fmt.Errorf("PRD content is required")
	}
	if p.GeneratedAt.IsZero() {
		p.GeneratedAt = time.Now()
	}
	if p.LastModified.IsZero() {
		p.LastModified = time.Now()
	}
	if p.Version == "" {
		p.Version = "1.0.0"
	}
	if p.Status == "" {
		p.Status = StatusDraft
	}
	return nil
}

// Validate validates the TRD entity
func (t *TRDEntity) Validate() error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.PRDID == "" {
		return fmt.Errorf("TRD must reference a PRD")
	}
	if t.Title == "" {
		return fmt.Errorf("TRD title is required")
	}
	if t.Content == "" {
		return fmt.Errorf("TRD content is required")
	}
	if t.GeneratedAt.IsZero() {
		t.GeneratedAt = time.Now()
	}
	if t.LastModified.IsZero() {
		t.LastModified = time.Now()
	}
	if t.Version == "" {
		t.Version = "1.0.0"
	}
	if t.Status == "" {
		t.Status = StatusDraft
	}
	return nil
}

// Validate validates the MainTask entity
func (m *MainTask) Validate() error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.TaskID == "" {
		return fmt.Errorf("main task ID (e.g., MT-001) is required")
	}
	if m.Name == "" {
		return fmt.Errorf("main task name is required")
	}
	if m.Description == "" {
		return fmt.Errorf("main task description is required")
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = time.Now()
	}
	if m.Status == "" {
		m.Status = StatusDraft
	}
	return nil
}

// Validate validates the SubTask entity
func (s *SubTask) Validate() error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.MainTaskID == "" {
		return fmt.Errorf("sub-task must reference a main task")
	}
	if s.SubTaskID == "" {
		return fmt.Errorf("sub-task ID (e.g., ST-MT-001-001) is required")
	}
	if s.Name == "" {
		return fmt.Errorf("sub-task name is required")
	}
	if s.Description == "" {
		return fmt.Errorf("sub-task description is required")
	}
	if s.EstimatedHours <= 0 {
		return fmt.Errorf("sub-task estimated hours must be greater than 0")
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = time.Now()
	}
	if s.Status == "" {
		s.Status = StatusDraft
	}
	return nil
}

// Validate validates the Rule entity
func (r *Rule) Validate() error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if r.Content == "" {
		return fmt.Errorf("rule content is required")
	}
	if r.Type == "" {
		return fmt.Errorf("rule type is required")
	}
	if r.Version == "" {
		r.Version = "1.0.0"
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = time.Now()
	}
	return nil
}

// ParseSections extracts sections from markdown content
func ParseSections(content string) []Section {
	lines := strings.Split(content, "\n")
	sections := []Section{}
	currentSection := &Section{}
	currentContent := []string{}
	order := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			// Save previous section if exists
			if currentSection.Title != "" {
				currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections = append(sections, *currentSection)
			}

			// Start new section
			level := strings.Count(strings.Split(line, " ")[0], "#")
			title := strings.TrimSpace(strings.TrimPrefix(line, strings.Repeat("#", level)))
			order++

			currentSection = &Section{
				Title: title,
				Level: level,
				Order: order,
			}
			currentContent = []string{}
		} else if currentSection.Title != "" {
			currentContent = append(currentContent, line)
		}
	}

	// Save last section
	if currentSection.Title != "" {
		currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
		sections = append(sections, *currentSection)
	}

	return sections
}

// ExtractKeywords extracts keywords from content
func ExtractKeywords(content string) []string {
	// Simple keyword extraction - can be enhanced with NLP
	keywords := []string{}
	keywordMap := make(map[string]bool)

	// Common technical keywords to look for
	techWords := []string{
		"API", "REST", "GraphQL", "Database", "PostgreSQL", "MongoDB",
		"Authentication", "Authorization", "Security", "Performance",
		"Scalability", "Docker", "Kubernetes", "CI/CD", "Testing",
		"Frontend", "Backend", "Full-stack", "Microservices", "Cloud",
	}

	content = strings.ToLower(content)
	for _, word := range techWords {
		if strings.Contains(content, strings.ToLower(word)) && !keywordMap[word] {
			keywords = append(keywords, word)
			keywordMap[word] = true
		}
	}

	return keywords
}

// GenerateTaskID generates a task ID with the given prefix and number
func GenerateTaskID(prefix string, number int) string {
	return fmt.Sprintf("%s-%03d", prefix, number)
}

// GenerateSubTaskID generates a sub-task ID
func GenerateSubTaskID(mainTaskID string, number int) string {
	return fmt.Sprintf("ST-%s-%03d", mainTaskID, number)
}

// ToJSON converts entity to JSON
func (p *PRDEntity) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func (t *TRDEntity) ToJSON() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

func (m *MainTask) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func (s *SubTask) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// EstimateComplexity estimates document complexity based on content
func EstimateComplexity(content string, sections []Section) int {
	score := 0

	// Base score on content length
	words := len(strings.Fields(content))
	score += words / 500 // 1 point per 500 words

	// Add points for number of sections
	score += len(sections) * 2

	// Add points for technical keywords
	keywords := ExtractKeywords(content)
	score += len(keywords) * 3

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// FormatTitle formats a title to title case
func FormatTitle(title string) string {
	caser := cases.Title(language.English)
	return caser.String(title)
}
