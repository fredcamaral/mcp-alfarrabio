import { NextRequest, NextResponse } from 'next/server'
import { validateCSRFTokenAsync } from '@/lib/csrf'
import { logger } from '@/lib/logger'
import { memorySchemas, validateRequest } from '@/lib/validation-schemas'
import apiClient from '@/lib/api-client'

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams
    const query = searchParams.get('query') || ''
    const repository = searchParams.get('repository') || undefined
    const limit = parseInt(searchParams.get('limit') || '50')
    const offset = parseInt(searchParams.get('offset') || '0')

    // Call the backend API
    const response = await apiClient.searchMemories({
      query,
      repository,
      limit,
      offset
    })

    return NextResponse.json(response)
  } catch (error) {
    logger.error('Failed to fetch memories', error, {
      component: 'MemoriesAPI',
      action: 'GET'
    })
    
    return NextResponse.json(
      { error: 'Failed to fetch memories' },
      { status: 500 }
    )
  }
}

export async function POST(request: NextRequest) {
  try {
    // Validate CSRF token
    const csrfToken = request.headers.get('X-CSRF-Token')
    if (!csrfToken || !await validateCSRFTokenAsync(csrfToken)) {
      return NextResponse.json(
        { error: 'Invalid CSRF token' },
        { status: 403 }
      )
    }

    const body = await request.json()
    
    // Validate request body
    const validation = validateRequest(memorySchemas.create, body)
    if (!validation.success) {
      return NextResponse.json(
        { error: 'Validation failed', details: validation.errors },
        { status: 400 }
      )
    }

    // Store the memory chunk
    const result = await apiClient.storeChunk({
      content: validation.data.content,
      type: validation.data.type,
      session_id: validation.data.sessionId || `session-${Date.now()}`,
      metadata: {
        repository: validation.data.repository,
        tags: validation.data.tags,
        extended_metadata: {
          priority: validation.data.priority,
          is_public: validation.data.isPublic
        }
      }
    })

    logger.info('Memory created successfully', {
      component: 'MemoriesAPI',
      action: 'POST',
      memoryId: result.chunk_id
    })

    return NextResponse.json({
      success: true,
      chunk_id: result.chunk_id,
      message: 'Memory created successfully'
    })
  } catch (error) {
    logger.error('Failed to create memory', error, {
      component: 'MemoriesAPI',
      action: 'POST'
    })
    
    return NextResponse.json(
      { error: 'Failed to create memory' },
      { status: 500 }
    )
  }
}

export async function PUT(request: NextRequest) {
  try {
    // Validate CSRF token
    const csrfToken = request.headers.get('X-CSRF-Token')
    if (!csrfToken || !await validateCSRFTokenAsync(csrfToken)) {
      return NextResponse.json(
        { error: 'Invalid CSRF token' },
        { status: 403 }
      )
    }

    const body = await request.json()
    
    // Validate request body
    const validation = validateRequest(memorySchemas.update, body)
    if (!validation.success) {
      return NextResponse.json(
        { error: 'Validation failed', details: validation.errors },
        { status: 400 }
      )
    }

    // TODO: Implement update logic when backend supports it
    logger.warn('Memory update not yet implemented', {
      component: 'MemoriesAPI',
      action: 'PUT',
      memoryId: validation.data.id
    })

    return NextResponse.json(
      { error: 'Update functionality not yet implemented' },
      { status: 501 }
    )
  } catch (error) {
    logger.error('Failed to update memory', error, {
      component: 'MemoriesAPI',
      action: 'PUT'
    })
    
    return NextResponse.json(
      { error: 'Failed to update memory' },
      { status: 500 }
    )
  }
}

export async function DELETE(request: NextRequest) {
  try {
    // Validate CSRF token
    const csrfToken = request.headers.get('X-CSRF-Token')
    if (!csrfToken || !await validateCSRFTokenAsync(csrfToken)) {
      return NextResponse.json(
        { error: 'Invalid CSRF token' },
        { status: 403 }
      )
    }

    const searchParams = request.nextUrl.searchParams
    const id = searchParams.get('id')
    
    if (!id) {
      return NextResponse.json(
        { error: 'Memory ID is required' },
        { status: 400 }
      )
    }

    // TODO: Implement delete logic when backend supports it
    logger.warn('Memory delete not yet implemented', {
      component: 'MemoriesAPI',
      action: 'DELETE',
      memoryId: id
    })

    return NextResponse.json(
      { error: 'Delete functionality not yet implemented' },
      { status: 501 }
    )
  } catch (error) {
    logger.error('Failed to delete memory', error, {
      component: 'MemoriesAPI',
      action: 'DELETE'
    })
    
    return NextResponse.json(
      { error: 'Failed to delete memory' },
      { status: 500 }
    )
  }
}