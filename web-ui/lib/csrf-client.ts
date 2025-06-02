/**
 * CSRF Protection Client-side Utilities
 * 
 * Provides client-side CSRF token management for the WebUI.
 * Uses cryptographically secure random tokens with double-submit cookie pattern.
 */

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