package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lerian-mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVectorStorage implements VectorStorage for testing
type MockVectorStorage struct {
	chunks      []types.ConversationChunk
	shouldError bool
	collections map[string]bool
}

func NewMockVectorStorage() *MockVectorStorage {
	return &MockVectorStorage{
		chunks:      make([]types.ConversationChunk, 0),
		collections: make(map[string]bool),
	}
}

func (m *MockVectorStorage) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return m.chunks, nil
}

func (m *MockVectorStorage) StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	if m.shouldError {
		return errors.New("mock error")
	}
	m.chunks = append(m.chunks, *chunk)
	return nil
}

func (m *MockVectorStorage) DeleteCollection(ctx context.Context, collection string) error {
	if m.shouldError {
		return errors.New("mock error")
	}
	delete(m.collections, collection)
	return nil
}

func (m *MockVectorStorage) ListCollections(ctx context.Context) ([]string, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	collections := make([]string, 0, len(m.collections))
	for collection := range m.collections {
		collections = append(collections, collection)
	}
	return collections, nil
}

func createTestChunks() []types.ConversationChunk {
	return []types.ConversationChunk{
		{
			ID:         "chunk1",
			SessionID:  "test-session",
			Content:    "Test content 1",
			Summary:    "Summary 1",
			Type:       types.ChunkTypeDiscussion,
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2, 0.3},
			Metadata: types.ChunkMetadata{
				Repository: "test-repo",
				Tags:       []string{"test", "chunk1"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
		{
			ID:         "chunk2",
			SessionID:  "test-session",
			Content:    "Test content 2",
			Summary:    "Summary 2",
			Type:       types.ChunkTypeSolution,
			Timestamp:  time.Now(),
			Embeddings: []float64{0.4, 0.5, 0.6},
			Metadata: types.ChunkMetadata{
				Repository: "test-repo",
				Tags:       []string{"test", "chunk2"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
		{
			ID:         "chunk3",
			SessionID:  "test-session",
			Content:    "Test content 3",
			Summary:    "Summary 3",
			Type:       types.ChunkTypeProblem,
			Timestamp:  time.Now(),
			Embeddings: []float64{0.7, 0.8, 0.9},
			Metadata: types.ChunkMetadata{
				Repository: "another-repo",
				Tags:       []string{"test", "chunk3"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
	}
}

func TestNewBackupManager(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()

	bm := NewBackupManager(storage, tempDir)
	assert.NotNil(t, bm)
	assert.Equal(t, tempDir, bm.GetBackupDir())
	assert.Equal(t, 30, bm.retentionDays) // Default retention
}

func TestBackupManager_CreateBackup(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()
	storage.chunks = createTestChunks()

	bm := NewBackupManager(storage, tempDir)

	tests := []struct {
		name       string
		repository string
		wantChunks int
		wantErr    bool
	}{
		{
			name:       "Backup all chunks",
			repository: "",
			wantChunks: 3,
			wantErr:    false,
		},
		{
			name:       "Backup specific repository",
			repository: "test-repo",
			wantChunks: 2,
			wantErr:    false,
		},
		{
			name:       "Backup non-existent repository",
			repository: "non-existent",
			wantChunks: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := bm.CreateBackup(ctx, tt.repository)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, metadata)
			assert.Equal(t, tt.wantChunks, metadata.ChunkCount)
			assert.Equal(t, tt.repository, metadata.Repository)
			if tt.wantChunks > 0 {
				assert.Greater(t, metadata.Size, int64(0))
			}

			// Verify backup file exists
			backupFile, ok := metadata.Metadata["backup_file"].(string)
			assert.True(t, ok)
			assert.FileExists(t, backupFile)

			// Verify metadata file exists
			metadataFile := backupFile + ".meta.json"
			assert.FileExists(t, metadataFile)

			// Verify metadata content
			// #nosec G304 -- metadataFile is constructed from controlled test data, not user input
			metadataData, err := os.ReadFile(metadataFile)
			require.NoError(t, err)

			var loadedMetadata BackupMetadata
			err = json.Unmarshal(metadataData, &loadedMetadata)
			require.NoError(t, err)
			assert.Equal(t, metadata.ChunkCount, loadedMetadata.ChunkCount)
		})
	}
}

func TestBackupManager_RestoreBackup(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create backup first
	storage1 := NewMockVectorStorage()
	storage1.chunks = createTestChunks()
	bm1 := NewBackupManager(storage1, tempDir)

	metadata, err := bm1.CreateBackup(ctx, "test-repo")
	require.NoError(t, err)

	backupFile, ok := metadata.Metadata["backup_file"].(string)
	require.True(t, ok)

	// Test restore
	storage2 := NewMockVectorStorage()
	bm2 := NewBackupManager(storage2, tempDir)

	err = bm2.RestoreBackup(ctx, backupFile, false)
	require.NoError(t, err)

	// Verify restored chunks
	assert.Len(t, storage2.chunks, 2) // Only test-repo chunks
	for _, chunk := range storage2.chunks {
		assert.Equal(t, "test-repo", chunk.Metadata.Repository)
	}
}

func TestBackupManager_RestoreBackup_Errors(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()
	bm := NewBackupManager(storage, tempDir)

	tests := []struct {
		name    string
		setup   func() string
		wantErr string
	}{
		{
			name: "Non-existent backup",
			setup: func() string {
				return filepath.Join(tempDir, "non-existent.tar.gz")
			},
			wantErr: "failed to read metadata file",
		},
		{
			name: "Invalid metadata",
			setup: func() string {
				backupFile := filepath.Join(tempDir, "invalid.tar.gz")
				metadataFile := backupFile + ".meta.json"
				err := os.WriteFile(metadataFile, []byte("invalid json"), 0o600)
				require.NoError(t, err)
				return backupFile
			},
			wantErr: "failed to unmarshal metadata",
		},
		{
			name: "Missing backup file",
			setup: func() string {
				backupFile := filepath.Join(tempDir, "missing.tar.gz")
				metadata := BackupMetadata{
					ChunkCount: 1,
				}
				metadataData, _ := json.Marshal(metadata)
				metadataFile := backupFile + ".meta.json"
				err := os.WriteFile(metadataFile, metadataData, 0o600)
				require.NoError(t, err)
				return backupFile
			},
			wantErr: "failed to open backup file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupFile := tt.setup()
			err := bm.RestoreBackup(ctx, backupFile, false)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestBackupManager_ListBackups(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()
	storage.chunks = createTestChunks()
	bm := NewBackupManager(storage, tempDir)

	// Create multiple backups
	_, err := bm.CreateBackup(ctx, "repo1")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Ensure different timestamps

	_, err = bm.CreateBackup(ctx, "repo2")
	require.NoError(t, err)

	// List backups
	backups, err := bm.ListBackups()
	require.NoError(t, err)
	assert.Len(t, backups, 2)

	// Verify backup metadata
	for _, backup := range backups {
		assert.NotEmpty(t, backup.Version)
		assert.NotZero(t, backup.CreatedAt)
		assert.NotEmpty(t, backup.Repository)
	}
}

func TestBackupManager_CleanupOldBackups(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()
	storage.chunks = createTestChunks()
	bm := NewBackupManager(storage, tempDir)

	// Set short retention for testing
	bm.SetRetentionDays(0) // Cleanup immediately

	// Create backup
	metadata, err := bm.CreateBackup(ctx, "test")
	require.NoError(t, err)

	backupFile, ok := metadata.Metadata["backup_file"].(string)
	require.True(t, ok)
	metadataFile := backupFile + ".meta.json"

	// Verify files exist
	assert.FileExists(t, backupFile)
	assert.FileExists(t, metadataFile)

	// Modify creation time to make it old
	oldTime := time.Now().AddDate(0, 0, -1)
	metadata.CreatedAt = oldTime
	metadataData, _ := json.Marshal(metadata)
	err = os.WriteFile(metadataFile, metadataData, 0o600)
	require.NoError(t, err)

	// Run cleanup
	err = bm.CleanupOldBackups()
	require.NoError(t, err)

	// Verify files are removed
	assert.NoFileExists(t, backupFile)
	assert.NoFileExists(t, metadataFile)
}

func TestBackupManager_MigrateData(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()

	// Add chunks without summary (simulating v1.0)
	chunks := createTestChunks()
	for i := range chunks {
		chunks[i].Summary = ""
	}
	storage.chunks = chunks

	bm := NewBackupManager(storage, tempDir)

	// Run migration from 1.1 to 2.0
	err := bm.MigrateData(ctx, "1.1", "2.0")
	require.NoError(t, err)

	// Verify migration applied
	// Note: In real test, we'd verify the stored chunks have summaries
	// For now, just verify no error and backup was created
	backups, err := bm.ListBackups()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(backups), 1)

	// Find pre-migration backup
	found := false
	for _, backup := range backups {
		if backup.Repository == "pre_migration" {
			found = true
			break
		}
	}
	assert.True(t, found, "Pre-migration backup should exist")
}

func TestBackupManager_VerifyIntegrity(t *testing.T) {
	ctx := context.Background()
	storage := NewMockVectorStorage()
	bm := NewBackupManager(storage, t.TempDir())

	tests := []struct {
		name         string
		chunks       []types.ConversationChunk
		wantErr      bool
		errMsg       string
		skipValidate bool // Skip chunk.Validate() check
	}{
		{
			name:    "Valid chunks",
			chunks:  createTestChunks(),
			wantErr: false,
		},
		{
			name: "Chunk with empty ID",
			chunks: []types.ConversationChunk{
				{
					ID:         "",
					SessionID:  "test",
					Content:    "Test",
					Type:       types.ChunkTypeDiscussion,
					Timestamp:  time.Now(),
					Embeddings: []float64{0.1},
				},
			},
			wantErr: true,
			errMsg:  "ID cannot be empty",
		},
		{
			name: "Chunk with zero timestamp",
			chunks: []types.ConversationChunk{
				{
					ID:         "test",
					SessionID:  "test",
					Content:    "Test",
					Type:       types.ChunkTypeDiscussion,
					Timestamp:  time.Time{},
					Embeddings: []float64{0.1},
				},
			},
			wantErr: true,
			errMsg:  "timestamp cannot be zero",
		},
		{
			name: "Chunk with no embeddings",
			chunks: []types.ConversationChunk{
				{
					ID:         "test",
					SessionID:  "test",
					Content:    "Test",
					Type:       types.ChunkTypeDiscussion,
					Timestamp:  time.Now(),
					Embeddings: []float64{},
					Metadata: types.ChunkMetadata{
						Outcome:    types.OutcomeSuccess,
						Difficulty: types.DifficultySimple,
					},
				},
			},
			wantErr: true,
			errMsg:  "has no embeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.chunks = tt.chunks
			err := bm.VerifyIntegrity(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackupManager_CompressData(t *testing.T) {
	ctx := context.Background()
	storage := NewMockVectorStorage()

	// Create chunks with compressible content
	chunks := []types.ConversationChunk{
		{
			ID:         "chunk1",
			SessionID:  "test",
			Content:    "This    has    lots    of    spaces",
			Type:       types.ChunkTypeDiscussion,
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1},
		},
		{
			ID:         "chunk2",
			SessionID:  "test",
			Content:    "\n\n\nMultiple\n\n\nNewlines\n\n\n",
			Type:       types.ChunkTypeDiscussion,
			Timestamp:  time.Now(),
			Embeddings: []float64{0.2},
		},
	}
	storage.chunks = chunks

	bm := NewBackupManager(storage, t.TempDir())

	// Run compression
	err := bm.CompressData(ctx)
	require.NoError(t, err)

	// Note: The actual compression in the code is simplistic
	// In a real implementation, we'd verify the content is compressed
}

func TestBackupManager_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()
	bm := NewBackupManager(storage, tempDir)

	t.Run("Create backup with storage error", func(t *testing.T) {
		storage.shouldError = true
		_, err := bm.CreateBackup(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve chunks")
		storage.shouldError = false
	})

	t.Run("Restore with storage error", func(t *testing.T) {
		// Create valid backup first
		storage.chunks = createTestChunks()
		metadata, err := bm.CreateBackup(ctx, "")
		require.NoError(t, err)

		backupFile, _ := metadata.Metadata["backup_file"].(string)

		// Make storage error on restore
		storage.shouldError = true
		err = bm.RestoreBackup(ctx, backupFile, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store chunk")
		storage.shouldError = false
	})

	t.Run("Verify integrity with storage error", func(t *testing.T) {
		storage.shouldError = true
		err := bm.VerifyIntegrity(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve chunks")
		storage.shouldError = false
	})
}

func TestBackupManager_PathSecurity(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage := NewMockVectorStorage()
	bm := NewBackupManager(storage, tempDir)

	// Test path traversal protection in CreateBackup
	maliciousRepo := "../../../etc/passwd"
	metadata, err := bm.CreateBackup(ctx, maliciousRepo)
	require.NoError(t, err)

	// Verify the path was cleaned
	backupFile, _ := metadata.Metadata["backup_file"].(string)
	assert.Contains(t, backupFile, filepath.Base(maliciousRepo))
	assert.NotContains(t, backupFile, "..")
}

// Benchmark tests
func BenchmarkCreateBackup(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()
	storage := NewMockVectorStorage()

	// Create many chunks
	for i := 0; i < 1000; i++ {
		chunk := types.ConversationChunk{
			ID:         fmt.Sprintf("chunk%d", i),
			SessionID:  "bench-session",
			Content:    fmt.Sprintf("Content for chunk %d", i),
			Summary:    fmt.Sprintf("Summary %d", i),
			Type:       types.ChunkTypeDiscussion,
			Timestamp:  time.Now(),
			Embeddings: []float64{float64(i) * 0.001},
			Metadata: types.ChunkMetadata{
				Repository: "bench-repo",
			},
		}
		storage.chunks = append(storage.chunks, chunk)
	}

	bm := NewBackupManager(storage, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.CreateBackup(ctx, "bench-repo")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRestoreBackup(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()
	storage := NewMockVectorStorage()
	storage.chunks = createTestChunks()

	bm := NewBackupManager(storage, tempDir)

	// Create backup once
	metadata, err := bm.CreateBackup(ctx, "")
	if err != nil {
		b.Fatal(err)
	}
	backupFile, _ := metadata.Metadata["backup_file"].(string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear storage
		storage.chunks = make([]types.ConversationChunk, 0)

		err := bm.RestoreBackup(ctx, backupFile, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}
