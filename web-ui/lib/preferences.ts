/**
 * User Preferences Management
 * 
 * Handles persistence of user preferences to localStorage
 * with type safety and fallback defaults
 */

import { logger } from './logger'

export interface UserPreferences {
  // Theme settings
  theme: 'light' | 'dark' | 'system'
  
  // Layout preferences  
  layout: 'default' | 'compact' | 'comfortable'
  sidebarCollapsed: boolean
  
  // UI preferences
  enableAnimations: boolean
  enableRealtime: boolean
  keyboardShortcutsEnabled: boolean
  
  // Performance preferences
  cacheEnabled: boolean
  debugMode: boolean
  
  // Memory preferences
  defaultView: 'memories' | 'patterns' | 'repositories'
  memoryListLayout: 'grid' | 'list'
  memoriesPerPage: number
  
  // Filter preferences
  defaultFilters: {
    tags: string[]
    repositories: string[]
    minConfidence: number
    outcome: 'all' | 'success' | 'failure'
    difficulty: 'all' | 'easy' | 'medium' | 'hard'
  }
  
  // Search preferences
  recentSearches: string[]
  searchHistoryEnabled: boolean
  maxRecentSearches: number
  
  // Advanced preferences
  autoBackupEnabled: boolean
  backupFrequency: 'daily' | 'weekly' | 'monthly'
  notificationSound: boolean
  
  // WebSocket preferences
  autoReconnect: boolean
  reconnectDelay: number
}

const PREFERENCES_KEY = 'mcp-memory-preferences'
const PREFERENCES_VERSION = '1.0'

// Default preferences
const defaultPreferences: UserPreferences = {
  theme: 'dark',
  layout: 'default',
  sidebarCollapsed: false,
  enableAnimations: true,
  enableRealtime: true,
  keyboardShortcutsEnabled: true,
  cacheEnabled: true,
  debugMode: false,
  defaultView: 'memories',
  memoryListLayout: 'list',
  memoriesPerPage: 20,
  defaultFilters: {
    tags: [],
    repositories: [],
    minConfidence: 0,
    outcome: 'all',
    difficulty: 'all'
  },
  recentSearches: [],
  searchHistoryEnabled: true,
  maxRecentSearches: 10,
  autoBackupEnabled: false,
  backupFrequency: 'weekly',
  notificationSound: true,
  autoReconnect: true,
  reconnectDelay: 5000
}

/**
 * Load user preferences from localStorage
 */
export function loadPreferences(): UserPreferences {
  try {
    const stored = localStorage.getItem(PREFERENCES_KEY)
    if (!stored) {
      return defaultPreferences
    }

    const parsed = JSON.parse(stored)
    
    // Version check - in future, we could migrate old preferences
    if (parsed.version !== PREFERENCES_VERSION) {
      logger.info('Preferences version mismatch, using defaults')
      return defaultPreferences
    }

    // Merge with defaults to ensure all keys exist
    const preferences = {
      ...defaultPreferences,
      ...parsed.preferences
    }

    logger.debug('Loaded user preferences', preferences)
    return preferences
  } catch (error) {
    logger.error('Failed to load preferences', { error })
    return defaultPreferences
  }
}

/**
 * Save user preferences to localStorage
 */
export function savePreferences(preferences: UserPreferences): boolean {
  try {
    const data = {
      version: PREFERENCES_VERSION,
      preferences,
      lastUpdated: new Date().toISOString()
    }

    localStorage.setItem(PREFERENCES_KEY, JSON.stringify(data))
    logger.debug('Saved user preferences')
    return true
  } catch (error) {
    logger.error('Failed to save preferences', { error })
    return false
  }
}

/**
 * Update specific preference values
 */
export function updatePreferences(updates: Partial<UserPreferences>): UserPreferences {
  const current = loadPreferences()
  const updated = {
    ...current,
    ...updates
  }

  savePreferences(updated)
  return updated
}

/**
 * Reset preferences to defaults
 */
export function resetPreferences(): UserPreferences {
  savePreferences(defaultPreferences)
  return defaultPreferences
}

/**
 * Export preferences as JSON
 */
export function exportPreferences(): string {
  const preferences = loadPreferences()
  return JSON.stringify({
    version: PREFERENCES_VERSION,
    preferences,
    exportedAt: new Date().toISOString()
  }, null, 2)
}

/**
 * Import preferences from JSON
 */
export function importPreferences(json: string): UserPreferences | null {
  try {
    const parsed = JSON.parse(json)
    
    if (!parsed.preferences || typeof parsed.preferences !== 'object') {
      throw new Error('Invalid preferences format')
    }

    const preferences = {
      ...defaultPreferences,
      ...parsed.preferences
    }

    savePreferences(preferences)
    logger.info('Imported user preferences')
    return preferences
  } catch (error) {
    logger.error('Failed to import preferences', { error })
    return null
  }
}

/**
 * Clear all stored preferences
 */
export function clearPreferences(): void {
  localStorage.removeItem(PREFERENCES_KEY)
  logger.info('Cleared user preferences')
}

/**
 * Get preference value with type safety
 */
export function getPreference<K extends keyof UserPreferences>(
  key: K
): UserPreferences[K] {
  const preferences = loadPreferences()
  return preferences[key]
}

/**
 * Set preference value with type safety
 */
export function setPreference<K extends keyof UserPreferences>(
  key: K,
  value: UserPreferences[K]
): void {
  updatePreferences({ [key]: value })
}

/**
 * Subscribe to preference changes
 * Returns unsubscribe function
 */
export function subscribeToPreferences(
  callback: (preferences: UserPreferences) => void
): () => void {
  const handleStorageChange = (event: StorageEvent) => {
    if (event.key === PREFERENCES_KEY) {
      const preferences = loadPreferences()
      callback(preferences)
    }
  }

  window.addEventListener('storage', handleStorageChange)

  return () => {
    window.removeEventListener('storage', handleStorageChange)
  }
}