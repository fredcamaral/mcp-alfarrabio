/**
 * Backup API Endpoints
 * 
 * Provides backup and restore functionality for memory data.
 * Integrates with the Go backend persistence layer.
 */

import { NextRequest, NextResponse } from 'next/server'
import { csrfProtection } from '@/lib/csrf'

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'

/**
 * GET /api/backup
 * 
 * Lists available backups with metadata
 */
export async function GET(request: NextRequest): Promise<NextResponse> {
  try {
    const response = await fetch(`${BACKEND_URL}/api/backup/list`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      const errorData = await response.json()
      return NextResponse.json(
        { error: errorData.error || 'Failed to list backups' },
        { status: response.status }
      )
    }

    const backups = await response.json()
    
    return NextResponse.json({
      backups,
      count: backups.length,
      status: 'success'
    })
  } catch (error) {
    console.error('Error listing backups:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

/**
 * POST /api/backup
 * 
 * Creates a new backup of memory data
 */
async function handleCreateBackup(request: NextRequest): Promise<NextResponse> {
  try {
    const body = await request.json()
    const { repository, name } = body

    const response = await fetch(`${BACKEND_URL}/api/backup/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        repository: repository || '',
        name: name || `backup_${new Date().toISOString().split('T')[0]}`
      }),
    })

    if (!response.ok) {
      const errorData = await response.json()
      return NextResponse.json(
        { error: errorData.error || 'Failed to create backup' },
        { status: response.status }
      )
    }

    const result = await response.json()
    
    return NextResponse.json({
      ...result,
      message: 'Backup created successfully'
    })
  } catch (error) {
    console.error('Error creating backup:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

/**
 * PUT /api/backup
 * 
 * Restores data from a backup
 */
async function handleRestoreBackup(request: NextRequest): Promise<NextResponse> {
  try {
    const body = await request.json()
    const { backupFile, overwrite = false } = body

    if (!backupFile) {
      return NextResponse.json(
        { error: 'Backup file is required' },
        { status: 400 }
      )
    }

    const response = await fetch(`${BACKEND_URL}/api/backup/restore`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        backup_file: backupFile,
        overwrite
      }),
    })

    if (!response.ok) {
      const errorData = await response.json()
      return NextResponse.json(
        { error: errorData.error || 'Failed to restore backup' },
        { status: response.status }
      )
    }

    const result = await response.json()
    
    return NextResponse.json({
      ...result,
      message: 'Backup restored successfully'
    })
  } catch (error) {
    console.error('Error restoring backup:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

/**
 * PATCH /api/backup
 * 
 * Cleanup old backups based on retention policy
 */
async function handleCleanupBackups(request: NextRequest): Promise<NextResponse> {
  try {
    const response = await fetch(`${BACKEND_URL}/api/backup/cleanup`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      const errorData = await response.json()
      return NextResponse.json(
        { error: errorData.error || 'Failed to cleanup backups' },
        { status: response.status }
      )
    }

    const result = await response.json()
    
    return NextResponse.json({
      ...result,
      message: 'Backup cleanup completed'
    })
  } catch (error) {
    console.error('Error cleaning up backups:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export const POST = csrfProtection(handleCreateBackup)
export const PUT = csrfProtection(handleRestoreBackup)
export const PATCH = csrfProtection(handleCleanupBackups)