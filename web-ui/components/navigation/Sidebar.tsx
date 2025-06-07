'use client'

import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectSidebarCollapsed,
  selectCurrentSection,
  toggleSidebarCollapsed,
  setCurrentSection,
  setShowMemoryForm,
  toggleFilterPanel,
  setGlobalSearchFocused
} from '@/store/slices/uiSlice'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Brain,
  Database,
  GitBranch,
  Search,
  Settings,
  ChevronLeft,
  ChevronRight,
  Plus,
  Filter,
  BarChart3,
  Globe,
  Layers3,
  Zap,
  Activity,
  Archive
} from 'lucide-react'

interface NavigationItem {
  id: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  section: 'memories' | 'patterns' | 'repositories' | 'settings' | 'graph' | 'performance' | 'realtime' | 'multi-repo' | 'backup'
  badge?: string
  disabled?: boolean
}

const mainNavItems: NavigationItem[] = [
  {
    id: 'memories',
    label: 'Memories',
    icon: Brain,
    section: 'memories',
  },
  {
    id: 'patterns',
    label: 'Patterns',
    icon: BarChart3,
    section: 'patterns',
  },
  {
    id: 'repositories',
    label: 'Repositories',
    icon: GitBranch,
    section: 'repositories',
  },
]

const toolsNavItems: NavigationItem[] = [
  {
    id: 'search',
    label: 'Search',
    icon: Search,
    section: 'memories', // Routes to memories with search active
  },
  {
    id: 'filters',
    label: 'Filters',
    icon: Filter,
    section: 'memories',
  },
  {
    id: 'relationships',
    label: 'Knowledge Graph',
    icon: Layers3,
    section: 'graph',
  },
  {
    id: 'realtime',
    label: 'Realtime Feed',
    icon: Activity,
    section: 'realtime',
  },
]

const bottomNavItems: NavigationItem[] = [
  {
    id: 'backup',
    label: 'Backup',
    icon: Archive,
    section: 'backup',
  },
  {
    id: 'performance',
    label: 'Performance',
    icon: Zap,
    section: 'performance',
  },
  {
    id: 'multi-repo',
    label: 'Multi-Repo',
    icon: Globe,
    section: 'multi-repo',
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: Settings,
    section: 'settings',
  },
]

export function Sidebar() {
  const dispatch = useAppDispatch()
  const isCollapsed = useAppSelector(selectSidebarCollapsed)
  const currentSection = useAppSelector(selectCurrentSection)

  const handleNavigation = (item: NavigationItem) => {
    dispatch(setCurrentSection(item.section))

    // Handle special navigation actions
    if (item.id === 'search') {
      // Set a flag in the UI state to focus search on next render
      dispatch(setGlobalSearchFocused(true))
    } else if (item.id === 'filters') {
      // Open the filter panel
      dispatch(toggleFilterPanel())
    }
  }

  const toggleCollapse = () => {
    dispatch(toggleSidebarCollapsed())
  }

  // Always render sidebar, let MainLayout handle visibility
  // if (!isOpen) {
  //   return null
  // }

  return (
    <div className={cn(
      "flex flex-col h-full bg-background border-r border-border transition-all duration-300",
      isCollapsed ? "w-16" : "w-64"
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-border">
        {!isCollapsed && (
          <div className="flex items-center space-x-2">
            <Database className="h-6 w-6 text-primary" />
            <span className="font-semibold text-foreground">MCP Memory</span>
          </div>
        )}

        <Button
          variant="ghost"
          size="sm"
          onClick={toggleCollapse}
          className="h-8 w-8 p-0"
        >
          {isCollapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* New Memory Button */}
      <div className="p-3">
        <Button
          className="w-full justify-start"
          size={isCollapsed ? "sm" : "default"}
          onClick={() => {
            dispatch(setCurrentSection('memories'))
            dispatch(setShowMemoryForm(true))
          }}
        >
          <Plus className="h-4 w-4" />
          {!isCollapsed && <span className="ml-2">New Memory</span>}
        </Button>
      </div>

      {/* Main Navigation */}
      <div className="flex-1 px-3 space-y-1">
        <div className="space-y-1">
          {!isCollapsed && (
            <div className="px-3 py-2">
              <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Navigate
              </h3>
            </div>
          )}

          {mainNavItems.map((item) => {
            const Icon = item.icon
            const isActive = currentSection === item.section

            return (
              <Button
                key={item.id}
                variant={isActive ? "secondary" : "ghost"}
                className={cn(
                  "w-full justify-start",
                  isCollapsed ? "px-2" : "px-3",
                  isActive && "bg-secondary text-secondary-foreground"
                )}
                onClick={() => handleNavigation(item)}
                disabled={item.disabled}
              >
                <Icon className="h-4 w-4" />
                {!isCollapsed && (
                  <>
                    <span className="ml-3">{item.label}</span>
                    {item.badge && (
                      <Badge variant="secondary" className="ml-auto">
                        {item.badge}
                      </Badge>
                    )}
                  </>
                )}
              </Button>
            )
          })}
        </div>

        <Separator className="my-4" />

        {/* Tools */}
        <div className="space-y-1">
          {!isCollapsed && (
            <div className="px-3 py-2">
              <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Tools
              </h3>
            </div>
          )}

          {toolsNavItems.map((item) => {
            const Icon = item.icon

            return (
              <Button
                key={item.id}
                variant="ghost"
                className={cn(
                  "w-full justify-start",
                  isCollapsed ? "px-2" : "px-3"
                )}
                onClick={() => handleNavigation(item)}
                disabled={item.disabled}
              >
                <Icon className="h-4 w-4" />
                {!isCollapsed && (
                  <>
                    <span className="ml-3">{item.label}</span>
                    {item.badge && (
                      <Badge variant="secondary" className="ml-auto">
                        {item.badge}
                      </Badge>
                    )}
                  </>
                )}
              </Button>
            )
          })}
        </div>
      </div>

      {/* Bottom Navigation */}
      <div className="p-3 border-t border-border">
        <div className="space-y-1">
          {bottomNavItems.map((item) => {
            const Icon = item.icon
            const isActive = currentSection === item.section

            return (
              <Button
                key={item.id}
                variant={isActive ? "secondary" : "ghost"}
                className={cn(
                  "w-full justify-start",
                  isCollapsed ? "px-2" : "px-3",
                  isActive && "bg-secondary text-secondary-foreground"
                )}
                onClick={() => handleNavigation(item)}
                disabled={item.disabled}
              >
                <Icon className="h-4 w-4" />
                {!isCollapsed && (
                  <>
                    <span className="ml-3">{item.label}</span>
                    {item.badge && (
                      <Badge variant="secondary" className="ml-auto">
                        {item.badge}
                      </Badge>
                    )}
                  </>
                )}
              </Button>
            )
          })}
        </div>
      </div>
    </div>
  )
}