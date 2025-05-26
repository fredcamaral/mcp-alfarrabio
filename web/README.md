# Memory System Web UI

A simple web interface for visualizing and searching memories in the MCP Memory system.

## Features

- **Memory List View**: Browse recent memories with summaries and tags
- **Search**: Full-text search across all memories
- **Filters**: Filter by repository, time range, and memory type
- **Detail View**: See full memory content and metadata
- **Relationship Visualization**: Simple graph showing related memories
- **Real-time Updates**: Automatically refreshes when filters change

## Usage

1. Start the GraphQL server:
   ```bash
   ./graphql
   ```

2. Open your browser to: http://localhost:8082/

3. The UI will load with recent memories displayed

## Interface Overview

### Top Controls
- **Search Box**: Enter natural language queries
- **Repository Filter**: Select specific project or global memories
- **Recency Filter**: Recent (7 days), Last Month, or All Time
- **Type Filter**: Filter by memory type (problems, solutions, decisions, etc.)

### Main Content
- **Left Panel**: List of memories with summaries, scores, and tags
- **Right Panel**: Detailed view of selected memory

### Visualization
- **Bottom Panel**: Graph showing relationships between memories
  - Center node: Selected memory
  - Connected nodes: Related memories (same session or shared tags)

## Memory Types

- 🐛 **Problem**: Issues or errors encountered
- ✅ **Solution**: Fixes or solutions implemented
- 🏗️ **Architecture Decision**: Design choices and rationale
- 📋 **Session Summary**: Overview of work sessions
- 💻 **Code Change**: Significant code modifications
- 💬 **Discussion**: Important conversations
- 📊 **Analysis**: Deep dives or investigations
- ✓ **Verification**: Testing or validation results

## Tips

1. Click on any memory in the list to see full details
2. The visualization updates to show related memories
3. Use specific search terms for better results
4. Filter by repository to focus on project-specific memories
5. Tags help identify memory categories quickly

## Technical Details

- Built with vanilla JavaScript (no framework dependencies)
- Uses GraphQL API for data fetching
- Canvas-based visualization for relationships
- Responsive design for different screen sizes
- Simple, fast, and lightweight