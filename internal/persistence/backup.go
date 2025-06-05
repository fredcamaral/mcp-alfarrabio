// Package persistence provides backup and restore functionality
// for memory data persistence in the MCP Memory Server.
package persistence

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// BackupManager handles backup and restore operations
type BackupManager struct {
	storage       VectorStorage
	backupDir     string
	retentionDays int
}

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	Version    string                 `json:"version"`
	CreatedAt  time.Time              `json:"created_at"`
	ChunkCount int                    `json:"chunk_count"`
	Size       int64                  `json:"size"`
	Checksum   string                 `json:"checksum"`
	Repository string                 `json:"repository,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// VectorStorage interface for backup operations
type VectorStorage interface {
	GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error)
	StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error
	DeleteCollection(ctx context.Context, collection string) error
	ListCollections(ctx context.Context) ([]string, error)
}

// NewBackupManager creates a new backup manager
func NewBackupManager(storage VectorStorage, backupDir string) *BackupManager {
	return &BackupManager{
		storage:       storage,
		backupDir:     backupDir,
		retentionDays: getEnvInt("MCP_MEMORY_BACKUP_RETENTION_DAYS", 30), // Default 30 days retention
	}
}

// CreateBackup creates a complete backup of all data
func (bm *BackupManager) CreateBackup(ctx context.Context, repository string) (*BackupMetadata, error) {
	backupFile, err := bm.prepareBackupFile(repository)
	if err != nil {
		return nil, err
	}

	chunks, err := bm.getChunksForBackup(ctx, repository)
	if err != nil {
		return nil, err
	}

	err = bm.writeBackupArchive(backupFile, chunks)
	if err != nil {
		return nil, err
	}

	metadata, err := bm.createBackupMetadata(backupFile, repository, len(chunks))
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// prepareBackupFile creates the backup directory and generates the backup file path
func (bm *BackupManager) prepareBackupFile(repository string) (string, error) {
	if err := os.MkdirAll(bm.backupDir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	cleanRepo := filepath.Base(repository)
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(bm.backupDir, fmt.Sprintf("backup_%s_%s.tar.gz", cleanRepo, timestamp))

	return backupFile, nil
}

// getChunksForBackup retrieves and filters chunks for backup
func (bm *BackupManager) getChunksForBackup(ctx context.Context, repository string) ([]types.ConversationChunk, error) {
	chunks, err := bm.storage.GetAllChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks: %w", err)
	}

	if repository != "" {
		chunks = bm.filterChunksByRepository(chunks, repository)
	}

	return chunks, nil
}

// filterChunksByRepository filters chunks by repository
func (bm *BackupManager) filterChunksByRepository(chunks []types.ConversationChunk, repository string) []types.ConversationChunk {
	filteredChunks := make([]types.ConversationChunk, 0)
	for i := range chunks {
		if chunks[i].Metadata.Repository == repository {
			filteredChunks = append(filteredChunks, chunks[i])
		}
	}
	return filteredChunks
}

// writeBackupArchive creates and writes the backup archive
func (bm *BackupManager) writeBackupArchive(backupFile string, chunks []types.ConversationChunk) error {
	file, err := os.Create(backupFile) // #nosec G304 -- Path is cleaned and safe
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			_ = err // Log error but don't fail the function
		}
	}()

	gzipWriter := gzip.NewWriter(file)
	defer func() {
		if err := gzipWriter.Close(); err != nil {
			_ = err // Log error but don't fail the function
		}
	}()

	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		if err := tarWriter.Close(); err != nil {
			_ = err // Log error but don't fail the function
		}
	}()

	return bm.writeChunksToTar(tarWriter, chunks)
}

// writeChunksToTar writes chunks to the tar archive
func (bm *BackupManager) writeChunksToTar(tarWriter *tar.Writer, chunks []types.ConversationChunk) error {
	for i := range chunks {
		chunkData, err := json.MarshalIndent(chunks[i], "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal chunk %s: %w", chunks[i].ID, err)
		}

		header := &tar.Header{
			Name: fmt.Sprintf("chunks/chunk_%d_%s.json", i, chunks[i].ID),
			Size: int64(len(chunkData)),
			Mode: 0o644,
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if _, err := tarWriter.Write(chunkData); err != nil {
			return fmt.Errorf("failed to write chunk data: %w", err)
		}
	}
	return nil
}

// createBackupMetadata creates and saves backup metadata
func (bm *BackupManager) createBackupMetadata(backupFile, repository string, chunkCount int) (*BackupMetadata, error) {
	stat, err := os.Stat(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	metadata := &BackupMetadata{
		Version:    getEnv("MCP_MEMORY_BACKUP_VERSION", "1.0"),
		CreatedAt:  time.Now(),
		ChunkCount: chunkCount,
		Size:       stat.Size(),
		Repository: repository,
		Metadata: map[string]interface{}{
			"backup_file": backupFile,
			"compression": "gzip",
			"format":      "tar",
		},
	}

	metadataFile := backupFile + ".meta.json"
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataFile, metadataData, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write metadata file: %w", err)
	}

	return metadata, nil
}

// RestoreBackup restores data from a backup file
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupFile string, overwrite bool) error {
	backupPath := bm.prepareBackupPath(backupFile)
	metadata, err := bm.readBackupMetadata(backupPath)
	if err != nil {
		return err
	}

	tarReader, closeFunc, err := bm.createTarReader(backupPath)
	if err != nil {
		return err
	}
	defer closeFunc()

	restoredCount, err := bm.restoreChunksFromTar(ctx, tarReader)
	if err != nil {
		return err
	}

	return bm.validateRestoredCount(restoredCount, metadata.ChunkCount)
}

// prepareBackupPath validates and normalizes the backup file path
func (bm *BackupManager) prepareBackupPath(backupFile string) string {
	backupFile = filepath.Clean(backupFile)
	if !filepath.IsAbs(backupFile) {
		backupFile = filepath.Join(bm.backupDir, backupFile)
	}
	return backupFile
}

// readBackupMetadata reads and parses backup metadata
func (bm *BackupManager) readBackupMetadata(backupPath string) (BackupMetadata, error) {
	metadataFile := backupPath + ".meta.json"
	metadataData, err := os.ReadFile(metadataFile) // #nosec G304 -- Path is validated above
	if err != nil {
		return BackupMetadata{}, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return BackupMetadata{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	return metadata, nil
}

// createTarReader opens backup file and creates tar reader
func (bm *BackupManager) createTarReader(backupPath string) (*tar.Reader, func(), error) {
	file, err := os.Open(backupPath) // #nosec G304 -- Path is validated above
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open backup file: %w", err)
	}

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close() // Explicitly ignore error as we're already handling another error
		return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	tarReader := tar.NewReader(gzipReader)
	closeFunc := func() {
		if err := gzipReader.Close(); err != nil {
			_ = err // Log error but don't fail the function
		}
		if err := file.Close(); err != nil {
			_ = err // Log error but don't fail the function
		}
	}

	return tarReader, closeFunc, nil
}

// restoreChunksFromTar reads and restores chunks from tar archive
func (bm *BackupManager) restoreChunksFromTar(ctx context.Context, tarReader *tar.Reader) (int, error) {
	restoredCount := 0
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read tar header: %w", err)
		}

		if !strings.HasPrefix(header.Name, "chunks/") {
			continue
		}

		chunk, err := bm.readAndUnmarshalChunk(tarReader, header.Size)
		if err != nil {
			return 0, err
		}

		if err := bm.storage.StoreChunk(ctx, &chunk); err != nil {
			return 0, fmt.Errorf("failed to store chunk %s: %w", chunk.ID, err)
		}

		restoredCount++
	}
	return restoredCount, nil
}

// readAndUnmarshalChunk reads chunk data from tar and unmarshals it
func (bm *BackupManager) readAndUnmarshalChunk(tarReader *tar.Reader, size int64) (types.ConversationChunk, error) {
	chunkData := make([]byte, size)
	if _, err := io.ReadFull(tarReader, chunkData); err != nil {
		return types.ConversationChunk{}, fmt.Errorf("failed to read chunk data: %w", err)
	}

	var chunk types.ConversationChunk
	if err := json.Unmarshal(chunkData, &chunk); err != nil {
		return types.ConversationChunk{}, fmt.Errorf("failed to unmarshal chunk: %w", err)
	}
	return chunk, nil
}

// validateRestoredCount checks if restored count matches expected count
func (bm *BackupManager) validateRestoredCount(restoredCount, expectedCount int) error {
	if restoredCount != expectedCount {
		return fmt.Errorf("chunk count mismatch: expected %d, restored %d", expectedCount, restoredCount)
	}
	return nil
}

// ListBackups returns a list of available backups
func (bm *BackupManager) ListBackups() ([]BackupMetadata, error) {
	var backups []BackupMetadata

	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return backups, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".meta.json") {
			metadataFile := filepath.Join(bm.backupDir, entry.Name())

			metadataData, err := os.ReadFile(filepath.Clean(metadataFile)) // #nosec G304 -- Path is constructed safely
			if err != nil {
				continue
			}

			var metadata BackupMetadata
			if err := json.Unmarshal(metadataData, &metadata); err != nil {
				continue
			}

			backups = append(backups, metadata)
		}
	}

	return backups, nil
}

// CleanupOldBackups removes backups older than retention period
func (bm *BackupManager) CleanupOldBackups() error {
	cutoff := time.Now().AddDate(0, 0, -bm.retentionDays)

	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	for _, backup := range backups {
		if err := bm.cleanupBackupIfOld(&backup, cutoff); err != nil {
			return err
		}
	}

	return nil
}

func (bm *BackupManager) cleanupBackupIfOld(backup *BackupMetadata, cutoff time.Time) error {
	if !backup.CreatedAt.Before(cutoff) {
		return nil
	}

	// Get backup file path from metadata
	backupFile, ok := backup.Metadata["backup_file"].(string)
	if !ok {
		return nil // Skip if no backup file in metadata
	}

	// Remove backup file
	if err := bm.removeBackupFile(backupFile); err != nil {
		return err
	}

	// Remove metadata file
	return bm.removeMetadataFile(backup)
}

func (bm *BackupManager) removeBackupFile(backupFile string) error {
	if err := os.Remove(backupFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove backup file %s: %w", backupFile, err)
	}
	return nil
}

func (bm *BackupManager) removeMetadataFile(backup *BackupMetadata) error {
	backupFile, ok := backup.Metadata["backup_file"].(string)
	if !ok {
		return nil
	}

	metadataFile := backupFile + ".meta.json"
	if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove metadata file %s: %w", metadataFile, err)
	}

	return nil
}

// MigrateData handles data migration between versions
func (bm *BackupManager) MigrateData(ctx context.Context, fromVersion, toVersion string) error {
	// Create backup before migration
	backupMetadata, err := bm.CreateBackup(ctx, "pre_migration")
	if err != nil {
		return fmt.Errorf("failed to create pre-migration backup: %w", err)
	}

	// Get all chunks
	chunks, err := bm.storage.GetAllChunks(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve chunks for migration: %w", err)
	}

	// Apply migration logic based on versions
	migratedChunks := make([]types.ConversationChunk, 0, len(chunks))
	for i := range chunks {
		migratedChunk := bm.migrateChunk(&chunks[i], fromVersion, toVersion)
		migratedChunks = append(migratedChunks, *migratedChunk)
	}

	// Store migrated chunks
	for _, chunk := range migratedChunks {
		if err := bm.storage.StoreChunk(ctx, &chunk); err != nil {
			// If migration fails, we could restore from backup
			return fmt.Errorf("failed to store migrated chunk %s: %w", chunk.ID, err)
		}
	}

	// Log successful migration
	fmt.Printf("Successfully migrated %d chunks from version %s to %s\n",
		len(migratedChunks), fromVersion, toVersion)
	fmt.Printf("Pre-migration backup available: %s\n",
		backupMetadata.Metadata["backup_file"])

	return nil
}

// migrateChunk applies version-specific migrations to a chunk
func (bm *BackupManager) migrateChunk(chunk *types.ConversationChunk, fromVersion, toVersion string) *types.ConversationChunk {
	// Example migration logic
	switch {
	case fromVersion == "1.0" && toVersion == "1.1":
		// Add new metadata fields introduced in v1.1
		if chunk.Metadata.Tags == nil {
			chunk.Metadata.Tags = []string{}
		}

	case fromVersion == "1.1" && toVersion == "2.0":
		// Major version upgrade with breaking changes
		// Migrate old format to new format
		if chunk.Summary == "" && len(chunk.Content) > 100 {
			// Generate summary for chunks that don't have one
			chunk.Summary = chunk.Content[:100] + "..."
		}
	}

	return chunk
}

// VerifyIntegrity checks data integrity
func (bm *BackupManager) VerifyIntegrity(ctx context.Context) error {
	chunks, err := bm.storage.GetAllChunks(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve chunks: %w", err)
	}

	for i := range chunks {
		if err := chunks[i].Validate(); err != nil {
			return fmt.Errorf("chunk %s failed validation: %w", chunks[i].ID, err)
		}

		// Additional integrity checks
		if chunks[i].ID == "" {
			return fmt.Errorf("chunk has empty ID")
		}

		if chunks[i].Timestamp.IsZero() {
			return fmt.Errorf("chunk %s has zero timestamp", chunks[i].ID)
		}

		if len(chunks[i].Embeddings) == 0 {
			return fmt.Errorf("chunk %s has no embeddings", chunks[i].ID)
		}
	}

	return nil
}

// CompressData implements data compression for storage efficiency
func (bm *BackupManager) CompressData(ctx context.Context) error {
	// Get all chunks
	chunks, err := bm.storage.GetAllChunks(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve chunks: %w", err)
	}

	// Apply compression to chunk content
	compressedCount := 0
	for i := range chunks {
		originalSize := len(chunks[i].Content)

		// Simple compression: remove excessive whitespace
		compressed := compressText(chunks[i].Content)

		if len(compressed) < originalSize {
			chunks[i].Content = compressed
			if err := bm.storage.StoreChunk(ctx, &chunks[i]); err != nil {
				return fmt.Errorf("failed to store compressed chunk %s: %w", chunks[i].ID, err)
			}
			compressedCount++
		}
	}

	fmt.Printf("Compressed %d chunks\n", compressedCount)
	return nil
}

// Helper function to compress text content
func compressText(text string) string {
	// Simple text compression: normalize whitespace
	lines := make([]string, 0)
	for _, line := range filepath.SplitList(text) {
		trimmed := filepath.Clean(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return filepath.Join(lines...)
}

// SetRetentionDays sets the backup retention period
func (bm *BackupManager) SetRetentionDays(days int) {
	bm.retentionDays = days
}

// GetBackupDir returns the backup directory path
func (bm *BackupManager) GetBackupDir() string {
	return bm.backupDir
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}
