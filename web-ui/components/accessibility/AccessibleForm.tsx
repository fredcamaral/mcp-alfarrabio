/**
 * Accessible Form Components
 * 
 * Form components with built-in accessibility features including
 * proper labeling, error handling, and keyboard navigation.
 */

'use client'

import { forwardRef, useId } from 'react'
import { cn } from '@/lib/utils'
import { ErrorMessage } from '@/lib/accessibility'
import { Label } from '@/components/ui/label'
import { Input, type InputProps } from '@/components/ui/input'
import { Textarea, type TextareaProps } from '@/components/ui/textarea'

// Accessible Input Field
interface AccessibleInputProps extends InputProps {
  label: string
  error?: string
  helperText?: string
  required?: boolean
}

export const AccessibleInput = forwardRef<HTMLInputElement, AccessibleInputProps>(
  ({ label, error, helperText, required, className, ...props }, ref) => {
    const inputId = useId()
    const errorId = `${inputId}-error`
    const helperId = `${inputId}-helper`
    
    const ariaDescribedBy = [
      error && errorId,
      helperText && helperId
    ].filter(Boolean).join(' ')

    return (
      <div className="space-y-2">
        <Label htmlFor={inputId} className={cn(required && "after:content-['*'] after:ml-0.5 after:text-destructive")}>
          {label}
        </Label>
        
        <Input
          ref={ref}
          id={inputId}
          aria-invalid={!!error}
          aria-describedby={ariaDescribedBy || undefined}
          aria-required={required}
          className={cn(
            error && "border-destructive focus-visible:ring-destructive",
            className
          )}
          {...props}
        />
        
        {helperText && !error && (
          <p id={helperId} className="text-sm text-muted-foreground">
            {helperText}
          </p>
        )}
        
        {error && <ErrorMessage error={error} id={errorId} />}
      </div>
    )
  }
)

AccessibleInput.displayName = 'AccessibleInput'

// Accessible Textarea
interface AccessibleTextareaProps extends TextareaProps {
  label: string
  error?: string
  helperText?: string
  required?: boolean
}

export const AccessibleTextarea = forwardRef<HTMLTextAreaElement, AccessibleTextareaProps>(
  ({ label, error, helperText, required, className, ...props }, ref) => {
    const textareaId = useId()
    const errorId = `${textareaId}-error`
    const helperId = `${textareaId}-helper`
    
    const ariaDescribedBy = [
      error && errorId,
      helperText && helperId
    ].filter(Boolean).join(' ')

    return (
      <div className="space-y-2">
        <Label htmlFor={textareaId} className={cn(required && "after:content-['*'] after:ml-0.5 after:text-destructive")}>
          {label}
        </Label>
        
        <Textarea
          ref={ref}
          id={textareaId}
          aria-invalid={!!error}
          aria-describedby={ariaDescribedBy || undefined}
          aria-required={required}
          className={cn(
            error && "border-destructive focus-visible:ring-destructive",
            className
          )}
          {...props}
        />
        
        {helperText && !error && (
          <p id={helperId} className="text-sm text-muted-foreground">
            {helperText}
          </p>
        )}
        
        {error && <ErrorMessage error={error} id={errorId} />}
      </div>
    )
  }
)

AccessibleTextarea.displayName = 'AccessibleTextarea'

// Accessible Select
interface AccessibleSelectProps {
  label: string
  options: Array<{ value: string; label: string; disabled?: boolean }>
  value: string
  onChange: (value: string) => void
  error?: string
  helperText?: string
  required?: boolean
  placeholder?: string
  className?: string
}

export function AccessibleSelect({
  label,
  options,
  value,
  onChange,
  error,
  helperText,
  required,
  placeholder = 'Select an option',
  className
}: AccessibleSelectProps) {
  const selectId = useId()
  const errorId = `${selectId}-error`
  const helperId = `${selectId}-helper`
  
  const ariaDescribedBy = [
    error && errorId,
    helperText && helperId
  ].filter(Boolean).join(' ')

  return (
    <div className="space-y-2">
      <Label htmlFor={selectId} className={cn(required && "after:content-['*'] after:ml-0.5 after:text-destructive")}>
        {label}
      </Label>
      
      <select
        id={selectId}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={!!error}
        aria-describedby={ariaDescribedBy || undefined}
        aria-required={required}
        className={cn(
          "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
          "disabled:cursor-not-allowed disabled:opacity-50",
          error && "border-destructive focus-visible:ring-destructive",
          className
        )}
      >
        <option value="" disabled>
          {placeholder}
        </option>
        {options.map((option) => (
          <option
            key={option.value}
            value={option.value}
            disabled={option.disabled}
          >
            {option.label}
          </option>
        ))}
      </select>
      
      {helperText && !error && (
        <p id={helperId} className="text-sm text-muted-foreground">
          {helperText}
        </p>
      )}
      
      {error && <ErrorMessage error={error} id={errorId} />}
    </div>
  )
}

// Accessible Checkbox
interface AccessibleCheckboxProps {
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
  error?: string
  helperText?: string
  required?: boolean
  className?: string
}

export function AccessibleCheckbox({
  label,
  checked,
  onChange,
  error,
  helperText,
  required,
  className
}: AccessibleCheckboxProps) {
  const checkboxId = useId()
  const errorId = `${checkboxId}-error`
  const helperId = `${checkboxId}-helper`
  
  const ariaDescribedBy = [
    error && errorId,
    helperText && helperId
  ].filter(Boolean).join(' ')

  return (
    <div className="space-y-2">
      <div className="flex items-center space-x-2">
        <input
          type="checkbox"
          id={checkboxId}
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          aria-invalid={!!error}
          aria-describedby={ariaDescribedBy || undefined}
          aria-required={required}
          className={cn(
            "h-4 w-4 rounded border-gray-300 text-primary",
            "focus:ring-2 focus:ring-primary focus:ring-offset-2",
            error && "border-destructive",
            className
          )}
        />
        <Label
          htmlFor={checkboxId}
          className={cn(
            "text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70",
            required && "after:content-['*'] after:ml-0.5 after:text-destructive"
          )}
        >
          {label}
        </Label>
      </div>
      
      {helperText && !error && (
        <p id={helperId} className="text-sm text-muted-foreground ml-6">
          {helperText}
        </p>
      )}
      
      {error && (
        <div className="ml-6">
          <ErrorMessage error={error} id={errorId} />
        </div>
      )}
    </div>
  )
}

// Accessible Radio Group
interface AccessibleRadioGroupProps {
  label: string
  options: Array<{ value: string; label: string; disabled?: boolean }>
  value: string
  onChange: (value: string) => void
  error?: string
  helperText?: string
  required?: boolean
  orientation?: 'horizontal' | 'vertical'
  className?: string
}

export function AccessibleRadioGroup({
  label,
  options,
  value,
  onChange,
  error,
  helperText,
  required,
  orientation = 'vertical',
  className
}: AccessibleRadioGroupProps) {
  const groupId = useId()
  const errorId = `${groupId}-error`
  const helperId = `${groupId}-helper`
  
  const ariaDescribedBy = [
    error && errorId,
    helperText && helperId
  ].filter(Boolean).join(' ')

  return (
    <fieldset
      className={cn("space-y-2", className)}
      aria-invalid={!!error}
      aria-describedby={ariaDescribedBy || undefined}
      aria-required={required}
    >
      <legend className={cn(
        "text-sm font-medium leading-none",
        required && "after:content-['*'] after:ml-0.5 after:text-destructive"
      )}>
        {label}
      </legend>
      
      <div
        role="radiogroup"
        aria-labelledby={groupId}
        className={cn(
          "space-y-2",
          orientation === 'horizontal' && "flex space-x-4 space-y-0"
        )}
      >
        {options.map((option) => {
          const optionId = `${groupId}-${option.value}`
          return (
            <div key={option.value} className="flex items-center space-x-2">
              <input
                type="radio"
                id={optionId}
                name={groupId}
                value={option.value}
                checked={value === option.value}
                onChange={(e) => onChange(e.target.value)}
                disabled={option.disabled}
                className={cn(
                  "h-4 w-4 border-gray-300 text-primary",
                  "focus:ring-2 focus:ring-primary focus:ring-offset-2",
                  error && "border-destructive"
                )}
              />
              <Label
                htmlFor={optionId}
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                {option.label}
              </Label>
            </div>
          )
        })}
      </div>
      
      {helperText && !error && (
        <p id={helperId} className="text-sm text-muted-foreground">
          {helperText}
        </p>
      )}
      
      {error && <ErrorMessage error={error} id={errorId} />}
    </fieldset>
  )
}

// Form Field Group for consistent spacing
interface FormFieldGroupProps {
  children: React.ReactNode
  className?: string
}

export function FormFieldGroup({ children, className }: FormFieldGroupProps) {
  return (
    <div className={cn("space-y-6", className)}>
      {children}
    </div>
  )
}

// Accessible Form with proper structure
interface AccessibleFormProps extends React.FormHTMLAttributes<HTMLFormElement> {
  children: React.ReactNode
  title?: string
  description?: string
}

export function AccessibleForm({
  children,
  title,
  description,
  className,
  ...props
}: AccessibleFormProps) {
  const formId = useId()
  const titleId = `${formId}-title`
  const descId = `${formId}-desc`

  return (
    <form
      className={cn("space-y-6", className)}
      aria-labelledby={title ? titleId : undefined}
      aria-describedby={description ? descId : undefined}
      {...props}
    >
      {title && (
        <h2 id={titleId} className="text-2xl font-semibold">
          {title}
        </h2>
      )}
      
      {description && (
        <p id={descId} className="text-muted-foreground">
          {description}
        </p>
      )}
      
      {children}
    </form>
  )
}