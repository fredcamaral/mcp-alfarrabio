// Core memory types based on the MCP Memory system

export type ChunkType = 
  | "problem"
  | "solution" 
  | "architecture_decision"
  | "session_summary"
  | "code_change"
  | "discussion"
  | "analysis"
  | "verification"
  | "question"

export type Outcome = "success" | "in_progress" | "failed" | "abandoned"
export type Difficulty = "simple" | "moderate" | "complex"

export interface ChunkMetadata {
  repository?: string
  branch?: string
  files_modified?: string[]
  tools_used?: string[]
  outcome?: Outcome
  tags?: string[]
  difficulty?: Difficulty
  time_spent?: number
  extended_metadata?: Record<string, any>
  confidence?: ConfidenceMetrics
  quality?: QualityMetrics
}

export interface ConfidenceMetrics {
  score: number
  factors: ConfidenceFactors
  source: string
  last_updated: string
}

export interface ConfidenceFactors {
  user_feedback: number
  pattern_match: number 
  context_relevance: number
  temporal_proximity: number
  cross_validation: number
}

export interface QualityMetrics {
  score: number
  completeness: number
  clarity: number
  actionability: number
  last_assessed: string
}

export interface ConversationChunk {
  id: string
  session_id: string
  timestamp: string
  type: ChunkType
  content: string
  summary?: string
  metadata: ChunkMetadata
  embeddings?: number[]
  related_chunks?: string[]
}

export interface SearchResult {
  chunk: ConversationChunk
  score: number
  explanation?: string
}

export interface SearchResults {
  results: SearchResult[]
  total: number
  query_time: number
}

export interface MemoryQuery {
  query: string
  repository?: string
  types?: ChunkType[]
  limit?: number
  min_relevance?: number
  session_id?: string
  time_range?: {
    start?: string
    end?: string
  }
}

// Relationship types
export type RelationType =
  | "led_to"
  | "solved_by"
  | "depends_on"
  | "enables"
  | "conflicts_with"
  | "supersedes"
  | "related_to"
  | "follows_up"
  | "precedes"
  | "learned_from"
  | "teaches"
  | "exemplifies"
  | "referenced_by"
  | "references"

export interface MemoryRelationship {
  id: string
  source_chunk_id: string
  target_chunk_id: string
  relation_type: RelationType
  confidence: number
  confidence_source: string
  confidence_factors?: ConfidenceFactors
  metadata?: Record<string, any>
  created_at: string
  created_by?: string
  last_validated?: string
  validation_count: number
}

export interface RelationshipResult {
  relationship: MemoryRelationship
  source_chunk?: ConversationChunk
  target_chunk?: ConversationChunk
}

export interface GraphNode {
  id: string
  label: string
  type: ChunkType
  metadata?: Record<string, any>
}

export interface GraphEdge {
  source: string
  target: string
  relation_type: RelationType
  confidence: number
}

export interface GraphPath {
  nodes: string[]
  edges: GraphEdge[]
  total_confidence: number
}

export interface GraphTraversalResult {
  paths: GraphPath[]
  nodes: GraphNode[]
  edges: GraphEdge[]
}

// Pattern and analytics types
export interface Pattern {
  id: string
  name: string
  description: string
  frequency: number
  confidence: number
  examples: string[]
  related_patterns: string[]
}

export interface Repository {
  name: string
  description?: string
  chunk_count: number
  last_activity: string
  patterns: Pattern[]
  technologies: string[]
}

export interface StoreStats {
  total_chunks: number
  chunks_by_type: Record<string, number>
  chunks_by_repo: Record<string, number>
  oldest_chunk?: string
  newest_chunk?: string
  storage_size_bytes: number
  average_embedding_size: number
}

// UI state types
export interface FilterState {
  query: string
  repository?: string
  types: ChunkType[]
  time_range: "recent" | "month" | "all"
  min_relevance: number
}

export interface UIState {
  sidebar_open: boolean
  selected_memory?: string
  filter_panel_open: boolean
  theme: "light" | "dark" | "system"
  view_mode: "list" | "graph" | "timeline"
}

// API response types
export interface ApiResponse<T> {
  data?: T
  error?: string
  loading: boolean
}

export interface PaginationInfo {
  page: number
  limit: number
  total: number
  has_next: boolean
  has_prev: boolean
}