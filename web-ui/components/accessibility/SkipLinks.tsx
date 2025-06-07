/**
 * Skip Links Component
 * 
 * Provides keyboard navigation shortcuts for accessibility
 */

'use client'

import { cn } from '@/lib/utils'

const skipLinks = [
  { href: '#main-content', label: 'Skip to main content' },
  { href: '#main-navigation', label: 'Skip to navigation' },
  { href: '#search', label: 'Skip to search' },
]

export function SkipLinks() {
  return (
    <div className="sr-only focus-within:not-sr-only">
      <div className="absolute top-0 left-0 z-[100] bg-background p-2">
        {skipLinks.map((link) => (
          <a
            key={link.href}
            href={link.href}
            className={cn(
              "inline-block px-4 py-2 mr-2 text-sm font-medium",
              "bg-primary text-primary-foreground rounded-md",
              "focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary",
              "hover:bg-primary/90"
            )}
          >
            {link.label}
          </a>
        ))}
      </div>
    </div>
  )
}