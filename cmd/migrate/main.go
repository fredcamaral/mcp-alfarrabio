// migrate is a command-line tool for migrating data from ChromaDB to Qdrant vector database,
// providing batch migration, validation, and backup capabilities for the MCP Memory Server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mcp-memory/internal/config"
	"mcp-memory/internal/storage"
	"mcp-memory/pkg/types"
)

const (
	batchSize           = 100
	migrationVersion    = "1.0.0"
	backupDirPermission = 0755
)

// MigrationStats tracks the progress of migration
type MigrationStats struct {
	TotalChunks      int           `json:"total_chunks"`
	MigratedChunks   int           `json:"migrated_chunks"`
	FailedChunks     int           `json:"failed_chunks"`
	StartTime        time.Time     `json:"start_time"`
	Duration         time.Duration `json:"duration"`
	BatchesMigrated  int           `json:"batches_migrated"`
	ValidationPassed bool          `json:"validation_passed"`
}

// MigrationTool handles the migration from ChromaDB to Qdrant
type MigrationTool struct {
	inputPath    string
	qdrantStore  storage.VectorStore
	backupDir    string
	dryRun       bool
	validateOnly bool
	isJSONExport bool
	stats        *MigrationStats
}

func main() {
	var (
		chromaDBPath = flag.String("chroma-path", "", "Path to ChromaDB data directory")
		chromaExport = flag.String("chroma-export", "", "Path to ChromaDB JSON export file")
		_            = flag.String("config", "configs/dev/config.yaml", "Path to configuration file (unused - uses env vars)")
		backupDir    = flag.String("backup-dir", "./migration-backup", "Directory for migration backups")
		dryRun       = flag.Bool("dry-run", false, "Perform dry run without writing to Qdrant")
		validateOnly = flag.Bool("validate-only", false, "Only validate existing data, don't migrate")
		force        = flag.Bool("force", false, "Force migration even if target collection exists")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *chromaDBPath == "" && *chromaExport == "" {
		fmt.Fprintf(os.Stderr, "Error: Either -chroma-path or -chroma-export is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *chromaDBPath != "" && *chromaExport != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both -chroma-path and -chroma-export\n")
		flag.Usage()
		os.Exit(1)
	}

	// Setup logging (simple approach for migration tool)
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate input path exists
	inputPath := *chromaDBPath
	if *chromaExport != "" {
		inputPath = *chromaExport
	}

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Fatalf("Input path does not exist: %s", inputPath)
	}

	// Create migration tool
	migrator, err := NewMigrationTool(inputPath, cfg, *backupDir, *dryRun, *validateOnly, *chromaExport != "")
	if err != nil {
		log.Fatalf("Failed to create migration tool: %v", err)
	}

	// Run migration
	if err := migrator.Migrate(context.Background(), *force); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Print results
	migrator.PrintResults()
}

// NewMigrationTool creates a new migration tool instance
func NewMigrationTool(inputPath string, cfg *config.Config, backupDir string, dryRun, validateOnly, isJSONExport bool) (*MigrationTool, error) {
	// Create Qdrant store
	qdrantStore := storage.NewQdrantStore(&cfg.Qdrant)

	// Create backup directory
	if err := os.MkdirAll(backupDir, backupDirPermission); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &MigrationTool{
		inputPath:    inputPath,
		qdrantStore:  qdrantStore,
		backupDir:    backupDir,
		dryRun:       dryRun,
		validateOnly: validateOnly,
		isJSONExport: isJSONExport,
		stats: &MigrationStats{
			StartTime: time.Now(),
		},
	}, nil
}

// Migrate performs the complete migration process
func (mt *MigrationTool) Migrate(ctx context.Context, force bool) error {
	log.Printf("Starting ChromaDB to Qdrant migration: input_path=%s, dry_run=%v, validate_only=%v",
		mt.inputPath, mt.dryRun, mt.validateOnly)

	// Initialize Qdrant
	if !mt.dryRun && !mt.validateOnly {
		if err := mt.qdrantStore.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize Qdrant: %w", err)
		}
	}

	// Check if target collection already has data
	if !mt.validateOnly && !force && !mt.dryRun {
		if err := mt.checkTargetCollection(ctx); err != nil {
			return err
		}
	}

	// Create backup before migration
	if !mt.dryRun && !mt.validateOnly {
		if err := mt.createPreMigrationBackup(ctx); err != nil {
			log.Printf("Failed to create backup, continuing anyway: %v", err)
		}
	}

	// Read ChromaDB data
	chunks, err := mt.readChromaDBData(ctx)
	if err != nil {
		return fmt.Errorf("failed to read ChromaDB data: %w", err)
	}

	mt.stats.TotalChunks = len(chunks)
	log.Printf("Found chunks to migrate: count=%d", mt.stats.TotalChunks)

	if mt.validateOnly {
		return mt.validateData(chunks)
	}

	// Migrate data in batches
	if err := mt.migrateInBatches(ctx, chunks); err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}

	// Validate migration
	if !mt.dryRun {
		if err := mt.validateMigration(ctx, chunks); err != nil {
			return fmt.Errorf("migration validation failed: %w", err)
		}
	}

	mt.stats.Duration = time.Since(mt.stats.StartTime)
	log.Printf("Migration completed successfully: total_chunks=%d, migrated_chunks=%d, failed_chunks=%d, duration=%v",
		mt.stats.TotalChunks, mt.stats.MigratedChunks, mt.stats.FailedChunks, mt.stats.Duration)

	return nil
}

// readChromaDBData reads all chunks from ChromaDB or JSON export
func (mt *MigrationTool) readChromaDBData(ctx context.Context) ([]types.ConversationChunk, error) {
	if mt.isJSONExport {
		return mt.readJSONExport()
	}
	return mt.readDirectChromaDB(ctx)
}

// readJSONExport reads chunks from JSON export file
func (mt *MigrationTool) readJSONExport() ([]types.ConversationChunk, error) {
	log.Printf("Reading JSON export: path=%s", mt.inputPath)

	file, err := os.Open(mt.inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON export file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Failed to close file: %v", closeErr)
		}
	}()

	var exportData struct {
		Chunks   []types.ConversationChunk `json:"chunks"`
		Metadata map[string]interface{}    `json:"metadata"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&exportData); err != nil {
		return nil, fmt.Errorf("failed to decode JSON export: %w", err)
	}

	log.Printf("Loaded chunks from JSON export: count=%d", len(exportData.Chunks))

	if exportData.Metadata != nil {
		if stats, ok := exportData.Metadata["stats"].(map[string]interface{}); ok {
			log.Printf("Export metadata: original_stats=%v", stats)
		}
	}

	return exportData.Chunks, nil
}

// readDirectChromaDB reads data directly from ChromaDB
func (mt *MigrationTool) readDirectChromaDB(ctx context.Context) ([]types.ConversationChunk, error) {
	_ = ctx // unused in placeholder implementation
	log.Printf("Reading ChromaDB data: path=%s", mt.inputPath)

	// NOTE: This is a placeholder for ChromaDB reading logic
	// In a real implementation, you would:
	// 1. Connect to ChromaDB using the original client
	// 2. Query all collections
	// 3. Read all documents with their embeddings and metadata
	// 4. Convert to our ConversationChunk format

	// For now, return empty slice as we no longer have ChromaDB client code
	// This would need to be implemented based on your specific ChromaDB setup
	log.Printf("Direct ChromaDB reading not implemented - this is a template")
	log.Printf("To complete migration, you would need to:")
	log.Printf("1. Add ChromaDB client dependency")
	log.Printf("2. Implement readDirectChromaDB to query existing ChromaDB")
	log.Printf("3. Convert ChromaDB documents to ConversationChunk format")
	log.Printf("")
	log.Printf("RECOMMENDED: Use the JSON export approach instead:")
	log.Printf("1. Run: python scripts/export_chromadb.py /path/to/chromadb")
	log.Printf("2. Then: go run cmd/migrate/main.go -chroma-export=chromadb_export.json")

	return []types.ConversationChunk{}, nil
}

// migrateInBatches migrates chunks to Qdrant in batches
func (mt *MigrationTool) migrateInBatches(ctx context.Context, chunks []types.ConversationChunk) error {
	log.Printf("Starting batch migration: total_chunks=%d, batch_size=%d", len(chunks), batchSize)

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]

		if mt.dryRun {
			log.Printf("DRY RUN: Would migrate batch: batch=%d, chunks=%d", mt.stats.BatchesMigrated+1, len(batch))
			mt.stats.MigratedChunks += len(batch)
		} else {
			if err := mt.migrateBatch(ctx, batch); err != nil {
				log.Printf("Failed to migrate batch: batch=%d, error=%v", mt.stats.BatchesMigrated+1, err)
				mt.stats.FailedChunks += len(batch)
				continue
			}
			mt.stats.MigratedChunks += len(batch)
		}

		mt.stats.BatchesMigrated++

		// Progress update
		progress := float64(mt.stats.MigratedChunks+mt.stats.FailedChunks) / float64(mt.stats.TotalChunks) * 100
		log.Printf("Migration progress: %.1f%%, migrated=%d, failed=%d, remaining=%d",
			progress, mt.stats.MigratedChunks, mt.stats.FailedChunks,
			mt.stats.TotalChunks-(mt.stats.MigratedChunks+mt.stats.FailedChunks))
	}

	// Check if too many chunks failed
	if mt.stats.FailedChunks > 0 && float64(mt.stats.FailedChunks)/float64(mt.stats.TotalChunks) > 0.5 {
		return fmt.Errorf("migration failed: too many chunks failed (%d/%d)", mt.stats.FailedChunks, mt.stats.TotalChunks)
	}

	return nil
}

// migrateBatch migrates a single batch of chunks
func (mt *MigrationTool) migrateBatch(ctx context.Context, batch []types.ConversationChunk) error {
	result, err := mt.qdrantStore.BatchStore(ctx, batch)
	if err != nil {
		return fmt.Errorf("batch store failed: %w", err)
	}

	if result.Failed > 0 {
		log.Printf("Some chunks failed in batch: failed=%d, success=%d, errors=%v",
			result.Failed, result.Success, result.Errors)
	}

	return nil
}

// validateMigration validates that all data was migrated correctly
func (mt *MigrationTool) validateMigration(ctx context.Context, originalChunks []types.ConversationChunk) error {
	log.Printf("Validating migration")

	// Get stats from Qdrant
	stats, err := mt.qdrantStore.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Qdrant stats: %w", err)
	}

	log.Printf("Qdrant stats after migration: total_chunks=%d, chunks_by_type=%v, chunks_by_repo=%v",
		stats.TotalChunks, stats.ChunksByType, stats.ChunksByRepo)

	// Basic validation: check total count
	if int64(len(originalChunks)) != stats.TotalChunks {
		return fmt.Errorf("chunk count mismatch: expected %d, got %d", len(originalChunks), stats.TotalChunks)
	}

	// Sample validation: check a few random chunks
	if len(originalChunks) > 0 {
		sampleSize := minInt(10, len(originalChunks))
		for i := 0; i < sampleSize; i++ {
			chunk := originalChunks[i]
			retrieved, err := mt.qdrantStore.GetByID(ctx, chunk.ID)
			if err != nil {
				return fmt.Errorf("failed to retrieve chunk %s: %w", chunk.ID, err)
			}

			if retrieved.Content != chunk.Content {
				return fmt.Errorf("content mismatch for chunk %s", chunk.ID)
			}
		}
	}

	mt.stats.ValidationPassed = true
	log.Printf("Migration validation passed")
	return nil
}

// validateData validates the data structure without migrating
func (mt *MigrationTool) validateData(chunks []types.ConversationChunk) error {
	log.Printf("Validating data structure: chunks=%d", len(chunks))

	invalidChunks := 0
	for _, chunk := range chunks {
		if err := chunk.Validate(); err != nil {
			log.Printf("Invalid chunk found: id=%s, error=%v", chunk.ID, err)
			invalidChunks++
		}
	}

	if invalidChunks > 0 {
		log.Printf("Found invalid chunks: count=%d, total=%d", invalidChunks, len(chunks))
	} else {
		log.Printf("All chunks are valid")
		mt.stats.ValidationPassed = true
	}

	return nil
}

// checkTargetCollection checks if target collection already has data
func (mt *MigrationTool) checkTargetCollection(ctx context.Context) error {
	stats, err := mt.qdrantStore.GetStats(ctx)
	if err != nil {
		// Collection might not exist yet, which is fine
		return nil
	}

	if stats.TotalChunks > 0 {
		return fmt.Errorf("target Qdrant collection already contains %d chunks. Use -force to override", stats.TotalChunks)
	}

	return nil
}

// createPreMigrationBackup creates a backup before migration
func (mt *MigrationTool) createPreMigrationBackup(ctx context.Context) error {
	_ = ctx // unused in placeholder implementation
	backupPath := filepath.Join(mt.backupDir, fmt.Sprintf("pre_migration_%s.tar.gz", time.Now().Format("20060102_150405")))

	log.Printf("Creating pre-migration backup: path=%s", backupPath)

	// Create the backup directory with secure permissions
	if err := os.MkdirAll(mt.backupDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// This would use the backup manager to create a backup
	// For now, just create a placeholder file with secure path handling
	placeholderPath := filepath.Clean(backupPath + ".placeholder")
	file, err := os.Create(placeholderPath)
	if err != nil {
		return fmt.Errorf("failed to create backup placeholder: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Failed to close backup file: %v", closeErr)
		}
	}()

	return nil
}

// PrintResults prints the final migration results
func (mt *MigrationTool) PrintResults() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("MIGRATION RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total chunks: %d\n", mt.stats.TotalChunks)
	fmt.Printf("Migrated: %d\n", mt.stats.MigratedChunks)
	fmt.Printf("Failed: %d\n", mt.stats.FailedChunks)
	fmt.Printf("Batches processed: %d\n", mt.stats.BatchesMigrated)
	fmt.Printf("Duration: %v\n", mt.stats.Duration)
	fmt.Printf("Validation passed: %v\n", mt.stats.ValidationPassed)

	if mt.dryRun {
		fmt.Println("\nNOTE: This was a dry run. No data was actually migrated.")
	}

	if mt.stats.FailedChunks > 0 {
		fmt.Printf("\nWARNING: %d chunks failed to migrate. Check logs for details.\n", mt.stats.FailedChunks)
	}

	if mt.stats.ValidationPassed && mt.stats.FailedChunks == 0 && !mt.dryRun {
		fmt.Println("\n✅ Migration completed successfully!")
	}
	fmt.Println(strings.Repeat("=", 60))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
