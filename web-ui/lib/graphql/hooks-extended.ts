/**
 * Extended GraphQL Hooks
 * 
 * Additional hooks for backup, config, and system operations
 */

import { useQuery, useMutation } from '@apollo/client'
import type { QueryResult } from '@apollo/client'
import { logger } from '@/lib/logger'

// Import backup operations
import {
  LIST_BACKUPS,
  CREATE_BACKUP,
  RESTORE_BACKUP,
  CLEANUP_BACKUPS,
  type Backup,
  type BackupResult,
  type CleanupResult
} from './backup-operations'

// Import config operations
import {
  GET_CONFIG,
  UPDATE_CONFIG,
  type SystemConfig,
  type ConfigInput
} from './config-operations'

// Import system operations
import {
  HEALTH_CHECK,
  SYSTEM_STATUS,
  EXPORT_MEMORIES,
  IMPORT_MEMORIES,
  CREATE_SESSION,
  END_SESSION,
  type HealthStatus,
  type SystemStatus,
  type ExportResult,
  type ImportResult,
  type Session,
  type SessionResult
} from './system-operations'

// BACKUP HOOKS

export function useListBackups(repository?: string): QueryResult<{ listBackups: Backup[] }> {
  return useQuery(LIST_BACKUPS, {
    variables: { repository },
    errorPolicy: 'all',
    notifyOnNetworkStatusChange: true,
    onError: (error) => {
      logger.error('Failed to list backups', { error })
    }
  })
}

export function useCreateBackup() {
  return useMutation<
    { createBackup: Backup },
    {
      name?: string
      repository?: string
      includeVectors?: boolean
      format?: string
      compress?: boolean
    }
  >(CREATE_BACKUP, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to create backup', { error })
    }
  })
}

export function useRestoreBackup() {
  return useMutation<
    { restoreBackup: BackupResult },
    { backupFile: string; overwrite: boolean }
  >(RESTORE_BACKUP, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to restore backup', { error })
    }
  })
}

export function useCleanupBackups() {
  return useMutation<
    { cleanupBackups: CleanupResult },
    { retentionDays?: number }
  >(CLEANUP_BACKUPS, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to cleanup backups', { error })
    }
  })
}

// CONFIG HOOKS

export function useGetConfig(): QueryResult<{ getConfig: SystemConfig }> {
  return useQuery(GET_CONFIG, {
    errorPolicy: 'all',
    notifyOnNetworkStatusChange: true,
    onError: (error) => {
      logger.error('Failed to get config', { error })
    }
  })
}

export function useUpdateConfig() {
  return useMutation<
    { updateConfig: SystemConfig },
    { input: ConfigInput }
  >(UPDATE_CONFIG, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to update config', { error })
    }
  })
}

// SYSTEM HOOKS

export function useHealthCheck(): QueryResult<{ health: HealthStatus }> {
  return useQuery(HEALTH_CHECK, {
    errorPolicy: 'all',
    pollInterval: 30000, // Poll every 30 seconds
    notifyOnNetworkStatusChange: true,
    onError: (error) => {
      logger.error('Health check failed', { error })
    }
  })
}

export function useSystemStatus(): QueryResult<{ systemStatus: SystemStatus }> {
  return useQuery(SYSTEM_STATUS, {
    errorPolicy: 'all',
    pollInterval: 60000, // Poll every minute
    notifyOnNetworkStatusChange: true,
    onError: (error) => {
      logger.error('Failed to get system status', { error })
    }
  })
}

// EXPORT/IMPORT HOOKS

export function useExportMemories() {
  return useMutation<
    { exportMemories: ExportResult },
    { repository: string; format: string }
  >(EXPORT_MEMORIES, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to export memories', { error })
    }
  })
}

export function useImportMemories() {
  return useMutation<
    { importMemories: ImportResult },
    { data: string; repository: string; sessionId: string }
  >(IMPORT_MEMORIES, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to import memories', { error })
    }
  })
}

// SESSION HOOKS

export function useCreateSession() {
  return useMutation<
    { createSession: Session },
    { sessionId: string; repository: string }
  >(CREATE_SESSION, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to create session', { error })
    }
  })
}

export function useEndSession() {
  return useMutation<
    { endSession: SessionResult },
    { sessionId: string; repository: string }
  >(END_SESSION, {
    errorPolicy: 'all',
    onError: (error) => {
      logger.error('Failed to end session', { error })
    }
  })
}

// Re-export types for convenience
export type {
  Backup,
  BackupResult,
  CleanupResult,
  SystemConfig,
  ConfigInput,
  HealthStatus,
  SystemStatus,
  ExportResult,
  ImportResult,
  Session,
  SessionResult
}