import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
import { logger } from "@/lib/logger"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// Format dates consistently across the app
export function formatDate(date: Date | string | number): string {
  const d = new Date(date)
  if (isNaN(d.getTime())) return "Invalid date"

  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffHours = diffMs / (1000 * 60 * 60)
  const diffDays = diffMs / (1000 * 60 * 60 * 24)

  if (diffHours < 1) {
    const minutes = Math.floor(diffMs / (1000 * 60))
    return `${minutes}m ago`
  } else if (diffHours < 24) {
    return `${Math.floor(diffHours)}h ago`
  } else if (diffDays < 7) {
    return `${Math.floor(diffDays)}d ago`
  } else {
    return d.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: d.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
    })
  }
}

// Format memory types for display
export function formatMemoryType(type: string): string {
  return type
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

// Get memory type icon
export function getMemoryTypeIcon(type: string): string {
  const icons: Record<string, string> = {
    problem: "🐛",
    solution: "✅",
    architecture_decision: "🏗️",
    session_summary: "📋",
    code_change: "💻",
    discussion: "💬",
    analysis: "📊",
    verification: "✓",
    question: "❓"
  }
  return icons[type] || "📝"
}

// Truncate text with ellipsis
export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength).trim() + "..."
}

// Generate a consistent color for a string (for avatars, tags, etc.)
export function stringToColor(str: string): string {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash)
  }

  const hue = Math.abs(hash) % 360
  return `hsl(${hue}, 70%, 60%)`
}

// Debounce function for search inputs
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout>
  return (...args: Parameters<T>) => {
    clearTimeout(timeout)
    timeout = setTimeout(() => func(...args), wait)
  }
}

// Calculate confidence score color
export function getConfidenceColor(score: number): string {
  if (score >= 0.8) return "text-success"
  if (score >= 0.6) return "text-warning"
  if (score >= 0.4) return "text-warning"
  return "text-destructive"
}

// Format confidence score as percentage
export function formatConfidence(score: number): string {
  return `${Math.round(score * 100)}%`
}

// Check if a value is defined and not null
export function isDefined<T>(value: T | null | undefined): value is T {
  return value !== null && value !== undefined
}

// Generate a unique ID
export function generateId(): string {
  return Math.random().toString(36).substring(2) + Date.now().toString(36)
}

// Validate environment variables
export function getEnvVar(name: string, defaultValue?: string): string {
  const value = process.env[name]
  if (!value && !defaultValue) {
    throw new Error(`Environment variable ${name} is required`)
  }
  return value || defaultValue!
}

// API URL helpers
export function getAPIUrl(): string {
  return getEnvVar('NEXT_PUBLIC_API_URL', 'http://localhost:9080')
}

export function getGraphQLUrl(): string {
  return getEnvVar('NEXT_PUBLIC_GRAPHQL_URL', '/api/graphql')
}

export function getWSUrl(): string {
  return getEnvVar('NEXT_PUBLIC_WS_URL', 'ws://localhost:9080/ws')
}

export function getWebSocketUrl(): string {
  return getWSUrl()
}

// Development logging utility
export function devLog(...args: unknown[]): void {
  if (process.env.NODE_ENV === 'development' || process.env.DEBUG === 'true') {
    logger.debug(`[DEV] ${args.join(' ')}`)
  }
}

export function devWarn(...args: unknown[]): void {
  if (process.env.NODE_ENV === 'development' || process.env.DEBUG === 'true') {
    logger.warn(`[DEV] ${args.join(' ')}`)
  }
}

export function devError(...args: unknown[]): void {
  if (process.env.NODE_ENV === 'development' || process.env.DEBUG === 'true') {
    const message = args.join(' ')
    const error = args.find(arg => arg instanceof Error) as Error | undefined
    logger.error(`[DEV] ${message}`, error)
  }
}