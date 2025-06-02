/**
 * CSRF Protection Utilities
 * 
 * Provides CSRF token generation, validation, and form protection for the WebUI.
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
 * Client-side CSRF token management
 */
export class CSRFManager {
  private static token: string | null = null
  private static listeners: Set<(token: string | null) => void> = new Set()

  /**
   * Get the current CSRF token
   */
  static getToken(): string | null {
    return this.token
  }

  /**
   * Set the CSRF token
   */
  static setToken(token: string | null): void {
    this.token = token
    this.listeners.forEach(listener => listener(token))
  }

  /**
   * Fetch CSRF token from server
   */
  static async fetchToken(): Promise<string | null> {
    try {
      const response = await fetch('/api/csrf-token', {
        method: 'GET',
        credentials: 'include'
      })

      if (response.ok) {
        const data = await response.json()
        this.setToken(data.token)
        return data.token
      }
    } catch (error) {
      console.error('Failed to fetch CSRF token:', error)
    }

    return null
  }

  /**
   * Add listener for token changes
   */
  static addListener(listener: (token: string | null) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  /**
   * Get headers with CSRF token
   */
  static getHeaders(additionalHeaders: Record<string, string> = {}): Record<string, string> {
    const headers: Record<string, string> = {
      ...additionalHeaders
    }

    if (this.token) {
      headers[CSRF_HEADER_NAME] = this.token
    }

    return headers
  }

  /**
   * Make a protected request with CSRF token
   */
  static async request(url: string, options: RequestInit = {}): Promise<Response> {
    const token = this.getToken() || await this.fetchToken()
    
    if (!token) {
      throw new Error('No CSRF token available')
    }

    const headers = this.getHeaders(options.headers as Record<string, string> || {})

    return fetch(url, {
      ...options,
      credentials: 'include',
      headers
    })
  }
}

/**
 * Form data helper that includes CSRF token
 */
export function createProtectedFormData(data: Record<string, any>): FormData {
  const formData = new FormData()
  const token = CSRFManager.getToken()

  if (token) {
    formData.append(CSRF_TOKEN_NAME, token)
  }

  Object.entries(data).forEach(([key, value]) => {
    if (value !== null && value !== undefined) {
      if (value instanceof File || value instanceof Blob) {
        formData.append(key, value)
      } else {
        formData.append(key, String(value))
      }
    }
  })

  return formData
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