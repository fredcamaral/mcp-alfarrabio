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
	var repository, branch sql.NullString

	err := rows.Scan(
		&task.ID, &task.Title, &task.Description, &task.Type, &task.Priority, &task.Status, &task.Assignee,
		&task.SourcePRDID, &task.DueDate, &task.Timestamps.Created, &task.Timestamps.Updated,
		&task.Timestamps.Started, &task.Timestamps.Completed,
		&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
		&task.Complexity.Level, &task.Complexity.Score, &task.QualityScore.OverallScore,
		&metadataJSON, &repository, &branch,
	)

	if err != nil {
		return nil, err
	}

	// Deserialize complex fields
	if err := json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria); err != nil {
		return nil, fmt.Errorf("failed to unmarshal acceptance criteria: %w", err)
	}
	if err := json.Unmarshal(dependenciesJSON, &task.Dependencies); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependencies: %w", err)
	}
	if err := json.Unmarshal(tagsJSON, &task.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &task.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Handle nullable fields
	if task.Metadata.ExtendedData == nil {
		task.Metadata.ExtendedData = make(map[string]interface{})
	}
	if repository.Valid {
		task.Metadata.ExtendedData["repository"] = repository.String
	}
	if branch.Valid {
		task.Metadata.ExtendedData["branch"] = branch.String
	}

	return &task, nil
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
	query := `
		INSERT INTO tasks (
			id, title, description, type, priority, status, assignee, 
			source_prd_id, due_date, created_at, updated_at, 
			acceptance_criteria, dependencies, tags, estimated_hours,
			complexity_level, complexity_score, quality_score,
			metadata, repository, branch
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, 
			$8, $9, $10, $11, 
			$12, $13, $14, $15,
			$16, $17, $18,
			$19, $20, $21
		)`

	// Serialize complex fields
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

	_, err = tr.db.ExecContext(ctx, query,
		task.ID, task.Title, task.Description, task.Type, task.Priority, task.Status, task.Assignee,
		task.SourcePRDID, task.DueDate, task.Timestamps.Created, task.Timestamps.Updated,
		acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, task.EstimatedEffort.Hours,
		task.Complexity.Level, task.Complexity.Score, task.QualityScore.OverallScore,
		metadataJSON, task.Metadata.ExtendedData["repository"], task.Metadata.ExtendedData["branch"],
	)

	return err
}

// GetByID retrieves a task by ID
func (tr *TaskRepository) GetByID(ctx context.Context, id string) (*types.Task, error) {
	query := `
		SELECT id, title, description, type, priority, status, assignee,
			   source_prd_id, due_date, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity_level, complexity_score, quality_score,
			   metadata, repository, branch
		FROM tasks 
		WHERE id = $1`

	var task types.Task
	var acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, metadataJSON []byte
	var repository, branch sql.NullString

	err := tr.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Type, &task.Priority, &task.Status, &task.Assignee,
		&task.SourcePRDID, &task.DueDate, &task.Timestamps.Created, &task.Timestamps.Updated,
		&task.Timestamps.Started, &task.Timestamps.Completed,
		&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
		&task.Complexity.Level, &task.Complexity.Score, &task.QualityScore.OverallScore,
		&metadataJSON, &repository, &branch,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, err
	}

	// Deserialize complex fields
	_ = json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria)
	_ = json.Unmarshal(dependenciesJSON, &task.Dependencies)
	_ = json.Unmarshal(tagsJSON, &task.Tags)
	_ = json.Unmarshal(metadataJSON, &task.Metadata)

	// Handle nullable fields
	if task.Metadata.ExtendedData == nil {
		task.Metadata.ExtendedData = make(map[string]interface{})
	}
	if repository.Valid {
		task.Metadata.ExtendedData["repository"] = repository.String
	}
	if branch.Valid {
		task.Metadata.ExtendedData["branch"] = branch.String
	}

	return &task, nil
}

// Update updates an existing task
func (tr *TaskRepository) Update(ctx context.Context, task *types.Task) error {
	query := `
		UPDATE tasks SET 
			title = $2, description = $3, type = $4, priority = $5, status = $6, 
			assignee = $7, due_date = $8, updated_at = $9, started_at = $10, completed_at = $11,
			acceptance_criteria = $12, dependencies = $13, tags = $14, estimated_hours = $15,
			complexity_level = $16, complexity_score = $17, quality_score = $18,
			metadata = $19, repository = $20, branch = $21
		WHERE id = $1`

	// Serialize complex fields
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

	var repository, branch interface{}
	if task.Metadata.ExtendedData != nil {
		repository = task.Metadata.ExtendedData["repository"]
		branch = task.Metadata.ExtendedData["branch"]
	}

	result, err := tr.db.ExecContext(ctx, query,
		task.ID, task.Title, task.Description, task.Type, task.Priority, task.Status,
		task.Assignee, task.DueDate, task.Timestamps.Updated, task.Timestamps.Started, task.Timestamps.Completed,
		acceptanceCriteriaJSON, dependenciesJSON, tagsJSON, task.EstimatedEffort.Hours,
		task.Complexity.Level, task.Complexity.Score, task.QualityScore.OverallScore,
		metadataJSON, repository, branch,
	)

	if err != nil {
		return err
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

// Delete deletes a task by ID
func (tr *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := tr.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
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
	// Build query with filters
	baseQuery := `
		SELECT id, title, description, type, priority, status, assignee,
			   source_prd_id, due_date, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity_level, complexity_score, quality_score,
			   metadata, repository, branch
		FROM tasks`

	whereClause, args := tr.filter.BuildWhereClause(filters)
	orderClause := tr.filter.BuildOrderClause(filters.SortBy)

	query := fmt.Sprintf("%s %s %s", baseQuery, whereClause, orderClause)

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

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

// Search performs full-text search on tasks
func (tr *TaskRepository) Search(ctx context.Context, query *tasks.SearchQuery) (interface{}, error) {
	startTime := time.Now()

	// Build search query with full-text search capabilities
	searchQuery := `
		SELECT id, title, description, type, priority, status, assignee,
			   source_prd_id, due_date, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity_level, complexity_score, quality_score,
			   metadata, repository, branch,
			   ts_rank(search_vector, plainto_tsquery($1)) as rank
		FROM tasks
		WHERE search_vector @@ plainto_tsquery($1)`

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
		var repository, branch sql.NullString
		var rank float64

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Type, &task.Priority, &task.Status, &task.Assignee,
			&task.SourcePRDID, &task.DueDate, &task.Timestamps.Created, &task.Timestamps.Updated,
			&task.Timestamps.Started, &task.Timestamps.Completed,
			&acceptanceCriteriaJSON, &dependenciesJSON, &tagsJSON, &task.EstimatedEffort.Hours,
			&task.Complexity.Level, &task.Complexity.Score, &task.QualityScore.OverallScore,
			&metadataJSON, &repository, &branch, &rank,
		)

		if err != nil {
			return nil, err
		}

		// Deserialize complex fields
		_ = json.Unmarshal(acceptanceCriteriaJSON, &task.AcceptanceCriteria)
		_ = json.Unmarshal(dependenciesJSON, &task.Dependencies)
		_ = json.Unmarshal(tagsJSON, &task.Tags)
		_ = json.Unmarshal(metadataJSON, &task.Metadata)

		// Handle nullable fields
		if task.Metadata.ExtendedData == nil {
			task.Metadata.ExtendedData = make(map[string]interface{})
		}
		if repository.Valid {
			task.Metadata.ExtendedData["repository"] = repository.String
		}
		if branch.Valid {
			task.Metadata.ExtendedData["branch"] = branch.String
		}

		taskList = append(taskList, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := map[string]interface{}{
		"tasks":         taskList,
		"total_results": len(taskList),
		"search_time":   time.Since(startTime),
		"query":         query.Query,
	}

	// Add highlights if requested
	if query.Options.HighlightMatches {
		results["highlights"] = tr.generateHighlights(taskList, query.Query)
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
		SELECT id, title, description, type, priority, status, assignee,
			   source_prd_id, due_date, created_at, updated_at, started_at, completed_at,
			   acceptance_criteria, dependencies, tags, estimated_hours,
			   complexity_level, complexity_score, quality_score,
			   metadata, repository, branch
		FROM tasks 
		WHERE id IN (%s)`, strings.Join(placeholders, ", "))

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
