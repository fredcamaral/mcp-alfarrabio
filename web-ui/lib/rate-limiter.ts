import { LRUCache } from 'lru-cache'
import { NextRequest } from 'next/server'
import { logger } from './logger'

export interface RateLimitConfig {
  interval: number // Time window in milliseconds
  uniqueTokenPerInterval: number // Max number of unique tokens per interval
}

export interface RateLimitResult {
  success: boolean
  limit: number
  remaining: number
  reset: number
}

// Default rate limit configurations
export const RATE_LIMITS = {
  // API routes
  api: {
    interval: 60 * 1000, // 1 minute
    uniqueTokenPerInterval: 30, // 30 requests per minute
  },
  // GraphQL endpoint
  graphql: {
    interval: 60 * 1000, // 1 minute
    uniqueTokenPerInterval: 100, // 100 requests per minute
  },
  // Mutations/writes
  mutations: {
    interval: 60 * 1000, // 1 minute
    uniqueTokenPerInterval: 10, // 10 write operations per minute
  },
  // File uploads
  uploads: {
    interval: 60 * 1000, // 1 minute
    uniqueTokenPerInterval: 5, // 5 uploads per minute
  },
  // Authentication
  auth: {
    interval: 15 * 60 * 1000, // 15 minutes
    uniqueTokenPerInterval: 5, // 5 attempts per 15 minutes
  },
} as const

// Create rate limiter instance
export function createRateLimiter(config: RateLimitConfig) {
  const tokenCache = new LRUCache<string, number[]>({
    max: config.uniqueTokenPerInterval,
    ttl: config.interval,
  })

  return {
    check: (token: string): RateLimitResult => {
      const now = Date.now()
      const timestamps = tokenCache.get(token) || []
      
      // Remove timestamps outside the current interval
      const validTimestamps = timestamps.filter(
        timestamp => now - timestamp < config.interval
      )

      if (validTimestamps.length >= config.uniqueTokenPerInterval) {
        logger.warn('Rate limit exceeded', {
          component: 'RateLimiter',
          token,
          limit: config.uniqueTokenPerInterval,
          current: validTimestamps.length,
        })

        return {
          success: false,
          limit: config.uniqueTokenPerInterval,
          remaining: 0,
          reset: Math.min(...validTimestamps) + config.interval,
        }
      }

      // Add current timestamp
      validTimestamps.push(now)
      tokenCache.set(token, validTimestamps)

      return {
        success: true,
        limit: config.uniqueTokenPerInterval,
        remaining: config.uniqueTokenPerInterval - validTimestamps.length,
        reset: now + config.interval,
      }
    },
  }
}

// Get identifier from request
export function getRequestIdentifier(request: NextRequest): string {
  // Try to get user ID from auth token
  const authHeader = request.headers.get('authorization')
  if (authHeader) {
    const token = authHeader.replace('Bearer ', '')
    // In production, decode JWT to get user ID
    // For now, use token as identifier
    return `auth:${token.substring(0, 16)}`
  }

  // Fall back to IP address
  const forwardedFor = request.headers.get('x-forwarded-for')
  const realIp = request.headers.get('x-real-ip')
  const ip = forwardedFor?.split(',')[0] || realIp || 'unknown'
  
  return `ip:${ip}`
}

// Rate limit middleware
export async function rateLimit(
  request: NextRequest,
  config: RateLimitConfig = RATE_LIMITS.api
): Promise<RateLimitResult | null> {
  try {
    const identifier = getRequestIdentifier(request)
    const limiter = createRateLimiter(config)
    const result = limiter.check(identifier)

    if (!result.success) {
      logger.info('Rate limit applied', {
        component: 'RateLimiter',
        identifier,
        path: request.nextUrl.pathname,
        remaining: result.remaining,
      })
    }

    return result
  } catch (error) {
    logger.error('Rate limiter error', error as Error, {
      component: 'RateLimiter',
      path: request.nextUrl.pathname,
    })
    
    // Fail open in case of errors
    return null
  }
}

// Helper to create rate limit response
export function createRateLimitResponse(result: RateLimitResult) {
  return new Response(
    JSON.stringify({
      error: 'Too Many Requests',
      message: 'Rate limit exceeded. Please try again later.',
      retryAfter: Math.ceil((result.reset - Date.now()) / 1000),
    }),
    {
      status: 429,
      headers: {
        'Content-Type': 'application/json',
        'X-RateLimit-Limit': result.limit.toString(),
        'X-RateLimit-Remaining': result.remaining.toString(),
        'X-RateLimit-Reset': result.reset.toString(),
        'Retry-After': Math.ceil((result.reset - Date.now()) / 1000).toString(),
      },
    }
  )
}