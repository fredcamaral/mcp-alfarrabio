/**
 * Type Helper Utilities
 * 
 * Common type patterns and utilities to replace 'any' types
 */

// Generic object type for when we truly need flexibility
export type AnyObject = Record<string, unknown>

// For JSON-serializable data
export type JsonValue = string | number | boolean | null | JsonObject | JsonArray
export type JsonObject = { [key: string]: JsonValue }
export type JsonArray = JsonValue[]

// Form data type for API submissions  
export type FormDataObject = Record<string, string | number | boolean | File | undefined>

// Error types
export type ErrorWithMessage = {
  message: string
  code?: string
  details?: unknown
}

// API handler types
export type APIHandler = (
  request: Request,
  context?: { params?: Record<string, string> }
) => Promise<Response> | Response

// React prop types
export type PropsWithClassName<P = unknown> = P & {
  className?: string
}

// Event handler types
export type ChangeHandler<T = HTMLInputElement> = (event: React.ChangeEvent<T>) => void
export type SubmitHandler = (event: React.FormEvent<HTMLFormElement>) => void
export type ClickHandler = (event: React.MouseEvent<HTMLButtonElement>) => void

// Hook return types
export type AsyncState<T> = {
  data: T | null
  loading: boolean
  error: Error | null
}

// Utility types
export type Nullable<T> = T | null
export type Optional<T> = T | undefined
export type ValueOf<T> = T[keyof T]

// Type guards
export function isErrorWithMessage(error: unknown): error is ErrorWithMessage {
  return (
    typeof error === 'object' &&
    error !== null &&
    'message' in error &&
    typeof (error as Record<string, unknown>).message === 'string'
  )
}

export function isJsonObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function isJsonArray(value: unknown): value is JsonArray {
  return Array.isArray(value)
}

// Type assertion helpers
export function assertNever(value: never): never {
  throw new Error(`Unexpected value: ${value}`)
}

export function assertDefined<T>(value: T | null | undefined, message?: string): asserts value is T {
  if (value === null || value === undefined) {
    throw new Error(message || 'Value is null or undefined')
  }
}