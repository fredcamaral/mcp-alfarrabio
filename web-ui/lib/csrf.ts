/**
 * CSRF Protection Server-side Utilities
 * 
 * Provides server-side CSRF token generation, validation, and form protection for the WebUI.
 * Uses cryptographically secure random tokens with double-submit cookie pattern.
 */

import { cookies } from 'next/headers'
import { NextRequest, NextResponse } from 'next/server'

export const CSRF_TOKEN_NAME = 'csrf-token'
export const CSRF_HEADER_NAME = 'X-CSRF-Token'
export const CSRF_COOKIE_NAME = '__csrf-token'

/**
 * Generate a cryptographically secure CSRF token
 * Uses Web Crypto API which is available in both browser and Edge Runtime
 */
export function generateCSRFToken(): string {
  // Use crypto global which is available in Edge Runtime
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('')
}

/**
 * Set CSRF token in cookies (server-side)
 */
export async function setCSRFToken(): Promise<string> {
  const token = generateCSRFToken()
  const cookieStore = await cookies()
  
  cookieStore.set(CSRF_COOKIE_NAME, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 60 * 60 * 24, // 24 hours
    path: '/'
  })
  
  return token
}

/**
 * Set CSRF token in response (for middleware use)
 */
export function setCSRFTokenInResponse(response: NextResponse): string {
  const token = generateCSRFToken()
  
  response.cookies.set(CSRF_COOKIE_NAME, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 60 * 60 * 24, // 24 hours
    path: '/'
  })
  
  return token
}

/**
 * Get CSRF token from cookies (server-side)
 */
export async function getCSRFToken(): Promise<string | null> {
  const cookieStore = await cookies()
  return cookieStore.get(CSRF_COOKIE_NAME)?.value || null
}

/**
 * Validate CSRF token from request or directly from token string
 */
export function validateCSRFToken(requestOrToken: NextRequest | string): boolean {
  if (typeof requestOrToken === 'string') {
    // Direct token validation - used in API routes
    // This is an async function that needs to get cookie value
    return false // Can't validate directly without cookie access
  }
  
  const request = requestOrToken
  const cookieToken = request.cookies.get(CSRF_COOKIE_NAME)?.value
  const headerToken = request.headers.get(CSRF_HEADER_NAME)
  const formToken = request.headers.get('content-type')?.includes('application/x-www-form-urlencoded')
    ? null // Will be extracted from form data in specific handlers
    : null

  if (!cookieToken) {
    return false
  }

  // Check header token first
  if (headerToken && headerToken === cookieToken) {
    return true
  }

  // For form submissions, token should be in form data
  // Note: formToken is not used in the current implementation
  // But keeping the variable for future form data extraction
  if (formToken) {
    return formToken === cookieToken
  }
  
  return false
}

/**
 * Validate CSRF token directly (async version for API routes)
 */
export async function validateCSRFTokenAsync(token: string): Promise<boolean> {
  const cookieStore = await cookies()
  const cookieToken = cookieStore.get(CSRF_COOKIE_NAME)?.value
  
  if (!cookieToken || !token) {
    return false
  }
  
  return token === cookieToken
}

/**
 * CSRF Protection Middleware
 */
export function csrfProtection(handler: (req: NextRequest) => Promise<NextResponse> | NextResponse) {
  return async (request: NextRequest): Promise<NextResponse> => {
    // Skip CSRF protection for GET, HEAD, OPTIONS requests
    if (['GET', 'HEAD', 'OPTIONS'].includes(request.method)) {
      return handler(request)
    }

    // Skip CSRF protection for API routes with API key authentication
    const apiKey = request.headers.get('Authorization')
    if (apiKey && apiKey.startsWith('Bearer ')) {
      return handler(request)
    }

    // Validate CSRF token
    if (!validateCSRFToken(request)) {
      return NextResponse.json(
        { 
          error: 'CSRF token validation failed',
          code: 'CSRF_TOKEN_INVALID'
        },
        { status: 403 }
      )
    }

    return handler(request)
  }
}


/**
 * Extract CSRF token from form data (server-side)
 */
export async function extractCSRFTokenFromForm(request: NextRequest): Promise<string | null> {
  try {
    const formData = await request.formData()
    return formData.get(CSRF_TOKEN_NAME) as string || null
  } catch {
    return null
  }
}

/**
 * Validate CSRF token from form data
 */
export async function validateCSRFTokenFromForm(request: NextRequest): Promise<boolean> {
  const cookieToken = request.cookies.get(CSRF_COOKIE_NAME)?.value
  const formToken = await extractCSRFTokenFromForm(request)

  if (!cookieToken || !formToken) {
    return false
  }

  return cookieToken === formToken
}