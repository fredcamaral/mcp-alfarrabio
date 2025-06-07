/**
 * Unified Error Handling System
 * 
 * Provides consistent error handling, logging, and user feedback across the WebUI.
 * Centralizes error classification, recovery strategies, and user notifications.
 */

import { toast } from 'react-hot-toast'
import type { ApiError } from '@/types/api'

// Error severity levels
export type ErrorSeverity = 'low' | 'medium' | 'high' | 'critical'

// Error categories for better classification
export type ErrorCategory = 
  | 'network' 
  | 'validation' 
  | 'authentication' 
  | 'authorization' 
  | 'not_found' 
  | 'server_error' 
  | 'client_error'
  | 'csrf'
  | 'timeout'
  | 'unknown'

// Structured error interface
export interface AppError {
  id: string
  message: string
  code?: string
  category: ErrorCategory
  severity: ErrorSeverity
  context?: Record<string, unknown>
  timestamp: Date
  stack?: string
  cause?: Error
  userMessage?: string
  recoverable?: boolean
  retryable?: boolean
}

// Error recovery strategies
export interface ErrorRecovery {
  canRecover: boolean
  retryable: boolean
  retryDelay?: number
  maxRetries?: number
  recoveryAction?: () => Promise<void> | void
  fallbackAction?: () => Promise<void> | void
}

// Error handler configuration
export interface ErrorHandlerConfig {
  enableLogging?: boolean
  enableUserNotifications?: boolean
  enableRetry?: boolean
  logLevel?: 'error' | 'warn' | 'info' | 'debug'
  maxRetries?: number
  retryDelay?: number
}

class ErrorHandler {
  private config: ErrorHandlerConfig
  private errorLog: AppError[] = []
  private maxLogSize = 1000

  constructor(config: ErrorHandlerConfig = {}) {
    this.config = {
      enableLogging: true,
      enableUserNotifications: true,
      enableRetry: true,
      logLevel: 'error',
      maxRetries: 3,
      retryDelay: 1000,
      ...config
    }
  }

  /**
   * Main error handling entry point
   */
  handle(error: unknown, context?: Record<string, unknown>): AppError {
    const appError = this.normalizeError(error, context)
    
    // Log the error
    if (this.config.enableLogging) {
      this.logError(appError)
    }
    
    // Store in memory for debugging
    this.storeError(appError)
    
    // Show user notification
    if (this.config.enableUserNotifications) {
      this.notifyUser(appError)
    }
    
    return appError
  }

  /**
   * Convert various error types to AppError
   */
  private normalizeError(error: unknown, context?: Record<string, unknown>): AppError {
    const id = this.generateErrorId()
    const timestamp = new Date()

    // Handle different error types
    if (error instanceof Error) {
      return {
        id,
        message: error.message,
        category: this.categorizeError(error),
        severity: this.determineSeverity(error),
        context,
        timestamp,
        stack: error.stack,
        cause: error,
        ...this.getErrorDetails(error)
      }
    }

    // Handle API errors
    if (typeof error === 'object' && error !== null && 'error' in error) {
      const apiError = error as { 
        error?: string; 
        code?: string; 
        status?: number; 
        statusCode?: number; 
        details?: Record<string, unknown>;
        response?: { status?: number; data?: { error?: string; message?: string } } 
      }
      return {
        id,
        message: apiError.error || 'API error occurred',
        code: apiError.code,
        category: this.categorizeApiError(apiError as ApiError & Error),
        severity: this.determineSeverityFromStatus(apiError.status),
        context: { ...context, ...apiError.details },
        timestamp,
        userMessage: this.getUserMessage(apiError as ApiError & Error),
        recoverable: this.isRecoverable(apiError as ApiError & Error),
        retryable: this.isRetryable(apiError as ApiError & Error)
      }
    }

    // Handle string errors
    if (typeof error === 'string') {
      return {
        id,
        message: error,
        category: 'unknown',
        severity: 'medium',
        context,
        timestamp,
        userMessage: error,
        recoverable: false,
        retryable: false
      }
    }

    // Handle unknown errors
    return {
      id,
      message: 'An unknown error occurred',
      category: 'unknown',
      severity: 'medium',
      context: { ...context, originalError: error },
      timestamp,
      userMessage: 'Something went wrong. Please try again.',
      recoverable: false,
      retryable: true
    }
  }

  /**
   * Categorize errors based on type and message
   */
  private categorizeError(error: Error): ErrorCategory {
    const message = error.message.toLowerCase()
    
    if (message.includes('network') || message.includes('fetch')) {
      return 'network'
    }
    if (message.includes('csrf')) {
      return 'csrf'
    }
    if (message.includes('timeout')) {
      return 'timeout'
    }
    if (message.includes('validation')) {
      return 'validation'
    }
    if (message.includes('unauthorized') || message.includes('authentication')) {
      return 'authentication'
    }
    if (message.includes('forbidden') || message.includes('authorization')) {
      return 'authorization'
    }
    if (message.includes('not found')) {
      return 'not_found'
    }
    
    return 'client_error'
  }

  /**
   * Categorize API errors based on status and code
   */
  private categorizeApiError(error: Error & { code?: string; status?: number; statusCode?: number; response?: { status?: number } }): ErrorCategory {
    if (error.code === 'CSRF_TOKEN_MISSING' || error.code === 'CSRF_TOKEN_INVALID') {
      return 'csrf'
    }
    
    const status = error.status
    if (status === 401) return 'authentication'
    if (status === 403) return 'authorization'
    if (status === 404) return 'not_found'
    if (status === 422) return 'validation'
    if (status && status >= 500) return 'server_error'
    if (status && status >= 400) return 'client_error'
    
    return 'network'
  }

  /**
   * Determine error severity
   */
  private determineSeverity(error: Error): ErrorSeverity {
    const message = error.message.toLowerCase()
    
    if (message.includes('critical') || message.includes('fatal')) {
      return 'critical'
    }
    if (message.includes('network') || message.includes('timeout')) {
      return 'high'
    }
    if (message.includes('validation') || message.includes('csrf')) {
      return 'medium'
    }
    
    return 'low'
  }

  /**
   * Determine severity from HTTP status
   */
  private determineSeverityFromStatus(status?: number): ErrorSeverity {
    if (!status) return 'medium'
    
    if (status >= 500) return 'critical'
    if (status === 403 || status === 401) return 'high'
    if (status === 422 || status === 400) return 'medium'
    
    return 'low'
  }

  /**
   * Get user-friendly error message
   */
  private getUserMessage(error: Error & { message?: string; code?: string; status?: number; statusCode?: number; error?: string }): string {
    const code = error.code
    const status = error.status
    
    switch (code) {
      case 'CSRF_TOKEN_MISSING':
      case 'CSRF_TOKEN_INVALID':
        return 'Security token expired. Please refresh the page and try again.'
      case 'NETWORK_ERROR':
        return 'Network connection problem. Please check your internet connection.'
      default:
        break
    }
    
    switch (status) {
      case 401:
        return 'Authentication required. Please log in again.'
      case 403:
        return 'You do not have permission to perform this action.'
      case 404:
        return 'The requested resource was not found.'
      case 422:
        return 'Invalid input data. Please check your information and try again.'
      case 500:
        return 'Server error occurred. Please try again later.'
      default:
        return error.error || 'An unexpected error occurred.'
    }
  }

  /**
   * Check if error is recoverable
   */
  private isRecoverable(error: Error & { code?: string; status?: number; statusCode?: number }): boolean {
    const code = error.code
    const status = error.status
    
    // CSRF errors are recoverable by refreshing token
    if (code === 'CSRF_TOKEN_MISSING' || code === 'CSRF_TOKEN_INVALID') {
      return true
    }
    
    // Network errors are often recoverable
    if (code === 'NETWORK_ERROR' || status === 0) {
      return true
    }
    
    // Server errors might be temporary
    if (status && status >= 500) {
      return true
    }
    
    return false
  }

  /**
   * Check if error is retryable
   */
  private isRetryable(error: Error & { code?: string; status?: number; statusCode?: number }): boolean {
    const code = error.code
    const status = error.status
    
    // Don't retry client errors
    if (status && status >= 400 && status < 500) {
      return false
    }
    
    // Retry network and server errors
    if (code === 'NETWORK_ERROR' || (status && status >= 500) || status === 0) {
      return true
    }
    
    return false
  }

  /**
   * Get additional error details
   */
  private getErrorDetails(error: Error): Partial<AppError> {
    // Add specific handling for known error types
    if (error.name === 'TypeError' && error.message.includes('fetch')) {
      return {
        category: 'network',
        userMessage: 'Network connection problem. Please check your internet connection.',
        recoverable: true,
        retryable: true
      }
    }
    
    return {}
  }

  /**
   * Log error to console and external services
   */
  private logError(error: AppError): void {
    const logData = {
      id: error.id,
      message: error.message,
      code: error.code,
      category: error.category,
      severity: error.severity,
      timestamp: error.timestamp,
      context: error.context,
      stack: error.stack
    }
    
    switch (error.severity) {
      case 'critical':
      case 'high':
        console.error('[ERROR]', logData)
        break
      case 'medium':
        console.warn('[WARN]', logData)
        break
      case 'low':
        console.info('[INFO]', logData)
        break
    }
    
    // Send to external logging service if configured
    // this.sendToExternalLogger(logData)
  }

  /**
   * Store error in memory for debugging
   */
  private storeError(error: AppError): void {
    this.errorLog.push(error)
    
    // Keep log size under control
    if (this.errorLog.length > this.maxLogSize) {
      this.errorLog = this.errorLog.slice(-this.maxLogSize / 2)
    }
  }

  /**
   * Show user notification
   */
  private notifyUser(error: AppError): void {
    const message = error.userMessage || error.message
    
    switch (error.severity) {
      case 'critical':
      case 'high':
        toast.error(message, {
          duration: 8000,
          position: 'top-right'
        })
        break
      case 'medium':
        toast.error(message, {
          duration: 5000,
          position: 'top-right'
        })
        break
      case 'low':
        toast(message, {
          duration: 3000,
          position: 'top-right'
        })
        break
    }
  }

  /**
   * Generate unique error ID
   */
  private generateErrorId(): string {
    return `err_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * Get recent errors for debugging
   */
  getRecentErrors(count: number = 10): AppError[] {
    return this.errorLog.slice(-count)
  }

  /**
   * Clear error log
   */
  clearErrors(): void {
    this.errorLog = []
  }

  /**
   * Get error statistics
   */
  getErrorStats(): Record<ErrorCategory, number> {
    const stats: Record<ErrorCategory, number> = {
      network: 0,
      validation: 0,
      authentication: 0,
      authorization: 0,
      not_found: 0,
      server_error: 0,
      client_error: 0,
      csrf: 0,
      timeout: 0,
      unknown: 0
    }
    
    this.errorLog.forEach(error => {
      stats[error.category]++
    })
    
    return stats
  }
}

// Global error handler instance
export const errorHandler = new ErrorHandler()

// Convenience functions
export const handleError = (error: unknown, context?: Record<string, unknown>): AppError => {
  return errorHandler.handle(error, context)
}

export const logError = (message: string, context?: Record<string, unknown>): void => {
  errorHandler.handle(new Error(message), context)
}

export const clearErrors = (): void => {
  errorHandler.clearErrors()
}

export const getRecentErrors = (count?: number): AppError[] => {
  return errorHandler.getRecentErrors(count)
}

export const getErrorStats = (): Record<ErrorCategory, number> => {
  return errorHandler.getErrorStats()
}