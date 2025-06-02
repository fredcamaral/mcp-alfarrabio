/**
 * Protected Form Component
 * 
 * A wrapper component that automatically adds CSRF protection to forms.
 * Provides built-in error handling and loading states.
 */

'use client'

import { FormEvent, ReactNode, useState } from 'react'
import { useCSRFForm } from '@/hooks/useCSRFProtection'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Loader2, Shield, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ProtectedFormProps {
  children: ReactNode
  onSubmit: (formData: FormData, submitProtected: (url: string, data: FormData | Record<string, any>) => Promise<Response>) => Promise<void> | void
  action?: string
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  className?: string
  disabled?: boolean
  showSecurityIndicator?: boolean
  validateBeforeSubmit?: (formData: FormData) => string | null
}

export function ProtectedForm({
  children,
  onSubmit,
  action,
  method = 'POST',
  className,
  disabled = false,
  showSecurityIndicator = true,
  validateBeforeSubmit
}: ProtectedFormProps) {
  const { token, isTokenValid, submitForm, getCSRFInput } = useCSRFForm()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    
    if (disabled || isSubmitting || !isTokenValid) {
      return
    }

    setIsSubmitting(true)
    setError(null)
    setSuccess(null)

    try {
      const form = event.currentTarget
      const formData = new FormData(form)

      // Validate form data if validator provided
      if (validateBeforeSubmit) {
        const validationError = validateBeforeSubmit(formData)
        if (validationError) {
          setError(validationError)
          return
        }
      }

      // Call the provided onSubmit handler
      await onSubmit(formData, submitForm)
      
      setSuccess('Form submitted successfully')
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'An error occurred'
      setError(errorMessage)
      console.error('Form submission error:', err)
    } finally {
      setIsSubmitting(false)
    }
  }

  const canSubmit = isTokenValid && !isSubmitting && !disabled

  return (
    <form 
      onSubmit={handleSubmit}
      action={action}
      method={method}
      className={cn('space-y-4', className)}
    >
      {/* CSRF Token Hidden Input */}
      {getCSRFInput()}

      {/* Security Indicator */}
      {showSecurityIndicator && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Shield className={cn(
            "h-3 w-3",
            isTokenValid ? "text-green-500" : "text-red-500"
          )} />
          <span>
            {isTokenValid ? 'Form protected against CSRF attacks' : 'Security token not available'}
          </span>
        </div>
      )}

      {/* Error Display */}
      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Success Display */}
      {success && (
        <Alert>
          <Shield className="h-4 w-4" />
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}

      {/* Form Content */}
      <div className="space-y-4">
        {children}
      </div>

      {/* Submit Button (if not provided in children) */}
      {!action && (
        <Button
          type="submit"
          disabled={!canSubmit}
          className="w-full"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Submitting...
            </>
          ) : (
            'Submit'
          )}
        </Button>
      )}
    </form>
  )
}

/**
 * Form Field Wrapper with CSRF Protection
 */
interface ProtectedFieldProps {
  children: ReactNode
  name: string
  required?: boolean
  className?: string
}

export function ProtectedField({ children, name, required, className }: ProtectedFieldProps) {
  return (
    <div className={cn('space-y-2', className)}>
      {children}
      {required && (
        <input
          type="hidden"
          name={`${name}_required`}
          value="true"
        />
      )}
    </div>
  )
}

/**
 * CSRF Token Display Component (for debugging)
 */
export function CSRFTokenDisplay() {
  const { token, isTokenValid } = useCSRFForm()

  if (process.env.NODE_ENV !== 'development') {
    return null
  }

  return (
    <div className="mt-4 p-2 bg-muted rounded text-xs font-mono">
      <div className="flex items-center gap-2 mb-1">
        <Shield className={cn(
          "h-3 w-3",
          isTokenValid ? "text-green-500" : "text-red-500"
        )} />
        <span className="font-semibold">CSRF Token Debug</span>
      </div>
      <div>
        <span className="text-muted-foreground">Token: </span>
        <span className={isTokenValid ? "text-green-600" : "text-red-600"}>
          {token ? `${token.substring(0, 8)}...${token.substring(token.length - 8)}` : 'Not available'}
        </span>
      </div>
      <div>
        <span className="text-muted-foreground">Valid: </span>
        <span className={isTokenValid ? "text-green-600" : "text-red-600"}>
          {isTokenValid ? 'Yes' : 'No'}
        </span>
      </div>
    </div>
  )
}