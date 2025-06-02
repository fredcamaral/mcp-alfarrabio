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
 */
export function generateCSRFToken(): string {
  if (typeof window !== 'undefined' && window.crypto && window.crypto.getRandomValues) {
    // Browser environment
    const array = new Uint8Array(32)
    window.crypto.getRandomValues(array)
    return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('')
  } else {
    // Node.js environment
    const crypto = require('crypto')
    return crypto.randomBytes(32).toString('hex')
  }
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
 * Validate CSRF token from request
 */
export function validateCSRFToken(request: NextRequest): boolean {
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
  return false
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
  } catch (error) {
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