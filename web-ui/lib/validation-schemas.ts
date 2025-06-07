/**
 * Validation schemas for API requests and form inputs
 * 
 * Provides Zod schemas for validating data across the WebUI to ensure
 * type safety and data integrity.
 */

import { z } from 'zod'

// Common schema parts
const idSchema = z.string().uuid()
const timestampSchema = z.string().datetime()
const urlSchema = z.string().url()
const repositorySchema = z.string().min(1).max(255)
const sessionIdSchema = z.string().uuid()

// Memory type enum - must match ChunkType from types/memory.ts
export const memoryTypeSchema = z.enum([
  'problem',
  'solution',
  'architecture_decision',
  'session_summary',
  'code_change',
  'discussion',
  'analysis',
  'verification',
  'question'
])

// Repository status enum
export const repositoryStatusSchema = z.enum([
  'ACTIVE',
  'INACTIVE',
  'SYNCING',
  'ERROR'
])

// Priority enum
export const prioritySchema = z.enum(['low', 'medium', 'high'])

// Configuration schemas
export const configSchemas = {
  update: z.object({
    host: z.string().min(1).optional(),
    port: z.number().int().min(1).max(65535).optional(),
    protocol: z.enum(['http', 'https']).optional(),
    transportProtocols: z.object({
      http: z.boolean().optional(),
      websocket: z.boolean().optional(),
      grpc: z.boolean().optional(),
    }).optional(),
    vectorDb: z.object({
      provider: z.enum(['qdrant', 'chroma']).optional(),
      host: z.string().min(1).optional(),
      port: z.number().int().min(1).optional(),
      collection: z.string().min(1).optional(),
      dimension: z.number().int().min(1).optional(),
    }).optional(),
    openai: z.object({
      apiKey: z.string().optional(),
      model: z.string().optional(),
      maxTokens: z.number().int().min(1).optional(),
      temperature: z.number().min(0).max(2).optional(),
      timeout: z.number().int().min(1000).optional(),
    }).optional(),
    cacheEnabled: z.boolean().optional(),
    realtimeEnabled: z.boolean().optional(),
    analyticsEnabled: z.boolean().optional(),
    debugMode: z.boolean().optional(),
    authEnabled: z.boolean().optional(),
  }).refine(data => {
    // At least one field must be provided
    return Object.values(data).some(v => v !== undefined)
  }, 'At least one configuration field must be provided')
}

// Backup schemas
export const backupSchemas = {
  create: z.object({
    repository: z.string().optional(),
    name: z.string().min(1).max(255).optional(),
    includeVectors: z.boolean().optional(),
    format: z.enum(['json', 'msgpack', 'csv']).optional(),
    compress: z.boolean().optional(),
  }),
  
  restore: z.object({
    file: z.string().min(1),
    overwrite: z.boolean().optional(),
    validateIntegrity: z.boolean().optional(),
  })
}

// Memory schemas
export const memorySchemas = {
  create: z.object({
    content: z.string().min(10).max(50000),
    type: memoryTypeSchema,
    repository: repositorySchema,
    sessionId: sessionIdSchema.optional(),
    tags: z.array(z.string().min(1).max(50)).max(20).optional(),
    metadata: z.record(z.unknown()).optional(),
    priority: prioritySchema.optional(),
    isPublic: z.boolean().optional(),
  }),
  
  update: z.object({
    id: idSchema,
    content: z.string().min(10).max(50000).optional(),
    type: memoryTypeSchema.optional(),
    tags: z.array(z.string().min(1).max(50)).max(20).optional(),
    metadata: z.record(z.unknown()).optional(),
    priority: prioritySchema.optional(),
  }).refine(data => {
    // At least one field besides id must be provided
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { id, ...rest } = data
    return Object.values(rest).some(v => v !== undefined)
  }, 'At least one field must be provided for update'),
  
  search: z.object({
    query: z.string().min(1).max(1000),
    repository: repositorySchema.optional(),
    types: z.array(memoryTypeSchema).optional(),
    tags: z.array(z.string()).optional(),
    sessionId: sessionIdSchema.optional(),
    limit: z.number().int().min(1).max(100).optional(),
    offset: z.number().int().min(0).optional(),
    minRelevanceScore: z.number().min(0).max(1).optional(),
    timeRange: z.object({
      start: timestampSchema.optional(),
      end: timestampSchema.optional(),
    }).optional(),
  }),
  
  delete: z.object({
    id: idSchema,
    confirm: z.boolean(),
  })
}

// Repository schemas
export const repositorySchemas = {
  add: z.object({
    url: urlSchema,
    description: z.string().max(500).optional(),
    metadata: z.record(z.unknown()).optional(),
  }),
  
  update: z.object({
    id: idSchema,
    description: z.string().max(500).optional(),
    status: repositoryStatusSchema.optional(),
    metadata: z.record(z.unknown()).optional(),
  }).refine(data => {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { id, ...rest } = data
    return Object.values(rest).some(v => v !== undefined)
  }, 'At least one field must be provided for update'),
  
  sync: z.object({
    id: idSchema,
    force: z.boolean().optional(),
  }),
  
  remove: z.object({
    id: idSchema,
    confirm: z.boolean(),
    removeData: z.boolean().optional(),
  })
}

// Pattern schemas
export const patternSchemas = {
  search: z.object({
    repository: repositorySchema,
    types: z.array(z.string()).optional(),
    minConfidence: z.number().min(0).max(1).optional(),
    limit: z.number().int().min(1).max(100).optional(),
  })
}

// WebSocket message schemas
export const wsMessageSchemas = {
  client: z.object({
    type: z.enum(['subscribe', 'unsubscribe', 'ping']),
    repository: repositorySchema.optional(),
    sessionId: sessionIdSchema.optional(),
    filters: z.record(z.unknown()).optional(),
  }),
  
  server: z.discriminatedUnion('type', [
    z.object({
      type: z.literal('memory_created'),
      chunk_id: idSchema,
      repository: repositorySchema,
      session_id: sessionIdSchema,
      content: z.string(),
      timestamp: timestampSchema,
    }),
    z.object({
      type: z.literal('memory_updated'),
      chunk_id: idSchema,
      repository: repositorySchema,
      changes: z.record(z.unknown()),
      timestamp: timestampSchema,
    }),
    z.object({
      type: z.literal('memory_deleted'),
      chunk_id: idSchema,
      repository: repositorySchema,
      timestamp: timestampSchema,
    }),
    z.object({
      type: z.literal('pattern_detected'),
      pattern_id: idSchema,
      repository: repositorySchema,
      pattern_type: z.string(),
      confidence: z.number(),
      timestamp: timestampSchema,
    }),
    z.object({
      type: z.literal('pong'),
      timestamp: timestampSchema,
    }),
    z.object({
      type: z.literal('error'),
      error: z.string(),
      code: z.string().optional(),
      timestamp: timestampSchema,
    }),
  ])
}

// Export type inference helpers
export type MemoryCreateInput = z.infer<typeof memorySchemas.create>
export type MemoryUpdateInput = z.infer<typeof memorySchemas.update>
export type MemorySearchInput = z.infer<typeof memorySchemas.search>
export type RepositoryAddInput = z.infer<typeof repositorySchemas.add>
export type RepositoryUpdateInput = z.infer<typeof repositorySchemas.update>
export type WSClientMessage = z.infer<typeof wsMessageSchemas.client>
export type WSServerMessage = z.infer<typeof wsMessageSchemas.server>

// Validation helpers
export function validateMemoryContent(content: string): { valid: boolean; error?: string } {
  if (!content || typeof content !== 'string') {
    return { valid: false, error: 'Content must be a non-empty string' }
  }
  
  if (content.length < 10) {
    return { valid: false, error: 'Content must be at least 10 characters long' }
  }
  
  if (content.length > 50000) {
    return { valid: false, error: 'Content must not exceed 50,000 characters' }
  }
  
  return { valid: true }
}

export function validateRepository(repo: string): { valid: boolean; error?: string } {
  if (!repo || typeof repo !== 'string') {
    return { valid: false, error: 'Repository must be a non-empty string' }
  }
  
  // Basic repository format validation
  const repoPattern = /^[a-zA-Z0-9.-]+\/[a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+$/
  if (!repoPattern.test(repo)) {
    return { valid: false, error: 'Invalid repository format. Expected: domain.com/owner/repo' }
  }
  
  return { valid: true }
}

export function validateTags(tags: string[]): { valid: boolean; error?: string } {
  if (!Array.isArray(tags)) {
    return { valid: false, error: 'Tags must be an array' }
  }
  
  if (tags.length > 20) {
    return { valid: false, error: 'Maximum 20 tags allowed' }
  }
  
  for (const tag of tags) {
    if (typeof tag !== 'string' || tag.length === 0) {
      return { valid: false, error: 'Each tag must be a non-empty string' }
    }
    
    if (tag.length > 50) {
      return { valid: false, error: 'Each tag must not exceed 50 characters' }
    }
    
    // Basic tag format validation (alphanumeric, hyphens, underscores)
    if (!/^[a-zA-Z0-9_-]+$/.test(tag)) {
      return { valid: false, error: 'Tags must contain only letters, numbers, hyphens, and underscores' }
    }
  }
  
  return { valid: true }
}

// Validation utilities for request validation
export function validateRequest<T>(schema: z.ZodSchema<T>, data: unknown): { success: true; data: T } | { success: false; errors: z.ZodIssue[] } {
  const result = schema.safeParse(data)
  if (result.success) {
    return { success: true, data: result.data }
  }
  return { success: false, errors: result.error.issues }
}

export function formatValidationErrors(errors: z.ZodIssue[]): { error: string; message: string; details: Array<{ path: string; message: string; code: string }> } {
  return {
    error: 'Validation Error',
    message: 'The request data is invalid',
    details: errors.map(err => ({
      path: err.path.join('.'),
      message: err.message,
      code: err.code,
    }))
  }
}