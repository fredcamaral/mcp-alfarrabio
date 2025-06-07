/**
 * Accessible Navigation Components
 * 
 * Navigation components with proper ARIA labels, keyboard navigation,
 * and screen reader support.
 */

'use client'

import { useRef, useState } from 'react'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import { 
  useKeyboardNavigation, 
  useRovingTabIndex,
  VisuallyHidden,
  ariaPatterns 
} from '@/lib/accessibility'
import Link from 'next/link'

// Navigation Item Type
interface NavItem {
  id: string
  label: string
  href?: string
  icon?: React.ReactNode
  badge?: string | number
  items?: NavItem[]
  disabled?: boolean
}

// Accessible Main Navigation
interface AccessibleNavigationProps {
  items: NavItem[]
  orientation?: 'horizontal' | 'vertical'
  className?: string
  'aria-label'?: string
}

export function AccessibleNavigation({
  items,
  orientation = 'horizontal',
  className,
  'aria-label': ariaLabel = 'Main navigation'
}: AccessibleNavigationProps) {
  const pathname = usePathname()
  const navRef = useRef<HTMLElement>(null)
  const itemRefs = useRef<HTMLElement[]>([])

  // Set up keyboard navigation
  useKeyboardNavigation(itemRefs.current, {
    orientation,
    loop: true,
    onSelect: (index) => {
      const item = items[index]
      if (item.href && !item.disabled) {
        window.location.href = item.href
      }
    }
  })

  const isActive = (href: string) => pathname === href

  return (
    <nav
      ref={navRef}
      aria-label={ariaLabel}
      className={cn(
        "flex",
        orientation === 'vertical' ? "flex-col space-y-1" : "space-x-1",
        className
      )}
    >
      <ul
        role="list"
        className={cn(
          "flex",
          orientation === 'vertical' ? "flex-col space-y-1" : "space-x-1"
        )}
      >
        {items.map((item, index) => (
          <li key={item.id} role="none">
            <Link
              ref={(el) => {
                if (el) itemRefs.current[index] = el
              }}
              href={item.href || '#'}
              className={cn(
                "flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                "hover:bg-accent hover:text-accent-foreground",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                isActive(item.href || '') && "bg-accent text-accent-foreground",
                item.disabled && "opacity-50 cursor-not-allowed pointer-events-none"
              )}
              aria-current={isActive(item.href || '') ? 'page' : undefined}
              aria-disabled={item.disabled}
              tabIndex={index === 0 ? 0 : -1}
            >
              {item.icon && (
                <span aria-hidden="true" className="flex-shrink-0">
                  {item.icon}
                </span>
              )}
              <span>{item.label}</span>
              {item.badge && (
                <span className="ml-auto flex h-5 min-w-[20px] items-center justify-center rounded-full bg-primary px-1 text-xs text-primary-foreground">
                  {item.badge}
                  <VisuallyHidden> notifications</VisuallyHidden>
                </span>
              )}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  )
}

// Accessible Breadcrumb Navigation
interface BreadcrumbItem {
  label: string
  href?: string
}

interface AccessibleBreadcrumbProps {
  items: BreadcrumbItem[]
  className?: string
}

export function AccessibleBreadcrumb({
  items,
  className
}: AccessibleBreadcrumbProps) {
  return (
    <nav aria-label="Breadcrumb" className={cn("flex", className)}>
      <ol className="flex items-center space-x-2 text-sm">
        {items.map((item, index) => {
          const isLast = index === items.length - 1
          
          return (
            <li key={index} className="flex items-center">
              {index > 0 && (
                <span aria-hidden="true" className="mx-2 text-muted-foreground">
                  /
                </span>
              )}
              
              {isLast || !item.href ? (
                <span
                  className={cn(
                    isLast ? "font-medium text-foreground" : "text-muted-foreground"
                  )}
                  aria-current={isLast ? "page" : undefined}
                >
                  {item.label}
                </span>
              ) : (
                <Link
                  href={item.href}
                  className="text-muted-foreground hover:text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
                >
                  {item.label}
                </Link>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}

// Accessible Tab Navigation
interface TabItem {
  id: string
  label: string
  content: React.ReactNode
  disabled?: boolean
}

interface AccessibleTabsProps {
  tabs: TabItem[]
  defaultTab?: string
  className?: string
}

export function AccessibleTabs({
  tabs,
  defaultTab = tabs[0]?.id,
  className
}: AccessibleTabsProps) {
  const [activeTab, setActiveTab] = useState(defaultTab)
  const { getRovingProps } = useRovingTabIndex(tabs.length)
  
  const tabListRef = useRef<HTMLDivElement>(null)
  const tabRefs = useRef<HTMLButtonElement[]>([])

  useKeyboardNavigation(tabRefs.current, {
    orientation: 'horizontal',
    loop: false,
    onSelect: (index) => {
      const tab = tabs[index]
      if (!tab.disabled) {
        setActiveTab(tab.id)
      }
    }
  })

  return (
    <div className={cn("w-full", className)}>
      <div
        ref={tabListRef}
        role="tablist"
        aria-orientation="horizontal"
        className="flex border-b"
      >
        {tabs.map((tab, index) => {
          const isActive = activeTab === tab.id
          
          return (
            <button
              key={tab.id}
              ref={(el) => {
                if (el) tabRefs.current[index] = el
              }}
              {...ariaPatterns.tab(isActive)}
              {...getRovingProps(index)}
              id={`tab-${tab.id}`}
              aria-controls={`panel-${tab.id}`}
              disabled={tab.disabled}
              onClick={() => !tab.disabled && setActiveTab(tab.id)}
              className={cn(
                "px-4 py-2 text-sm font-medium transition-colors",
                "hover:text-foreground",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                "disabled:opacity-50 disabled:cursor-not-allowed",
                isActive
                  ? "text-foreground border-b-2 border-primary"
                  : "text-muted-foreground"
              )}
            >
              {tab.label}
            </button>
          )
        })}
      </div>
      
      {tabs.map((tab) => {
        const isActive = activeTab === tab.id
        
        return (
          <div
            key={tab.id}
            {...ariaPatterns.tabPanel(!isActive)}
            id={`panel-${tab.id}`}
            aria-labelledby={`tab-${tab.id}`}
            className={cn(
              "mt-4",
              !isActive && "hidden"
            )}
          >
            {tab.content}
          </div>
        )
      })}
    </div>
  )
}

// Accessible Pagination
interface AccessiblePaginationProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
  siblingCount?: number
  className?: string
}

export function AccessiblePagination({
  currentPage,
  totalPages,
  onPageChange,
  siblingCount = 1,
  className
}: AccessiblePaginationProps) {
  
  // Generate page numbers to display
  const getPageNumbers = () => {
    const pages: (number | string)[] = []
    const leftSibling = Math.max(currentPage - siblingCount, 1)
    const rightSibling = Math.min(currentPage + siblingCount, totalPages)
    
    // Always show first page
    pages.push(1)
    
    // Add ellipsis if needed
    if (leftSibling > 2) {
      pages.push('...')
    }
    
    // Add sibling pages
    for (let i = leftSibling; i <= rightSibling; i++) {
      if (i !== 1 && i !== totalPages) {
        pages.push(i)
      }
    }
    
    // Add ellipsis if needed
    if (rightSibling < totalPages - 1) {
      pages.push('...')
    }
    
    // Always show last page
    if (totalPages > 1) {
      pages.push(totalPages)
    }
    
    return pages
  }

  const pages = getPageNumbers()

  return (
    <nav
      aria-label="Pagination Navigation"
      className={cn("flex items-center justify-center", className)}
    >
      <ul className="flex items-center space-x-1">
        {/* Previous button */}
        <li>
          <button
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            aria-label="Go to previous page"
            className={cn(
              "px-3 py-2 rounded-md text-sm font-medium transition-colors",
              "hover:bg-accent hover:text-accent-foreground",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent"
            )}
          >
            Previous
          </button>
        </li>
        
        {/* Page numbers */}
        {pages.map((page, index) => {
          if (page === '...') {
            return (
              <li key={`ellipsis-${index}`}>
                <span className="px-3 py-2 text-sm text-muted-foreground">
                  …
                </span>
              </li>
            )
          }
          
          const pageNumber = page as number
          const isActive = pageNumber === currentPage
          
          return (
            <li key={pageNumber}>
              <button
                onClick={() => onPageChange(pageNumber)}
                aria-label={`Go to page ${pageNumber}`}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  "px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  "hover:bg-accent hover:text-accent-foreground",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  isActive && "bg-primary text-primary-foreground hover:bg-primary/90"
                )}
              >
                {pageNumber}
              </button>
            </li>
          )
        })}
        
        {/* Next button */}
        <li>
          <button
            onClick={() => onPageChange(currentPage + 1)}
            disabled={currentPage === totalPages}
            aria-label="Go to next page"
            className={cn(
              "px-3 py-2 rounded-md text-sm font-medium transition-colors",
              "hover:bg-accent hover:text-accent-foreground",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent"
            )}
          >
            Next
          </button>
        </li>
      </ul>
      
      {/* Screen reader announcement */}
      <VisuallyHidden aria-live="polite">
        Page {currentPage} of {totalPages}
      </VisuallyHidden>
    </nav>
  )
}