/**
 * React-based Screen Reader Announcer Component
 * 
 * Provides accessible announcements for screen readers
 */

'use client'

import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'

interface AnnouncerProps {
  message?: string
  priority?: 'polite' | 'assertive'
}

export function Announcer({ message = '', priority = 'polite' }: AnnouncerProps) {
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted || !message) {
    return null
  }

  return createPortal(
    <div
      className="sr-only"
      aria-live={priority}
      aria-atomic="true"
      role="status"
    >
      {message}
    </div>,
    document.body
  )
}

// Hook to use the announcer
import { useCallback } from 'react'
import { useAppDispatch } from '@/store/store'
import { addNotification } from '@/store/slices/uiSlice'

export function useAnnouncer() {
  const dispatch = useAppDispatch()

  const announce = useCallback((message: string, priority: 'polite' | 'assertive' = 'polite') => {
    // For now, we'll use the notification system which is accessible
    // In the future, we could add a dedicated announcer state
    dispatch(addNotification({
      type: priority === 'assertive' ? 'error' : 'info',
      title: 'Screen Reader Announcement',
      message,
      duration: 1000,
      ariaOnly: true // Custom property to indicate this is for screen readers only
    }))
  }, [dispatch])

  const announceError = useCallback((message: string) => {
    announce(message, 'assertive')
  }, [announce])

  return {
    announce,
    announceError
  }
}