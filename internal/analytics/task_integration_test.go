package analytics

import (
	"testing"
	"time"

	"lerian-mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
)

// TestTaskChunkIntegration verifies that task chunks are properly handled in analytics
func TestTaskChunkIntegration(t *testing.T) {
	analytics := NewMemoryAnalytics(nil)

	tests := []struct {
		name          string
		chunk         types.ConversationChunk
		expectedScore float64
		minScore      float64
	}{
		{
			name: "high_priority_completed_task",
			chunk: types.ConversationChunk{
				ID:        "task-1",
				Type:      types.ChunkTypeTask,
				Content:   "Complete critical security audit",
				Timestamp: time.Now().Add(-1 * time.Hour), // Recent
				Metadata: types.ChunkMetadata{
					Repository:   "security-project",
					TaskStatus:   taskStatusPtr(types.TaskStatusCompleted),
					TaskPriority: stringPtr("high"),
				},
			},
			minScore: 0.8, // Should be high due to completion + high priority + recency
		},
		{
			name: "medium_priority_in_progress_task",
			chunk: types.ConversationChunk{
				ID:        "task-2",
				Type:      types.ChunkTypeTask,
				Content:   "Implement user authentication",
				Timestamp: time.Now().Add(-2 * time.Hour),
				Metadata: types.ChunkMetadata{
					Repository:   "auth-project",
					TaskStatus:   taskStatusPtr(types.TaskStatusInProgress),
					TaskPriority: stringPtr("medium"),
				},
			},
			minScore: 0.4, // Should be moderate
		},
		{
			name: "task_update_with_progress",
			chunk: types.ConversationChunk{
				ID:        "task-update-1",
				Type:      types.ChunkTypeTaskUpdate,
				Content:   "Updated task progress - 90% complete",
				Timestamp: time.Now().Add(-30 * time.Minute),
				Metadata: types.ChunkMetadata{
					Repository: "project-x",
				},
			},
			minScore: 0.5, // Updates are valuable
		},
		{
			name: "high_progress_tracking",
			chunk: types.ConversationChunk{
				ID:        "progress-1",
				Type:      types.ChunkTypeTaskProgress,
				Content:   "Task is 85% complete with all tests passing",
				Timestamp: time.Now().Add(-1 * time.Hour),
				Metadata: types.ChunkMetadata{
					Repository:   "test-project",
					TaskProgress: intPtr(85),
				},
			},
			minScore: 0.6, // High progress gets bonus
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analytics.CalculateEffectivenessScore(&tt.chunk)

			assert.GreaterOrEqual(t, score, tt.minScore,
				"Effectiveness score for %s should be at least %.2f, got %.2f",
				tt.name, tt.minScore, score)

			assert.LessOrEqual(t, score, 1.0,
				"Effectiveness score should not exceed 1.0, got %.2f", score)
		})
	}
}

// Helper functions for pointer values
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func taskStatusPtr(ts types.TaskStatus) *types.TaskStatus {
	return &ts
}
