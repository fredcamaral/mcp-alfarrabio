import { NextRequest, NextResponse } from 'next/server'
import { z } from 'zod'
import { rateLimit, RATE_LIMITS, createRateLimitResponse, RateLimitConfig } from './rate-limiter'
import { validateRequest, formatValidationErrors } from './validation-schemas'
import { logger } from './logger'

export interface ApiHandlerOptions<T = unknown> {
  // Rate limiting
  rateLimit?: RateLimitConfig | keyof typeof RATE_LIMITS
  
  // Validation
  bodySchema?: z.ZodSchema<T>
  querySchema?: z.ZodSchema
  
  // Authentication
  requireAuth?: boolean
  
  // CORS
  allowedOrigins?: string[]
  
  // Method restrictions
  allowedMethods?: string[]
}

export type ApiHandler<T = unknown> = (
  request: NextRequest,
  context: {
    params?: Record<string, string>
    body?: T
    query?: Record<string, string>
    user?: { token: string; id?: string; email?: string } // User from auth
  }
) => Promise<Response> | Response

// Main middleware wrapper
export function withApiMiddleware<T = unknown>(
  handler: ApiHandler<T>,
  options: ApiHandlerOptions<T> = {}
) {
  return async (request: NextRequest, context?: { params?: Record<string, string> }): Promise<Response> => {
    try {
      // Method check
      if (options.allowedMethods && !options.allowedMethods.includes(request.method)) {
        return NextResponse.json(
          { error: 'Method Not Allowed' },
          { status: 405 }
        )
      }

      // CORS handling
      if (options.allowedOrigins) {
        const origin = request.headers.get('origin')
        if (origin && !options.allowedOrigins.includes(origin)) {
          return NextResponse.json(
            { error: 'CORS Error', message: 'Origin not allowed' },
            { status: 403 }
          )
        }
      }

      // Rate limiting
      if (options.rateLimit) {
        const rateLimitConfig = typeof options.rateLimit === 'string' 
          ? RATE_LIMITS[options.rateLimit]
          : options.rateLimit

        const rateLimitResult = await rateLimit(request, rateLimitConfig)
        if (rateLimitResult && !rateLimitResult.success) {
          return createRateLimitResponse(rateLimitResult)
        }
      }

      // Authentication check
      if (options.requireAuth) {
        const authHeader = request.headers.get('authorization')
        if (!authHeader || !authHeader.startsWith('Bearer ')) {
          return NextResponse.json(
            { error: 'Unauthorized', message: 'Authentication required' },
            { status: 401 }
          )
        }

        // TODO: Validate JWT token and extract user
        // For now, just pass through
        const token = authHeader.replace('Bearer ', '')
        const extendedContext = { ...context, user: { token } }
        context = extendedContext as typeof context
      }

      // Parse request data
      let body: T | undefined
      let query: Record<string, string> = {}

      // Parse query parameters
      const { searchParams } = new URL(request.url)
      searchParams.forEach((value, key) => {
        query[key] = value
      })

      // Parse body for POST/PUT/PATCH
      if (['POST', 'PUT', 'PATCH'].includes(request.method)) {
        try {
          body = await request.json()
        } catch {
          return NextResponse.json(
            { error: 'Bad Request', message: 'Invalid JSON body' },
            { status: 400 }
          )
        }
      }

      // Validate query parameters
      if (options.querySchema) {
        const validation = validateRequest(options.querySchema, query)
        if (!validation.success) {
          return NextResponse.json(
            formatValidationErrors(validation.errors),
            { status: 400 }
          )
        }
        query = validation.data
      }

      // Validate body
      if (options.bodySchema && body !== undefined) {
        const validation = validateRequest(options.bodySchema, body)
        if (!validation.success) {
          return NextResponse.json(
            formatValidationErrors(validation.errors),
            { status: 400 }
          )
        }
        body = validation.data
      }

      // Log API request
      logger.info('API request', {
        component: 'ApiMiddleware',
        method: request.method,
        path: request.nextUrl.pathname,
        hasBody: !!body,
        hasQuery: Object.keys(query).length > 0,
      })

      // Call the actual handler
      const response = await handler(request, {
        ...context,
        body,
        query,
      })

      // Log response status
      logger.info('API response', {
        component: 'ApiMiddleware',
        method: request.method,
        path: request.nextUrl.pathname,
        status: response.status,
      })

      return response

    } catch (error) {
      // Log error
      logger.error('API handler error', error as Error, {
        component: 'ApiMiddleware',
        method: request.method,
        path: request.nextUrl.pathname,
      })

      // Return generic error response
      return NextResponse.json(
        {
          error: 'Internal Server Error',
          message: process.env.NODE_ENV === 'development' 
            ? (error as Error).message 
            : 'An unexpected error occurred',
        },
        { status: 500 }
      )
    }
  }
}

// Convenience wrappers for common patterns
export const withPublicApi = <T = unknown>(
  handler: ApiHandler<T>,
  options: Omit<ApiHandlerOptions<T>, 'requireAuth'> = {}
) => withApiMiddleware(handler, { ...options, rateLimit: options.rateLimit || 'api' })

export const withAuthenticatedApi = <T = unknown>(
  handler: ApiHandler<T>,
  options: Omit<ApiHandlerOptions<T>, 'requireAuth'> = {}
) => withApiMiddleware(handler, { ...options, requireAuth: true, rateLimit: options.rateLimit || 'api' })

export const withGraphQLApi = <T = unknown>(
  handler: ApiHandler<T>,
  options: Omit<ApiHandlerOptions<T>, 'rateLimit'> = {}
) => withApiMiddleware(handler, { ...options, rateLimit: 'graphql' })

// Helper to create JSON responses with consistent formatting
export function createApiResponse<T>(
  data: T,
  options: {
    status?: number
    headers?: Record<string, string>
    meta?: Record<string, unknown>
  } = {}
) {
  const { status = 200, headers = {}, meta } = options

  const responseData = meta ? { data, meta } : data

  return NextResponse.json(responseData, {
    status,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
  })
}

// Helper for error responses
export function createErrorResponse(
  error: string,
  message: string,
  status = 400,
  details?: unknown
) {
  return NextResponse.json(
    {
      error,
      message,
      ...(details ? { details } : {}),
    },
    { status }
  )
}