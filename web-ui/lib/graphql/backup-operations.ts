/**
 * GraphQL Backup Operations
 * 
 * Queries and mutations for backup management
 */

import { gql } from '@apollo/client'

// Types
export interface Backup {
  id: string
  version: string
  createdAt: string
  chunkCount: number
  size: number
  repository?: string
  metadata?: {
    backupFile: string
    compression?: string
    format?: string
  }
}

export interface BackupResult {
  success: boolean
  message: string
  backupId?: string
  restoredCount?: number
}

export interface CleanupResult {
  success: boolean
  deletedCount: number
  freedSpace: number
}

// Queries
export const LIST_BACKUPS = gql`
  query ListBackups($repository: String) {
    listBackups(repository: $repository) {
      id
      version
      createdAt
      chunkCount
      size
      repository
      metadata {
        backupFile
        compression
        format
      }
    }
  }
`

// Mutations
export const CREATE_BACKUP = gql`
  mutation CreateBackup(
    $name: String
    $repository: String
    $includeVectors: Boolean
    $format: String
    $compress: Boolean
  ) {
    createBackup(
      name: $name
      repository: $repository
      includeVectors: $includeVectors
      format: $format
      compress: $compress
    ) {
      id
      version
      createdAt
      chunkCount
      size
      repository
      metadata {
        backupFile
        compression
        format
      }
    }
  }
`

export const RESTORE_BACKUP = gql`
  mutation RestoreBackup($backupFile: String!, $overwrite: Boolean!) {
    restoreBackup(backupFile: $backupFile, overwrite: $overwrite) {
      success
      message
      restoredCount
    }
  }
`

export const CLEANUP_BACKUPS = gql`
  mutation CleanupBackups($retentionDays: Int) {
    cleanupBackups(retentionDays: $retentionDays) {
      success
      deletedCount
      freedSpace
    }
  }
`