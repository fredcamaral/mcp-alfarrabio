/**
 * CSRF Protection Provider
 * 
 * Provides CSRF protection context to the entire application.
 * Automatically initializes and manages CSRF tokens for all components.
 */

'use client'

import { createContext, useContext, useEffect, ReactNode } from 'react'
import { useCSRFProtection } from '@/hooks/useCSRFProtection'
import { logger } from '@/lib/logger'

interface CSRFContextValue {
  token: string | null
  isLoading: boolean
  error: string | null
  refreshToken: () => Promise<void>
  makeProtectedRequest: (url: string, options?: RequestInit) => Promise<Response>
  isTokenValid: boolean
}

const CSRFContext = createContext<CSRFContextValue | null>(null)

interface CSRFProviderProps {
  children: ReactNode
  autoRefreshInterval?: number
}

/**
 * CSRF Protection Provider Component
 */
export function CSRFProvider({ children, autoRefreshInterval = 30 * 60 * 1000 }: CSRFProviderProps) {
  const csrfProtection = useCSRFProtection({
    autoFetch: true,
    refreshInterval: autoRefreshInterval // Default: 30 minutes
  })

  // Initialize CSRF protection on app start
  useEffect(() => {
    if (!csrfProtection.token && !csrfProtection.isLoading) {
      csrfProtection.refreshToken()
    }
  }, [csrfProtection.token, csrfProtection.isLoading, csrfProtection.refreshToken])

  // Handle network errors and retry
  useEffect(() => {
    if (csrfProtection.error) {
      logger.warn('CSRF protection error', { error: csrfProtection.error })
      
      // Retry token fetch after 5 seconds on error
      const timeout = setTimeout(() => {
        csrfProtection.refreshToken()
      }, 5000)

      return () => clearTimeout(timeout)
    }
    // Return undefined when there's no error
    return undefined
  }, [csrfProtection.error, csrfProtection.refreshToken])

  return (
    <CSRFContext.Provider value={csrfProtection}>
      {children}
    </CSRFContext.Provider>
  )
}

/**
 * Hook to use CSRF protection context
 */
export function useCSRF(): CSRFContextValue {
  const context = useContext(CSRFContext)
  
  if (!context) {
    throw new Error('useCSRF must be used within a CSRFProvider')
  }
  
  return context
}

/**
 * HOC for components that need CSRF protection
 */
export function withCSRFProtection<P extends object>(
  Component: React.ComponentType<P>
): React.ComponentType<P> {
  return function CSRFProtectedComponent(props: P) {
    return (
      <CSRFProvider>
        <Component {...props} />
      </CSRFProvider>
    )
  }
}

/**
 * CSRF Status Indicator Component
 */
export function CSRFStatusIndicator({ className }: { className?: string }) {
  const { isTokenValid, isLoading, error } = useCSRF()

  if (isLoading) {
    return (
      <div className={`inline-flex items-center gap-1 text-xs text-muted-foreground ${className}`}>
        <div className="h-2 w-2 bg-warning rounded-full animate-pulse" />
        <span>Initializing security...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className={`inline-flex items-center gap-1 text-xs text-destructive ${className}`}>
        <div className="h-2 w-2 bg-destructive rounded-full" />
        <span>Security error</span>
      </div>
    )
  }

  if (isTokenValid) {
    return (
      <div className={`inline-flex items-center gap-1 text-xs text-success ${className}`}>
        <div className="h-2 w-2 bg-success rounded-full" />
        <span>Secure</span>
      </div>
    )
  }

  return (
    <div className={`inline-flex items-center gap-1 text-xs text-destructive ${className}`}>
      <div className="h-2 w-2 bg-destructive rounded-full" />
      <span>Not secure</span>
    </div>
  )
}