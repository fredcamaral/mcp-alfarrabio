#!/usr/bin/env python3
"""
ChromaDB Export Script for Migration to Qdrant

This script exports data from ChromaDB in a format suitable for 
importing into the Qdrant-based MCP Memory system.
"""

import argparse
import json
import logging
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any

try:
    import chromadb
    from chromadb.config import Settings
except ImportError:
    print("ChromaDB not installed. Install with: pip install chromadb")
    sys.exit(1)

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class ChromaDBExporter:
    """Exports ChromaDB data to JSON format compatible with ConversationChunk"""
    
    def __init__(self, chroma_path: str, output_file: str):
        self.chroma_path = Path(chroma_path)
        self.output_file = Path(output_file)
        self.client = None
        self.stats = {
            'collections': 0,
            'total_documents': 0,
            'exported_chunks': 0,
            'errors': 0
        }
    
    def connect(self):
        """Connect to ChromaDB"""
        logger.info(f"Connecting to ChromaDB at: {self.chroma_path}")
        
        try:
            # Try persistent client first
            self.client = chromadb.PersistentClient(
                path=str(self.chroma_path),
                settings=Settings(anonymized_telemetry=False)
            )
            logger.info("✅ Connected to ChromaDB")
        except Exception as e:
            logger.error(f"Failed to connect to ChromaDB: {e}")
            logger.info("Trying with default settings...")
            try:
                self.client = chromadb.PersistentClient(path=str(self.chroma_path))
                logger.info("✅ Connected to ChromaDB with default settings")
            except Exception as e2:
                logger.error(f"Failed to connect with default settings: {e2}")
                raise
    
    def list_collections(self) -> List[str]:
        """List all collections in ChromaDB"""
        try:
            collections = self.client.list_collections()
            collection_names = [coll.name for coll in collections]
            logger.info(f"Found {len(collection_names)} collections: {collection_names}")
            return collection_names
        except Exception as e:
            logger.error(f"Failed to list collections: {e}")
            return []
    
    def convert_to_conversation_chunk(self, doc_id: str, document: str, 
                                    metadata: Dict, embedding: List[float]) -> Dict[str, Any]:
        """Convert ChromaDB document to ConversationChunk format"""
        
        # Extract timestamp - try multiple possible formats
        timestamp = metadata.get('timestamp')
        if isinstance(timestamp, (int, float)):
            # Unix timestamp
            timestamp = datetime.fromtimestamp(timestamp).isoformat()
        elif isinstance(timestamp, str):
            # String timestamp
            try:
                # Try parsing ISO format
                datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
            except:
                # If parsing fails, use current time
                timestamp = datetime.now().isoformat()
        else:
            # Default to current time
            timestamp = datetime.now().isoformat()
        
        # Map ChromaDB metadata to ConversationChunk format
        chunk = {
            "id": doc_id,
            "session_id": metadata.get("session_id", "migrated"),
            "timestamp": timestamp,
            "type": metadata.get("type", "discussion"),
            "content": document or "",
            "summary": metadata.get("summary", ""),
            "metadata": {
                "repository": metadata.get("repository", "migrated"),
                "branch": metadata.get("branch", "main"),
                "files_modified": metadata.get("files_modified", []),
                "tools_used": metadata.get("tools_used", []),
                "outcome": metadata.get("outcome", "success"),
                "tags": metadata.get("tags", []),
                "difficulty": metadata.get("difficulty", "simple"),
                "extended_metadata": {k: v for k, v in metadata.items() 
                                   if k not in ['repository', 'branch', 'files_modified', 
                                              'tools_used', 'outcome', 'tags', 'difficulty']}
            },
            "embeddings": embedding or [],
            "related_chunks": metadata.get("related_chunks", [])
        }
        
        return chunk
    
    def export_collection(self, collection_name: str) -> List[Dict[str, Any]]:
        """Export a single collection"""
        logger.info(f"Exporting collection: {collection_name}")
        
        try:
            collection = self.client.get_collection(collection_name)
            
            # Get all documents from the collection
            results = collection.get(
                include=["documents", "metadatas", "embeddings"]
            )
            
            chunks = []
            doc_count = len(results.get("ids", []))
            logger.info(f"Found {doc_count} documents in {collection_name}")
            
            for i in range(doc_count):
                try:
                    doc_id = results["ids"][i] if i < len(results.get("ids", [])) else f"doc_{i}"
                    document = results["documents"][i] if i < len(results.get("documents", [])) else ""
                    metadata = results["metadatas"][i] if i < len(results.get("metadatas", [])) else {}
                    embedding = results["embeddings"][i] if i < len(results.get("embeddings", [])) else []
                    
                    # Convert to ConversationChunk format
                    chunk = self.convert_to_conversation_chunk(doc_id, document, metadata, embedding)
                    chunks.append(chunk)
                    self.stats['exported_chunks'] += 1
                    
                except Exception as e:
                    logger.warning(f"Failed to process document {i} in {collection_name}: {e}")
                    self.stats['errors'] += 1
            
            self.stats['total_documents'] += doc_count
            logger.info(f"✅ Exported {len(chunks)} chunks from {collection_name}")
            return chunks
            
        except Exception as e:
            logger.error(f"Failed to export collection {collection_name}: {e}")
            self.stats['errors'] += 1
            return []
    
    def export_all(self) -> Dict[str, Any]:
        """Export all collections to ConversationChunk format"""
        logger.info("Starting ChromaDB export...")
        
        self.connect()
        collection_names = self.list_collections()
        
        if not collection_names:
            logger.warning("No collections found in ChromaDB")
            return {"chunks": [], "metadata": self.get_export_metadata()}
        
        all_chunks = []
        
        for collection_name in collection_names:
            chunks = self.export_collection(collection_name)
            all_chunks.extend(chunks)
            self.stats['collections'] += 1
        
        export_data = {
            "chunks": all_chunks,
            "metadata": self.get_export_metadata()
        }
        
        logger.info(f"Export completed: {len(all_chunks)} total chunks from {len(collection_names)} collections")
        return export_data
    
    def get_export_metadata(self) -> Dict[str, Any]:
        """Get metadata about the export"""
        return {
            "export_timestamp": datetime.now().isoformat(),
            "source": "ChromaDB",
            "source_path": str(self.chroma_path),
            "target_format": "ConversationChunk",
            "version": "1.0.0",
            "stats": self.stats
        }
    
    def save_to_file(self, data: Dict[str, Any]):
        """Save exported data to JSON file"""
        logger.info(f"Saving export to: {self.output_file}")
        
        try:
            self.output_file.parent.mkdir(parents=True, exist_ok=True)
            
            with open(self.output_file, 'w', encoding='utf-8') as f:
                json.dump(data, f, indent=2, ensure_ascii=False)
            
            file_size = self.output_file.stat().st_size / (1024 * 1024)  # MB
            logger.info(f"✅ Export saved successfully ({file_size:.2f} MB)")
            
        except Exception as e:
            logger.error(f"Failed to save export file: {e}")
            raise
    
    def print_summary(self):
        """Print export summary"""
        print("\n" + "="*60)
        print("CHROMADB EXPORT SUMMARY")
        print("="*60)
        print(f"Collections processed: {self.stats['collections']}")
        print(f"Total documents: {self.stats['total_documents']}")
        print(f"Exported chunks: {self.stats['exported_chunks']}")
        print(f"Errors: {self.stats['errors']}")
        print(f"Output file: {self.output_file}")
        print(f"File size: {self.output_file.stat().st_size / (1024 * 1024):.2f} MB")
        
        if self.stats['errors'] > 0:
            print(f"\n⚠️  {self.stats['errors']} errors occurred during export")
        else:
            print("\n✅ Export completed successfully!")
        
        print("\nNext steps:")
        print("1. Review the exported data in the JSON file")
        print("2. Use the Go migration tool to import into Qdrant:")
        print(f"   go run cmd/migrate/main.go -chroma-export='{self.output_file}' -config='configs/dev/config.yaml'")
        print("="*60)

def main():
    parser = argparse.ArgumentParser(description="Export ChromaDB data for migration to Qdrant")
    parser.add_argument("chroma_path", help="Path to ChromaDB data directory")
    parser.add_argument("-o", "--output", default="chromadb_export.json", 
                       help="Output JSON file (default: chromadb_export.json)")
    parser.add_argument("-v", "--verbose", action="store_true", help="Enable verbose logging")
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    # Validate input path
    chroma_path = Path(args.chroma_path)
    if not chroma_path.exists():
        logger.error(f"ChromaDB path does not exist: {chroma_path}")
        sys.exit(1)
    
    # Run export
    try:
        exporter = ChromaDBExporter(args.chroma_path, args.output)
        data = exporter.export_all()
        exporter.save_to_file(data)
        exporter.print_summary()
        
    except KeyboardInterrupt:
        logger.info("Export cancelled by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Export failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()