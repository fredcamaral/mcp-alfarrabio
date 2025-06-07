/**
 * Preferences Sync Provider
 * 
 * Provides preferences synchronization across the application
 */

'use client'

import { usePreferencesSync } from '@/hooks/usePreferencesSync'

interface PreferencesSyncProviderProps {
  children: React.ReactNode
}

export function PreferencesSyncProvider({ children }: PreferencesSyncProviderProps) {
  // Initialize preferences sync
  usePreferencesSync()
  
  return <>{children}</>
}