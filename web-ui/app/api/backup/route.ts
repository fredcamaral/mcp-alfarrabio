/**
 * Backup API Endpoints
 * 
 * Provides backup and restore functionality for memory data.
 * Integrates with the Go backend persistence layer.
 */

import { NextRequest, NextResponse } from 'next/server'
import { logger } from '@/lib/logger'

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'

/**
 * GET /api/backup
 * 
 * Lists available backups with metadata
 */
export async function GET() {
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
        { error: 'Backup Error', message: errorData.error || 'Failed to list backups' },
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
    logger.error('Error listing backups', { error })
    return NextResponse.json(
      { error: 'Server Error', message: 'Internal server error' },
      { status: 500 }
    )
  }
}

/**
 * POST /api/backup
 * 
 * Creates a new backup of memory data
 */
export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { repository, name, includeVectors, format, compress } = body || {}

    const response = await fetch(`${BACKEND_URL}/api/backup/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        repository: repository || '',
        name: name || `backup_${new Date().toISOString().split('T')[0]}`,
        includeVectors,
        format,
        compress
      }),
    })

    if (!response.ok) {
      const errorData = await response.json()
      return NextResponse.json(
        { error: 'Backup Error', message: errorData.error || 'Failed to create backup' },
        { status: response.status }
      )
    }

    const result = await response.json()

    logger.info('Backup created', { repository, name: result.name })

    return NextResponse.json({
      ...result,
      message: 'Backup created successfully'
    })
  } catch (error) {
    logger.error('Error creating backup', { error })
    return NextResponse.json(
      { error: 'Server Error', message: 'Internal server error' },
      { status: 500 }
    )
  }
}

/**
 * PUT /api/backup
 * 
 * Restores data from a backup
 */
export async function PUT(request: NextRequest) {
  try {
    const body = await request.json()
    const { file, overwrite, validateIntegrity } = body || {}

    if (!file) {
      return NextResponse.json(
        { error: 'Validation Error', message: 'Backup file is required' },
        { status: 400 }
      )
    }

    const response = await fetch(`${BACKEND_URL}/api/backup/restore`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        backup_file: file,
        overwrite,
        validate_integrity: validateIntegrity
      }),
    })

    if (!response.ok) {
      const errorData = await response.json()
      return NextResponse.json(
        { error: 'Restore Error', message: errorData.error || 'Failed to restore backup' },
        { status: response.status }
      )
    }

    const result = await response.json()

    logger.info('Backup restored', { file })

    return NextResponse.json({
      ...result,
      message: 'Backup restored successfully'
    })
  } catch (error) {
    logger.error('Error restoring backup', { error })
    return NextResponse.json(
      { error: 'Server Error', message: 'Internal server error' },
      { status: 500 }
    )
  }
}

/**
 * PATCH /api/backup
 * 
 * Cleanup old backups based on retention policy
 */
export async function PATCH() {
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
        { error: 'Cleanup Error', message: errorData.error || 'Failed to cleanup backups' },
        { status: response.status }
      )
    }

    const result = await response.json()

    logger.info('Backup cleanup completed', result)

    return NextResponse.json({
      ...result,
      message: 'Backup cleanup completed'
    })
  } catch (error) {
    logger.error('Error cleaning up backups', { error })
    return NextResponse.json(
      { error: 'Server Error', message: 'Internal server error' },
      { status: 500 }
    )
  }
}