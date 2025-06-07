/**
 * React Error Boundary Components
 * 
 * Provides error boundaries for graceful error handling in React components.
 * Integrates with the unified error handling system for consistent error management.
 */

'use client'

import React, { Component, ReactNode } from 'react'
import { handleError, type AppError } from '@/lib/error-handling'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { AlertTriangle, RefreshCw, Home, Bug } from 'lucide-react'

interface ErrorBoundaryState {
  hasError: boolean
  error?: AppError
  errorInfo?: React.ErrorInfo
}

interface ErrorBoundaryProps {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: AppError, errorInfo: React.ErrorInfo) => void
  enableRetry?: boolean
  enableLogging?: boolean
  showErrorDetails?: boolean
}

/**
 * Main Error Boundary Component
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    // Update state so the next render will show the fallback UI
    const appError = handleError(error, { 
      source: 'react_error_boundary',
      boundary: true 
    })
    
    return {
      hasError: true,
      error: appError
    }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    const appError = handleError(error, {
      source: 'react_error_boundary',
      boundary: true,
      componentStack: errorInfo.componentStack
    })

    this.setState({
      error: appError,
      errorInfo
    })

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(appError, errorInfo)
    }

    // Log error details
    if (this.props.enableLogging !== false) {
      console.error('React Error Boundary caught an error:', {
        error,
        errorInfo,
        appError
      })
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: undefined, errorInfo: undefined })
  }

  handleGoHome = () => {
    window.location.href = '/'
  }

  render() {
    if (this.state.hasError) {
      // Custom fallback UI
      if (this.props.fallback) {
        return this.props.fallback
      }

      // Default error UI
      return (
        <ErrorFallbackUI
          error={this.state.error}
          errorInfo={this.state.errorInfo}
          onRetry={this.props.enableRetry !== false ? this.handleRetry : undefined}
          onGoHome={this.handleGoHome}
          showErrorDetails={this.props.showErrorDetails}
        />
      )
    }

    return this.props.children
  }
}

/**
 * Error Fallback UI Component
 */
interface ErrorFallbackUIProps {
  error?: AppError
  errorInfo?: React.ErrorInfo
  onRetry?: () => void
  onGoHome?: () => void
  showErrorDetails?: boolean
}

function ErrorFallbackUI({
  error,
  errorInfo,
  onRetry,
  onGoHome,
  showErrorDetails = false
}: ErrorFallbackUIProps) {
  const [showDetails, setShowDetails] = React.useState(false)

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-muted/30">
      <Card className="w-full max-w-2xl">
        <CardHeader>
          <div className="flex items-center space-x-3">
            <AlertTriangle className="h-8 w-8 text-destructive" />
            <div>
              <CardTitle className="text-xl">Something went wrong</CardTitle>
              <p className="text-sm text-muted-foreground mt-1">
                {error?.userMessage || error?.message || 'An unexpected error occurred'}
              </p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Error Summary */}
          {error && (
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                <div className="space-y-1">
                  <div><strong>Error ID:</strong> {error.id}</div>
                  <div><strong>Category:</strong> {error.category}</div>
                  <div><strong>Severity:</strong> {error.severity}</div>
                  {error.code && <div><strong>Code:</strong> {error.code}</div>}
                </div>
              </AlertDescription>
            </Alert>
          )}

          {/* Action Buttons */}
          <div className="flex gap-3">
            {onRetry && (
              <Button onClick={onRetry} className="flex-1">
                <RefreshCw className="h-4 w-4 mr-2" />
                Try Again
              </Button>
            )}
            <Button variant="outline" onClick={onGoHome} className="flex-1">
              <Home className="h-4 w-4 mr-2" />
              Go Home
            </Button>
          </div>

          {/* Error Details Toggle */}
          {(showErrorDetails || process.env.NODE_ENV === 'development') && (
            <div className="pt-4 border-t">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowDetails(!showDetails)}
                className="mb-3"
              >
                <Bug className="h-4 w-4 mr-2" />
                {showDetails ? 'Hide' : 'Show'} Error Details
              </Button>

              {showDetails && (
                <div className="space-y-3">
                  {/* Error Message */}
                  {error && (
                    <div className="bg-destructive/10 p-3 rounded-md">
                      <h4 className="font-medium text-destructive mb-2">Error Message</h4>
                      <pre className="text-sm text-destructive/90 whitespace-pre-wrap">
                        {error.message}
                      </pre>
                    </div>
                  )}

                  {/* Stack Trace */}
                  {error?.stack && (
                    <div className="bg-muted p-3 rounded-md">
                      <h4 className="font-medium text-foreground mb-2">Stack Trace</h4>
                      <pre className="text-xs text-muted-foreground whitespace-pre-wrap overflow-x-auto">
                        {error.stack}
                      </pre>
                    </div>
                  )}

                  {/* Component Stack */}
                  {errorInfo?.componentStack && (
                    <div className="bg-info-muted p-3 rounded-md">
                      <h4 className="font-medium text-info mb-2">Component Stack</h4>
                      <pre className="text-xs text-info/90 whitespace-pre-wrap overflow-x-auto">
                        {errorInfo.componentStack}
                      </pre>
                    </div>
                  )}

                  {/* Context */}
                  {error?.context && Object.keys(error.context).length > 0 && (
                    <div className="bg-warning-muted p-3 rounded-md">
                      <h4 className="font-medium text-warning mb-2">Context</h4>
                      <pre className="text-xs text-warning/90 whitespace-pre-wrap">
                        {JSON.stringify(error.context, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Report Issue */}
          <div className="pt-4 border-t text-center">
            <p className="text-sm text-muted-foreground">
              If this problem persists, please report it to the development team.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Async Error Boundary for handling async errors
 */
interface AsyncErrorBoundaryState {
  asyncError?: AppError
}

export class AsyncErrorBoundary extends Component<ErrorBoundaryProps, AsyncErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = {}
  }

  componentDidMount() {
    // Listen for unhandled promise rejections
    window.addEventListener('unhandledrejection', this.handlePromiseRejection)
  }

  componentWillUnmount() {
    window.removeEventListener('unhandledrejection', this.handlePromiseRejection)
  }

  handlePromiseRejection = (event: PromiseRejectionEvent) => {
    const appError = handleError(event.reason, {
      source: 'async_error_boundary',
      type: 'unhandled_promise_rejection'
    })

    this.setState({ asyncError: appError })

    if (this.props.onError) {
      this.props.onError(appError, { componentStack: '' })
    }

    // Prevent the default browser behavior
    event.preventDefault()
  }

  handleAsyncRetry = () => {
    this.setState({ asyncError: undefined })
  }

  render() {
    if (this.state.asyncError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <ErrorFallbackUI
          error={this.state.asyncError}
          onRetry={this.props.enableRetry !== false ? this.handleAsyncRetry : undefined}
          onGoHome={() => window.location.href = '/'}
          showErrorDetails={this.props.showErrorDetails}
        />
      )
    }

    return this.props.children
  }
}

/**
 * HOC for wrapping components with error boundary
 */
export function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  errorBoundaryProps?: Omit<ErrorBoundaryProps, 'children'>
) {
  return function WrappedComponent(props: P) {
    return (
      <ErrorBoundary {...errorBoundaryProps}>
        <Component {...props} />
      </ErrorBoundary>
    )
  }
}

/**
 * Hook for throwing async errors to error boundary
 */
export function useErrorHandler() {
  return React.useCallback((error: unknown, context?: Record<string, unknown>) => {
    const appError = handleError(error, context)
    
    // For async errors that aren't caught by error boundaries,
    // we can trigger a re-render that will be caught
    throw appError
  }, [])
}