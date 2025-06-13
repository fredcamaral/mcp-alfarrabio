// Package storage provides task data access and persistence functionality.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/internal/tasks"
	"lerian-mcp-memory/pkg/types"

	"github.com/google/uuid"
)

// TaskRepository implements task data access using SQL database
type TaskRepository struct {
	db     *sql.DB
	filter *tasks.FilterManager
}

// scanTaskFromRows scans a single task from database rows to reduce duplicate code
func (tr *TaskRepository) scanTaskFromRows(rows *sql.Rows) (*types.Task, error) {
	var task types.Task
	var acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, metadataJSON []byte
	var repository, sessionID sql.NullString
	var content, complexityStr, typeStr, priorityStr, statusStr string

	err := rows.Scan(
		&task.ID, &task.Title, &task.Description, &content, &typeStr, &priorityStr, &statusStr, &task.Assignee,
		&repository, &sessionID, &task.Timestamps.Created, &task.Timestamps.Updated,
		&task.Timestamps.Started, &task.Timestamps.Completed,
		&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
		&complexityStr, &task.Complexity.Score, &task.QualityScore.OverallScore,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Convert enum strings to typed values
	task.Type = types.TaskType(typeStr)
	task.Priority = types.TaskPriority(priorityStr)
	task.Status = types.TaskStatus(statusStr)
	task.Complexity.Level = types.ComplexityLevel(complexityStr)

	// Deserialize JSONB fields
	if len(acceptanceCriteriaJSON) > 0 {
		if err := json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria); err != nil {
			return nil, fmt.Errorf("failed to unmarshal acceptance criteria: %w", err)
		}
	}
	if len(dependenciesJSON) > 0 {
		if err := json.Unmarshal(dependenciesJSON, &task.Dependencies); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dependencies: %w", err)
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &task.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &task.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Initialize metadata if nil
	if task.Metadata.ExtendedData == nil {
		task.Metadata.ExtendedData = make(map[string]interface{})
	}

	// Store repository in metadata for compatibility
	if repository.Valid {
		task.Metadata.ExtendedData["repository"] = repository.String
	}
	if sessionID.Valid {
		task.Metadata.ExtendedData["session_id"] = sessionID.String
	}

	return &task, nil
}

// generateUUID generates a new UUID string
func generateUUID() string {
	return uuid.New().String()
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{
		db:     db,
		filter: tasks.NewFilterManager(),
	}
}

// Create creates a new task in the database
func (tr *TaskRepository) Create(ctx context.Context, task *types.Task) error {
	// Use UUID generation if ID is empty
	taskID := task.ID
	if taskID == "" {
		taskID = generateUUID()
	}

	query := `
		INSERT INTO tasks (
			id, title, description, content, type, priority, status, assignee, 
			repository, session_id, created_at, updated_at, 
			acceptance_criteria, dependencies, tags, estimated_hours,
			complexity, complexity_score, quality_score,
			metadata
		) VALUES (
			$1, $2, $3, $4, $5::task_type, $6::task_priority, $7::task_status, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17::task_complexity, $18, $19,
			$20
		)`

	// Serialize JSONB fields
	acceptanceCriteriaJSON, err := json.Marshal(task.AcceptanceCriteria)
	if err != nil {
		return fmt.Errorf("failed to marshal acceptance criteria: %w", err)
	}
	dependenciesJSON, err := json.Marshal(task.Dependencies)
	if err != nil {
		return fmt.Errorf("failed to marshal dependencies: %w", err)
	}
	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}
	metadataJSON, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Extract repository from metadata or set default
	repository := "default"
	if task.Metadata.ExtendedData != nil {
		if repo, ok := task.Metadata.ExtendedData["repository"].(string); ok && repo != "" {
			repository = repo
		}
	}

	// Set content to description as that's what the database expects
	content := task.Description
	if content == "" {
		content = task.Title
	}

	_, err = tr.db.ExecContext(ctx, query,
		taskID, task.Title, task.Description, content,
		string(task.Type), string(task.Priority), string(task.Status), task.Assignee,
		repository, nil, // session_id can be null
		task.Timestamps.Created, task.Timestamps.Updated,
		acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, task.EstimatedEffort.Hours,
		string(task.Complexity.Level), task.Complexity.Score, task.QualityScore.OverallScore,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Update the task ID in case it was generated
	task.ID = taskID
	return nil
}

// GetByID retrieves a task by ID
func (tr *TaskRepository) GetByID(ctx context.Context, id string) (*types.Task, error) {
	query := `
		SELECT id, title, description, content, type, priority, status, assignee,
			   repository, session_id, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity, complexity_score, quality_score,
			   metadata
		FROM tasks 
		WHERE id = $1 AND deleted_at IS NULL`

	var task types.Task
	var acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, metadataJSON []byte
	var repository, sessionID sql.NullString
	var content, complexityStr, typeStr, priorityStr, statusStr string

	err := tr.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.Title, &task.Description, &content, &typeStr, &priorityStr, &statusStr, &task.Assignee,
		&repository, &sessionID, &task.Timestamps.Created, &task.Timestamps.Updated,
		&task.Timestamps.Started, &task.Timestamps.Completed,
		&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
		&complexityStr, &task.Complexity.Score, &task.QualityScore.OverallScore,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Convert enum strings to typed values
	task.Type = types.TaskType(typeStr)
	task.Priority = types.TaskPriority(priorityStr)
	task.Status = types.TaskStatus(statusStr)
	task.Complexity.Level = types.ComplexityLevel(complexityStr)

	// Deserialize JSONB fields
	if len(acceptanceCriteriaJSON) > 0 {
		if err := json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria); err != nil {
			return nil, fmt.Errorf("failed to unmarshal acceptance criteria: %w", err)
		}
	}
	if len(dependenciesJSON) > 0 {
		if err := json.Unmarshal(dependenciesJSON, &task.Dependencies); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dependencies: %w", err)
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &task.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &task.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Initialize metadata if nil
	if task.Metadata.ExtendedData == nil {
		task.Metadata.ExtendedData = make(map[string]interface{})
	}

	// Store repository in metadata for compatibility
	if repository.Valid {
		task.Metadata.ExtendedData["repository"] = repository.String
	}
	if sessionID.Valid {
		task.Metadata.ExtendedData["session_id"] = sessionID.String
	}

	return &task, nil
}

// Update updates an existing task
func (tr *TaskRepository) Update(ctx context.Context, task *types.Task) error {
	query := `
		UPDATE tasks SET 
			title = $2, description = $3, content = $4, type = $5::task_type, priority = $6::task_priority, 
			status = $7::task_status, assignee = $8, due_date = $9, updated_at = $10, 
			started_at = $11, completed_at = $12, acceptance_criteria = $13, dependencies = $14, 
			tags = $15, estimated_hours = $16, complexity = $17::task_complexity, 
			complexity_score = $18, quality_score = $19, metadata = $20
		WHERE id = $1 AND deleted_at IS NULL`

	// Serialize JSONB fields
	acceptanceCriteriaJSON, err := json.Marshal(task.AcceptanceCriteria)
	if err != nil {
		return fmt.Errorf("failed to marshal acceptance criteria: %w", err)
	}
	dependenciesJSON, err := json.Marshal(task.Dependencies)
	if err != nil {
		return fmt.Errorf("failed to marshal dependencies: %w", err)
	}
	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}
	metadataJSON, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Set content to description as that's what the database expects
	content := task.Description
	if content == "" {
		content = task.Title
	}

	result, err := tr.db.ExecContext(ctx, query,
		task.ID, task.Title, task.Description, content,
		string(task.Type), string(task.Priority), string(task.Status),
		task.Assignee, task.DueDate, task.Timestamps.Updated, task.Timestamps.Started, task.Timestamps.Completed,
		acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, task.EstimatedEffort.Hours,
		string(task.Complexity.Level), task.Complexity.Score, task.QualityScore.OverallScore,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	return nil
}

// Delete deletes a task by ID (soft delete)
func (tr *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE tasks SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`

	result, err := tr.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

// List retrieves tasks with filtering and pagination
func (tr *TaskRepository) List(ctx context.Context, filters *tasks.TaskFilters) ([]types.Task, error) {
	// Simple query without complex filtering for now - can be enhanced later
	query := `
		SELECT id, title, description, content, type, priority, status, assignee,
			   repository, session_id, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity, complexity_score, quality_score,
			   metadata
		FROM tasks
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`

	// Add pagination if specified
	if filters != nil {
		if filters.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", filters.Limit)
		}
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filters.Offset)
		}
	}

	rows, err := tr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var taskList []types.Task
	for rows.Next() {
		var task types.Task
		var acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, metadataJSON []byte
		var repository, sessionID sql.NullString
		var content, complexityStr, typeStr, priorityStr, statusStr string

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &content, &typeStr, &priorityStr, &statusStr, &task.Assignee,
			&repository, &sessionID, &task.Timestamps.Created, &task.Timestamps.Updated,
			&task.Timestamps.Started, &task.Timestamps.Completed,
			&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
			&complexityStr, &task.Complexity.Score, &task.QualityScore.OverallScore,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		// Convert enum strings to typed values
		task.Type = types.TaskType(typeStr)
		task.Priority = types.TaskPriority(priorityStr)
		task.Status = types.TaskStatus(statusStr)
		task.Complexity.Level = types.ComplexityLevel(complexityStr)

		// Deserialize JSONB fields
		if len(acceptanceCriteriaJSON) > 0 {
			_ = json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria)
		}
		if len(dependenciesJSON) > 0 {
			_ = json.Unmarshal(dependenciesJSON, &task.Dependencies)
		}
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &task.Tags)
		}
		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &task.Metadata)
		}

		// Initialize metadata if nil
		if task.Metadata.ExtendedData == nil {
			task.Metadata.ExtendedData = make(map[string]interface{})
		}

		// Store repository in metadata for compatibility
		if repository.Valid {
			task.Metadata.ExtendedData["repository"] = repository.String
		}
		if sessionID.Valid {
			task.Metadata.ExtendedData["session_id"] = sessionID.String
		}

		taskList = append(taskList, task)
	}

	return taskList, rows.Err()
}

// Search performs full-text search on tasks
func (tr *TaskRepository) Search(ctx context.Context, query *tasks.SearchQuery) (*tasks.SearchResults, error) {
	startTime := time.Now()

	// Build search query with full-text search capabilities
	searchQuery := `
		SELECT id, title, description, content, type, priority, status, assignee,
			   repository, session_id, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity, complexity_score, quality_score,
			   metadata,
			   ts_rank(search_vector, plainto_tsquery($1)) as rank
		FROM tasks
		WHERE search_vector @@ plainto_tsquery($1) AND deleted_at IS NULL`

	// Add additional filters
	whereClause, args := tr.filter.BuildWhereClause(&query.Filters)
	if whereClause != "" {
		// Remove "WHERE" since we already have it
		whereClause = strings.Replace(whereClause, "WHERE", "AND", 1)
		searchQuery += " " + whereClause
	}

	// Combine query argument with filter arguments
	allArgs := append([]interface{}{query.Query}, args...)

	// Add ordering and limits
	searchQuery += " ORDER BY rank DESC, created_at DESC"
	if query.Options.MaxResults > 0 {
		searchQuery += fmt.Sprintf(" LIMIT %d", query.Options.MaxResults)
	}

	rows, err := tr.db.QueryContext(ctx, searchQuery, allArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Ignore close errors in defer
		_ = rows.Close()
	}()

	var taskList []types.Task
	for rows.Next() {
		var task types.Task
		var acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, metadataJSON []byte
		var repository, sessionID sql.NullString
		var content, complexityStr, typeStr, priorityStr, statusStr string
		var rank float64

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &content, &typeStr, &priorityStr, &statusStr, &task.Assignee,
			&repository, &sessionID, &task.Timestamps.Created, &task.Timestamps.Updated,
			&task.Timestamps.Started, &task.Timestamps.Completed,
			&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
			&complexityStr, &task.Complexity.Score, &task.QualityScore.OverallScore,
			&metadataJSON, &rank,
		)

		if err != nil {
			return nil, err
		}

		// Convert enum strings to typed values
		task.Type = types.TaskType(typeStr)
		task.Priority = types.TaskPriority(priorityStr)
		task.Status = types.TaskStatus(statusStr)
		task.Complexity.Level = types.ComplexityLevel(complexityStr)

		// Deserialize JSONB fields
		if len(acceptanceCriteriaJSON) > 0 {
			_ = json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria)
		}
		if len(dependenciesJSON) > 0 {
			_ = json.Unmarshal(dependenciesJSON, &task.Dependencies)
		}
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &task.Tags)
		}
		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &task.Metadata)
		}

		// Initialize metadata if nil
		if task.Metadata.ExtendedData == nil {
			task.Metadata.ExtendedData = make(map[string]interface{})
		}

		// Store repository in metadata for compatibility
		if repository.Valid {
			task.Metadata.ExtendedData["repository"] = repository.String
		}
		if sessionID.Valid {
			task.Metadata.ExtendedData["session_id"] = sessionID.String
		}

		taskList = append(taskList, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Create SearchResults struct
	results := &tasks.SearchResults{
		Tasks:        taskList,
		TotalResults: len(taskList),
		SearchTime:   time.Since(startTime),
		Query:        query.Query,
	}

	// Add highlights if requested
	if query.Options.HighlightMatches {
		results.Highlights = tr.generateHighlights(taskList, query.Query)
	}

	return results, nil
}

// BatchUpdate performs batch updates on multiple tasks
func (tr *TaskRepository) BatchUpdate(ctx context.Context, updates []tasks.BatchUpdate) error {
	tx, err := tr.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		// Ignore rollback errors in defer - transaction might be committed
		_ = tx.Rollback()
	}()

	for _, update := range updates {
		query := "UPDATE tasks SET updated_at = $1"
		args := []interface{}{update.UpdatedAt}
		argIndex := 2

		// Build dynamic update query
		if update.Status != nil {
			query += fmt.Sprintf(", status = $%d", argIndex)
			args = append(args, *update.Status)
			argIndex++
		}
		if update.Priority != nil {
			query += fmt.Sprintf(", priority = $%d", argIndex)
			args = append(args, *update.Priority)
			argIndex++
		}
		if update.Assignee != nil {
			query += fmt.Sprintf(", assignee = $%d", argIndex)
			args = append(args, *update.Assignee)
			argIndex++
		}
		if update.DueDate != nil {
			query += fmt.Sprintf(", due_date = $%d", argIndex)
			args = append(args, *update.DueDate)
			argIndex++
		}
		if len(update.Tags) > 0 {
			tagsJSON, _ := json.Marshal(update.Tags)
			query += fmt.Sprintf(", tags = $%d", argIndex)
			args = append(args, tagsJSON)
			argIndex++
		}

		query += fmt.Sprintf(" WHERE id = $%d", argIndex)
		args = append(args, update.TaskID)

		_, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByIDs retrieves multiple tasks by their IDs
func (tr *TaskRepository) GetByIDs(ctx context.Context, ids []string) ([]types.Task, error) {
	if len(ids) == 0 {
		return []types.Task{}, nil
	}

	// Build query with IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// #nosec G201 - This is safe as it uses parameterized placeholders ($1, $2, etc.)
	query := fmt.Sprintf(`
		SELECT id, title, description, content, type, priority, status, assignee,
			   repository, session_id, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity, complexity_score, quality_score,
			   metadata
		FROM tasks 
		WHERE id IN (%s) AND deleted_at IS NULL`, strings.Join(placeholders, ", "))

	rows, err := tr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Ignore close errors in defer
		_ = rows.Close()
	}()

	var taskList []types.Task
	for rows.Next() {
		task, err := tr.scanTaskFromRows(rows)
		if err != nil {
			return nil, err
		}
		taskList = append(taskList, *task)
	}

	return taskList, rows.Err()
}

// generateHighlights generates search result highlights
func (tr *TaskRepository) generateHighlights(taskList []types.Task, query string) map[string][]string {
	highlights := make(map[string][]string)

	for i := range taskList {
		task := &taskList[i]
		taskHighlights := make([]string, 0)

		// Simple highlighting logic - could be enhanced
		if strings.Contains(strings.ToLower(task.Title), strings.ToLower(query)) {
			taskHighlights = append(taskHighlights, "Title: "+task.Title)
		}
		if strings.Contains(strings.ToLower(task.Description), strings.ToLower(query)) {
			// Truncate description for highlight
			desc := task.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			taskHighlights = append(taskHighlights, "Description: "+desc)
		}

		if len(taskHighlights) > 0 {
			highlights[task.ID] = taskHighlights
		}
	}

	return highlights
}
