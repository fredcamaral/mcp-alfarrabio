/**
 * CSRF Protection React Hook
 * 
 * Provides easy integration of CSRF protection in React components.
 * Automatically fetches and manages CSRF tokens for forms and API calls.
 */

import React, { useState, useEffect, useCallback } from 'react'
import { CSRFManager } from '@/lib/csrf'

interface UseCSRFProtectionOptions {
  autoFetch?: boolean
  refreshInterval?: number
}

interface UseCSRFProtectionReturn {
  token: string | null
  isLoading: boolean
  error: string | null
  refreshToken: () => Promise<void>
  makeProtectedRequest: (url: string, options?: RequestInit) => Promise<Response>
  getFormHeaders: () => Record<string, string>
  isTokenValid: boolean
}

/**
 * Hook for managing CSRF protection in React components
 */
export function useCSRFProtection(options: UseCSRFProtectionOptions = {}): UseCSRFProtectionReturn {
  const { autoFetch = true, refreshInterval } = options
  
  const [token, setToken] = useState<string | null>(CSRFManager.getToken())
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  /**
   * Fetch or refresh the CSRF token
   */
  const refreshToken = useCallback(async (): Promise<void> => {
    setIsLoading(true)
    setError(null)

    try {
      const newToken = await CSRFManager.fetchToken()
      
      if (newToken) {
        setToken(newToken)
      } else {
        throw new Error('Failed to fetch CSRF token')
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred'
      setError(errorMessage)
      console.error('CSRF token refresh failed:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  /**
   * Make a protected API request with CSRF token
   */
  const makeProtectedRequest = useCallback(async (url: string, options: RequestInit = {}): Promise<Response> => {
    try {
      return await CSRFManager.request(url, options)
    } catch (err) {
      // Try to refresh token and retry once
      if (err instanceof Error && err.message.includes('CSRF')) {
        await refreshToken()
        return await CSRFManager.request(url, options)
      }
      throw err
    }
  }, [refreshToken])

  /**
   * Get headers for form submission
   */
  const getFormHeaders = useCallback((): Record<string, string> => {
    return CSRFManager.getHeaders()
  }, [token])

  /**
   * Check if token is valid (not null and not expired)
   */
  const isTokenValid = Boolean(token && token.length > 0)

  // Set up token change listener
  useEffect(() => {
    const unsubscribe = CSRFManager.addListener((newToken) => {
      setToken(newToken)
    })

    return unsubscribe
  }, [])

  // Auto-fetch token on mount
  useEffect(() => {
    if (autoFetch && !token && !isLoading) {
      refreshToken()
    }
  }, [autoFetch, token, isLoading, refreshToken])

  // Set up automatic token refresh
  useEffect(() => {
    if (!refreshInterval || refreshInterval <= 0) {
      return
    }

    const interval = setInterval(() => {
      if (!isLoading) {
        refreshToken()
      }
    }, refreshInterval)

    return () => clearInterval(interval)
  }, [refreshInterval, isLoading, refreshToken])

  // Handle page visibility changes to refresh token
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && !isLoading && !token) {
        refreshToken()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
  }, [isLoading, token, refreshToken])

  return {
    token,
    isLoading,
    error,
    refreshToken,
    makeProtectedRequest,
    getFormHeaders,
    isTokenValid
  }
}

/**
 * Hook for form-specific CSRF protection
 */
export function useCSRFForm() {
  const { token, getFormHeaders, makeProtectedRequest, isTokenValid } = useCSRFProtection()

  /**
   * Submit form with CSRF protection
   */
  const submitForm = useCallback(async (
    url: string, 
    formData: FormData | Record<string, any>,
    options: RequestInit = {}
  ): Promise<Response> => {
    const headers = getFormHeaders()

    if (formData instanceof FormData) {
      // Add CSRF token to FormData
      if (token) {
        formData.append('csrf-token', token)
      }
      
      return makeProtectedRequest(url, {
        method: 'POST',
        body: formData,
        ...options
      })
    } else {
      // Handle JSON form data
      return makeProtectedRequest(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...headers,
          ...(options.headers as Record<string, string> || {})
        },
        body: JSON.stringify(formData),
        ...options
      })
    }
  }, [token, getFormHeaders, makeProtectedRequest])

  /**
   * Get hidden input element for forms
   */
  const getCSRFInput = useCallback((): JSX.Element | null => {
    if (!token) return null

    return (
      <input
        type="hidden"
        name="csrf-token"
        value={token}
      />
    )
  }, [token])

  return {
    token,
    isTokenValid,
    submitForm,
    getCSRFInput,
    getFormHeaders
  }
}