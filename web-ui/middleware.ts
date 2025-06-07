/**
 * Next.js Middleware for CSRF Protection
 * 
 * Automatically protects API routes and form submissions from CSRF attacks.
 * Implements double-submit cookie pattern with secure token validation.
 */

import { NextRequest, NextResponse } from 'next/server'
import { validateCSRFToken, validateCSRFTokenFromForm, setCSRFTokenInResponse } from '@/lib/csrf'

// Routes that require CSRF protection
const PROTECTED_ROUTES = [
  '/api/backup',
  '/api/memories',
  '/api/search',
  '/api/chunks',
  '/api/sessions',
  '/api/exports',
  '/api/imports'
]

// Routes that are exempt from CSRF protection
const EXEMPT_ROUTES = [
  '/api/csrf-token',
  '/api/health',
  '/api/status',
  '/api/errors',
  '/api/performance',
  '/api/mcp',  // MCP proxy doesn't need CSRF as the backend handles security
  '/health'
]


/**
 * Check if a route requires CSRF protection
 */
function requiresCSRFProtection(pathname: string): boolean {
  // Exempt routes are never protected
  if (EXEMPT_ROUTES.some(route => pathname.startsWith(route))) {
    return false
  }

  // Protected routes always require protection
  if (PROTECTED_ROUTES.some(route => pathname.startsWith(route))) {
    return true
  }

  // Default: protect all API routes except health/status
  return pathname.startsWith('/api/')
}

/**
 * Check if request method requires CSRF protection
 */
function isProtectedMethod(method: string): boolean {
  return ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method.toUpperCase())
}


/**
 * Main middleware function
 */
export async function middleware(request: NextRequest): Promise<NextResponse> {
  const { pathname } = request.nextUrl
  const method = request.method

  // Skip CSRF protection for safe methods or exempt routes
  if (!isProtectedMethod(method) || !requiresCSRFProtection(pathname)) {
    return NextResponse.next()
  }

  // Ensure CSRF token exists in cookies for protected routes
  const existingToken = request.cookies.get('__csrf-token')?.value
  if (!existingToken) {
    // For API requests without token, return 403
    if (pathname.startsWith('/api/')) {
      return NextResponse.json(
        { 
          error: 'CSRF token required',
          code: 'CSRF_TOKEN_MISSING'
        },
        { status: 403 }
      )
    }
    
    // For page requests, generate and set a new token
    const response = NextResponse.next()
    setCSRFTokenInResponse(response)
    return response
  }

  // Validate CSRF token for protected requests
  const contentType = request.headers.get('content-type')
  let isValid = false

  if (contentType?.includes('application/x-www-form-urlencoded') || 
      contentType?.includes('multipart/form-data')) {
    // Validate token from form data
    isValid = await validateCSRFTokenFromForm(request)
  } else {
    // Validate token from headers
    isValid = validateCSRFToken(request)
  }

  if (!isValid) {
    return NextResponse.json(
      { 
        error: 'CSRF token validation failed',
        code: 'CSRF_TOKEN_INVALID',
        details: {
          method,
          pathname,
          contentType,
          hasToken: !!existingToken,
          hasHeader: !!request.headers.get('X-CSRF-Token')
        }
      },
      { status: 403 }
    )
  }

  // Token is valid, proceed with request
  return NextResponse.next()
}

/**
 * Configure which routes the middleware should run on
 */
export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder files
     */
    '/((?!_next/static|_next/image|favicon.ico|public/).*)',
  ],
}