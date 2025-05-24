package persistence

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	ChunkCount  int                    `json:"chunk_count"`
	Size        int64                  `json:"size"`
	Checksum    string                 `json:"checksum"`
	Repository  string                 `json:"repository,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// VectorStorage interface for backup operations
type VectorStorage interface {
	GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error)
	StoreChunk(ctx context.Context, chunk types.ConversationChunk) error
	DeleteCollection(ctx context.Context, collection string) error
	ListCollections(ctx context.Context) ([]string, error)
}

// NewBackupManager creates a new backup manager
func NewBackupManager(storage VectorStorage, backupDir string) *BackupManager {
	return &BackupManager{
		storage:       storage,
		backupDir:     backupDir,
		retentionDays: 30, // Default 30 days retention
	}
}

// CreateBackup creates a complete backup of all data
func (bm *BackupManager) CreateBackup(ctx context.Context, repository string) (*BackupMetadata, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(bm.backupDir, fmt.Sprintf("backup_%s_%s.tar.gz", repository, timestamp))
	
	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Get all chunks
	chunks, err := bm.storage.GetAllChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks: %w", err)
	}
	
	// Filter chunks by repository if specified
	if repository != "" {
		filteredChunks := make([]types.ConversationChunk, 0)
		for _, chunk := range chunks {
			if chunk.Metadata.Repository == repository {
				filteredChunks = append(filteredChunks, chunk)
			}
		}
		chunks = filteredChunks
	}
	
	// Create backup file
	file, err := os.Create(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()
	
	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	// Write chunks to tar
	for i, chunk := range chunks {
		chunkData, err := json.MarshalIndent(chunk, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal chunk %s: %w", chunk.ID, err)
		}
		
		header := &tar.Header{
			Name: fmt.Sprintf("chunks/chunk_%d_%s.json", i, chunk.ID),
			Size: int64(len(chunkData)),
			Mode: 0644,
		}
		
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write tar header: %w", err)
		}
		
		if _, err := tarWriter.Write(chunkData); err != nil {
			return nil, fmt.Errorf("failed to write chunk data: %w", err)
		}
	}
	
	// Create metadata
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}
	
	metadata := &BackupMetadata{
		Version:    "1.0",
		CreatedAt:  time.Now(),
		ChunkCount: len(chunks),
		Size:       stat.Size(),
		Repository: repository,
		Metadata: map[string]interface{}{
			"backup_file": backupFile,
			"compression": "gzip",
			"format":      "tar",
		},
	}
	
	// Write metadata file
	metadataFile := backupFile + ".meta.json"
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := os.WriteFile(metadataFile, metadataData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata file: %w", err)
	}
	
	return metadata, nil
}

// RestoreBackup restores data from a backup file
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupFile string, overwrite bool) error {
	// Read metadata
	metadataFile := backupFile + ".meta.json"
	metadataData, err := os.ReadFile(metadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	var metadata BackupMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	// Open backup file
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()
	
	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzipReader)
	
	// Read and restore chunks
	restoredCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		
		if !filepath.HasPrefix(header.Name, "chunks/") {
			continue
		}
		
		// Read chunk data
		chunkData := make([]byte, header.Size)
		if _, err := io.ReadFull(tarReader, chunkData); err != nil {
			return fmt.Errorf("failed to read chunk data: %w", err)
		}
		
		// Unmarshal chunk
		var chunk types.ConversationChunk
		if err := json.Unmarshal(chunkData, &chunk); err != nil {
			return fmt.Errorf("failed to unmarshal chunk: %w", err)
		}
		
		// Store chunk
		if err := bm.storage.StoreChunk(ctx, chunk); err != nil {
			return fmt.Errorf("failed to store chunk %s: %w", chunk.ID, err)
		}
		
		restoredCount++
	}
	
	if restoredCount != metadata.ChunkCount {
		return fmt.Errorf("chunk count mismatch: expected %d, restored %d", metadata.ChunkCount, restoredCount)
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
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" && 
		   filepath.Base(entry.Name()) != entry.Name() {
			metadataFile := filepath.Join(bm.backupDir, entry.Name())
			
			metadataData, err := os.ReadFile(metadataFile)
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
		if backup.CreatedAt.Before(cutoff) {
			// Get backup file path from metadata
			if backupFile, ok := backup.Metadata["backup_file"].(string); ok {
				// Remove backup file
				if err := os.Remove(backupFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove backup file %s: %w", backupFile, err)
				}
				
				// Remove metadata file
				metadataFile := backupFile + ".meta.json"
				if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove metadata file %s: %w", metadataFile, err)
				}
			}
		}
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
	for _, chunk := range chunks {
		migratedChunk, err := bm.migrateChunk(chunk, fromVersion, toVersion)
		if err != nil {
			return fmt.Errorf("failed to migrate chunk %s: %w", chunk.ID, err)
		}
		migratedChunks = append(migratedChunks, migratedChunk)
	}
	
	// Store migrated chunks
	for _, chunk := range migratedChunks {
		if err := bm.storage.StoreChunk(ctx, chunk); err != nil {
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
func (bm *BackupManager) migrateChunk(chunk types.ConversationChunk, fromVersion, toVersion string) (types.ConversationChunk, error) {
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
	
	return chunk, nil
}

// VerifyIntegrity checks data integrity
func (bm *BackupManager) VerifyIntegrity(ctx context.Context) error {
	chunks, err := bm.storage.GetAllChunks(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve chunks: %w", err)
	}
	
	for _, chunk := range chunks {
		if err := chunk.Validate(); err != nil {
			return fmt.Errorf("chunk %s failed validation: %w", chunk.ID, err)
		}
		
		// Additional integrity checks
		if chunk.ID == "" {
			return fmt.Errorf("chunk has empty ID")
		}
		
		if chunk.Timestamp.IsZero() {
			return fmt.Errorf("chunk %s has zero timestamp", chunk.ID)
		}
		
		if len(chunk.Embeddings) == 0 {
			return fmt.Errorf("chunk %s has no embeddings", chunk.ID)
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
	for _, chunk := range chunks {
		originalSize := len(chunk.Content)
		
		// Simple compression: remove excessive whitespace
		compressed := compressText(chunk.Content)
		
		if len(compressed) < originalSize {
			chunk.Content = compressed
			if err := bm.storage.StoreChunk(ctx, chunk); err != nil {
				return fmt.Errorf("failed to store compressed chunk %s: %w", chunk.ID, err)
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