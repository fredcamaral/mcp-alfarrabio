/**
 * Hook to sync preferences with UI state
 * 
 * Ensures preferences are applied to the UI on load
 * and when they change
 */

import { useEffect } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import { 
  selectTheme as selectPrefsTheme,
  selectLayout as selectPrefsLayout,
  selectSidebarCollapsed as selectPrefsSidebarCollapsed,
  selectEnableAnimations,
  selectEnableRealtime as selectPrefsEnableRealtime,
  selectKeyboardShortcutsEnabled as selectPrefsKeyboardShortcutsEnabled,
  selectDebugMode as selectPrefsDebugMode
} from '@/store/slices/preferencesSlice'
import {
  setTheme as setUITheme,
  setLayout as setUILayout,
  setSidebarCollapsed as setUISidebarCollapsed,
  setEnableAnimations as setUIEnableAnimations,
  setEnableRealtime as setUIEnableRealtime,
  setKeyboardShortcutsEnabled as setUIKeyboardShortcutsEnabled,
  setDebugMode as setUIDebugMode
} from '@/store/slices/uiSlice'
import { subscribeToPreferences } from '@/lib/preferences'
import { logger } from '@/lib/logger'

export function usePreferencesSync() {
  const dispatch = useAppDispatch()
  
  // Get preferences from store
  const theme = useAppSelector(selectPrefsTheme)
  const layout = useAppSelector(selectPrefsLayout)
  const sidebarCollapsed = useAppSelector(selectPrefsSidebarCollapsed)
  const enableAnimations = useAppSelector(selectEnableAnimations)
  const enableRealtime = useAppSelector(selectPrefsEnableRealtime)
  const keyboardShortcutsEnabled = useAppSelector(selectPrefsKeyboardShortcutsEnabled)
  const debugMode = useAppSelector(selectPrefsDebugMode)

  // Sync preferences to UI state on mount and when they change
  useEffect(() => {
    logger.debug('Syncing preferences to UI state')
    
    // Sync all preferences to UI state
    dispatch(setUITheme(theme))
    dispatch(setUILayout(layout))
    dispatch(setUISidebarCollapsed(sidebarCollapsed))
    dispatch(setUIEnableAnimations(enableAnimations))
    dispatch(setUIEnableRealtime(enableRealtime))
    dispatch(setUIKeyboardShortcutsEnabled(keyboardShortcutsEnabled as boolean))
    dispatch(setUIDebugMode(debugMode as boolean))
  }, [
    dispatch,
    theme,
    layout,
    sidebarCollapsed,
    enableAnimations,
    enableRealtime,
    keyboardShortcutsEnabled,
    debugMode
  ])

  // Subscribe to localStorage changes (for multi-tab sync)
  useEffect(() => {
    const unsubscribe = subscribeToPreferences((preferences) => {
      logger.debug('Preferences changed in another tab, syncing...')
      
      // Update UI state with new preferences
      dispatch(setUITheme(preferences.theme))
      dispatch(setUILayout(preferences.layout))
      dispatch(setUISidebarCollapsed(preferences.sidebarCollapsed))
      dispatch(setUIEnableAnimations(preferences.enableAnimations))
      dispatch(setUIEnableRealtime(preferences.enableRealtime))
      dispatch(setUIKeyboardShortcutsEnabled(preferences.keyboardShortcutsEnabled))
      dispatch(setUIDebugMode(preferences.debugMode))
    })

    return unsubscribe
  }, [dispatch])

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement
    
    if (theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      root.classList.toggle('dark', prefersDark)
    } else {
      root.classList.toggle('dark', theme === 'dark')
    }
  }, [theme])

  // Apply animations preference
  useEffect(() => {
    if (!enableAnimations) {
      document.documentElement.style.setProperty('--animation-duration', '0ms')
      document.documentElement.classList.add('no-animations')
    } else {
      document.documentElement.style.removeProperty('--animation-duration')
      document.documentElement.classList.remove('no-animations')
    }
  }, [enableAnimations])

  // Enable/disable debug mode
  useEffect(() => {
    if (debugMode) {
      window.localStorage.setItem('debug', 'mcp-memory:*')
      logger.debug('Debug mode enabled')
    } else {
      window.localStorage.removeItem('debug')
      logger.info('Debug mode disabled')
    }
  }, [debugMode])
}