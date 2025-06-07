'use client'

/**
 * Accessibility Utilities
 * 
 * Comprehensive accessibility support for keyboard navigation,
 * screen readers, and WCAG compliance.
 */

import React, { useEffect, useRef, useCallback, useState } from 'react'

// ARIA Live Region Announcer
class LiveRegionAnnouncer {
  private static instance: LiveRegionAnnouncer
  private container: HTMLDivElement | null = null

  private constructor() {
    if (typeof window !== 'undefined') {
      this.createContainer()
    }
  }

  static getInstance(): LiveRegionAnnouncer {
    if (!LiveRegionAnnouncer.instance) {
      LiveRegionAnnouncer.instance = new LiveRegionAnnouncer()
    }
    return LiveRegionAnnouncer.instance
  }

  private createContainer(): void {
    this.container = document.createElement('div')
    this.container.className = 'sr-only'
    this.container.setAttribute('aria-live', 'polite')
    this.container.setAttribute('aria-atomic', 'true')
    this.container.setAttribute('role', 'status')
    document.body.appendChild(this.container)
  }

  announce(message: string, priority: 'polite' | 'assertive' = 'polite'): void {
    if (!this.container) return

    this.container.setAttribute('aria-live', priority)
    this.container.textContent = message

    // Clear after announcement
    setTimeout(() => {
      if (this.container) {
        this.container.textContent = ''
      }
    }, 1000)
  }

  announceError(message: string): void {
    this.announce(message, 'assertive')
  }
}

// Export singleton instance
export const announcer = LiveRegionAnnouncer.getInstance()

/**
 * Announce message to screen readers
 */
export function announce(message: string, priority?: 'polite' | 'assertive'): void {
  announcer.announce(message, priority)
}

/**
 * Announce error to screen readers
 */
export function announceError(message: string): void {
  announcer.announceError(message)
}

/**
 * Focus trap hook for modals and dialogs
 */
export function useFocusTrap(isActive = true) {
  const containerRef = useRef<HTMLDivElement>(null)
  const previousFocusRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!isActive || !containerRef.current) return

    const container = containerRef.current
    const focusableElements = container.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )

    const firstFocusable = focusableElements[0]
    const lastFocusable = focusableElements[focusableElements.length - 1]

    // Store current focus
    previousFocusRef.current = document.activeElement as HTMLElement

    // Focus first element
    firstFocusable?.focus()

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return

      if (e.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstFocusable) {
          e.preventDefault()
          lastFocusable?.focus()
        }
      } else {
        // Tab
        if (document.activeElement === lastFocusable) {
          e.preventDefault()
          firstFocusable?.focus()
        }
      }
    }

    container.addEventListener('keydown', handleKeyDown)

    return () => {
      container.removeEventListener('keydown', handleKeyDown)
      // Restore focus
      previousFocusRef.current?.focus()
    }
  }, [isActive])

  return containerRef
}

/**
 * Keyboard navigation hook
 */
export function useKeyboardNavigation(items: HTMLElement[], options?: {
  orientation?: 'horizontal' | 'vertical' | 'both'
  loop?: boolean
  onSelect?: (index: number) => void
}) {
  const {
    orientation = 'vertical',
    loop = true,
    onSelect
  } = options || {}

  const currentIndexRef = useRef(0)

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    let newIndex = currentIndexRef.current
    const itemCount = items.length

    switch (e.key) {
      case 'ArrowDown':
        if (orientation === 'vertical' || orientation === 'both') {
          e.preventDefault()
          newIndex = loop 
            ? (currentIndexRef.current + 1) % itemCount
            : Math.min(currentIndexRef.current + 1, itemCount - 1)
        }
        break

      case 'ArrowUp':
        if (orientation === 'vertical' || orientation === 'both') {
          e.preventDefault()
          newIndex = loop
            ? (currentIndexRef.current - 1 + itemCount) % itemCount
            : Math.max(currentIndexRef.current - 1, 0)
        }
        break

      case 'ArrowRight':
        if (orientation === 'horizontal' || orientation === 'both') {
          e.preventDefault()
          newIndex = loop
            ? (currentIndexRef.current + 1) % itemCount
            : Math.min(currentIndexRef.current + 1, itemCount - 1)
        }
        break

      case 'ArrowLeft':
        if (orientation === 'horizontal' || orientation === 'both') {
          e.preventDefault()
          newIndex = loop
            ? (currentIndexRef.current - 1 + itemCount) % itemCount
            : Math.max(currentIndexRef.current - 1, 0)
        }
        break

      case 'Home':
        e.preventDefault()
        newIndex = 0
        break

      case 'End':
        e.preventDefault()
        newIndex = itemCount - 1
        break

      case 'Enter':
      case ' ':
        e.preventDefault()
        onSelect?.(currentIndexRef.current)
        return

      default:
        return
    }

    if (newIndex !== currentIndexRef.current) {
      currentIndexRef.current = newIndex
      items[newIndex]?.focus()
    }
  }, [items, orientation, loop, onSelect])

  useEffect(() => {
    items.forEach(item => {
      item.addEventListener('keydown', handleKeyDown)
    })

    return () => {
      items.forEach(item => {
        item.removeEventListener('keydown', handleKeyDown)
      })
    }
  }, [items, handleKeyDown])

  return {
    setFocus: (index: number) => {
      currentIndexRef.current = index
      items[index]?.focus()
    }
  }
}

/**
 * Skip to main content link
 */
export function SkipToMain() {
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-4 focus:py-2 focus:bg-primary focus:text-primary-foreground focus:rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
    >
      Skip to main content
    </a>
  )
}

/**
 * Visually hidden component for screen reader only content
 */
export function VisuallyHidden({ children }: { children: React.ReactNode }) {
  return <span className="sr-only">{children}</span>
}

/**
 * ARIA describedby hook
 */
export function useAriaDescribedBy(description: string) {
  const id = useRef(`aria-${Math.random().toString(36).substr(2, 9)}`)

  return {
    describedById: id.current,
    describedByProps: {
      'aria-describedby': id.current
    },
    DescribedBy: () => (
      <span id={id.current} className="sr-only">
        {description}
      </span>
    )
  }
}

/**
 * ARIA label hook
 */
export function useAriaLabel(label: string) {
  return {
    'aria-label': label
  }
}

/**
 * Roving tabindex hook for complex widgets
 */
export function useRovingTabIndex(_itemsCount: number, initialIndex = 0) {
  const [activeIndex, setActiveIndex] = useState(initialIndex)

  const getRovingProps = useCallback((index: number) => ({
    tabIndex: index === activeIndex ? 0 : -1,
    onFocus: () => setActiveIndex(index)
  }), [activeIndex])

  return {
    activeIndex,
    setActiveIndex,
    getRovingProps
  }
}

/**
 * Accessible loading state
 */
export function LoadingState({ message = 'Loading...' }: { message?: string }) {
  useEffect(() => {
    announce(message)
    return () => {
      announce('Loading complete')
    }
  }, [message])

  return (
    <div role="status" aria-live="polite">
      <VisuallyHidden>{message}</VisuallyHidden>
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    </div>
  )
}

/**
 * Accessible error message
 */
export function ErrorMessage({ 
  error, 
  id 
}: { 
  error: string | null
  id: string 
}) {
  useEffect(() => {
    if (error) {
      announceError(error)
    }
  }, [error])

  if (!error) return null

  return (
    <div 
      id={id}
      role="alert"
      className="text-sm text-destructive mt-1 flex items-center gap-1"
    >
      <span aria-hidden="true">⚠</span>
      {error}
    </div>
  )
}

/**
 * Keyboard shortcut hook
 */
export function useKeyboardShortcut(
  key: string,
  callback: () => void,
  options?: {
    ctrl?: boolean
    shift?: boolean
    alt?: boolean
    meta?: boolean
    preventDefault?: boolean
  }
) {
  const {
    ctrl = false,
    shift = false,
    alt = false,
    meta = false,
    preventDefault = true
  } = options || {}

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (
        e.key === key &&
        e.ctrlKey === ctrl &&
        e.shiftKey === shift &&
        e.altKey === alt &&
        e.metaKey === meta
      ) {
        if (preventDefault) {
          e.preventDefault()
        }
        callback()
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [key, callback, ctrl, shift, alt, meta, preventDefault])
}

/**
 * Reduced motion preference hook
 */
export function useReducedMotion() {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false)

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    setPrefersReducedMotion(mediaQuery.matches)

    const handler = (e: MediaQueryListEvent) => {
      setPrefersReducedMotion(e.matches)
    }

    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }, [])

  return prefersReducedMotion
}

/**
 * ARIA attributes for common patterns
 */
export const ariaPatterns = {
  button: (pressed?: boolean) => ({
    role: 'button',
    tabIndex: 0,
    'aria-pressed': pressed
  }),
  
  link: {
    role: 'link',
    tabIndex: 0
  },
  
  menu: {
    role: 'menu',
    'aria-orientation': 'vertical' as const
  },
  
  menuItem: {
    role: 'menuitem',
    tabIndex: -1
  },
  
  tab: (selected: boolean) => ({
    role: 'tab',
    'aria-selected': selected,
    tabIndex: selected ? 0 : -1
  }),
  
  tabPanel: (hidden: boolean) => ({
    role: 'tabpanel',
    'aria-hidden': hidden,
    tabIndex: 0
  }),
  
  dialog: (labelId: string) => ({
    role: 'dialog',
    'aria-modal': true,
    'aria-labelledby': labelId
  }),
  
  alert: {
    role: 'alert',
    'aria-live': 'assertive' as const,
    'aria-atomic': true
  },
  
  status: {
    role: 'status',
    'aria-live': 'polite' as const,
    'aria-atomic': true
  }
}

// Re-export React types for convenience