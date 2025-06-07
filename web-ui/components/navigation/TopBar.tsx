'use client'

import { useState } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectCurrentSection,
  selectRecentSearches,
  toggleCommandPalette,
  addNotification,
  setGlobalSearchFocused,
  toggleSidebar,
  setCurrentSection
} from '@/store/slices/uiSlice'
import { useTheme } from '@/providers/ThemeProvider'
import { setQuery } from '@/store/slices/filtersSlice'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Menu,
  Search,
  Command,
  Bell,
  Settings,
  User,
  Moon,
  Sun,
  Monitor,
  Github,
  LifeBuoy,
  Zap,
  Activity,
  Server
} from 'lucide-react'
import { GraphQLStatus } from '@/components/GraphQLStatus'

interface TopBarProps {
  className?: string
}

export function TopBar({ className }: TopBarProps) {
  const dispatch = useAppDispatch()
  const currentSection = useAppSelector(selectCurrentSection)
  const recentSearches = useAppSelector(selectRecentSearches)
  const { setTheme } = useTheme()

  const [searchValue, setSearchValue] = useState('')

  const handleSearch = (query: string) => {
    if (query.trim()) {
      dispatch(setQuery(query))
      dispatch(addNotification({
        type: 'info',
        title: 'Search started',
        message: `Searching for "${query}"`,
        duration: 3000
      }))
    }
  }

  const handleSearchKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch(searchValue)
    }
  }

  const getSectionTitle = () => {
    switch (currentSection) {
      case 'memories':
        return 'Memories'
      case 'patterns':
        return 'Patterns & Insights'
      case 'repositories':
        return 'Repositories'
      case 'settings':
        return 'Settings'
      default:
        return 'MCP Memory'
    }
  }

  const getSectionDescription = () => {
    switch (currentSection) {
      case 'memories':
        return 'Browse and search your conversation memories'
      case 'patterns':
        return 'Discover patterns and insights across your memories'
      case 'repositories':
        return 'Manage repository connections and cross-repo learning'
      case 'settings':
        return 'Configure your MCP Memory server'
      default:
        return 'AI Memory Management System'
    }
  }

  return (
    <div className={cn(
      "flex items-center justify-between px-6 py-4 bg-background border-b border-border",
      className
    )}>
      {/* Left section */}
      <div className="flex items-center space-x-4">
        {/* Mobile menu toggle */}
        <Button
          variant="ghost"
          size="sm"
          className="lg:hidden"
          onClick={() => dispatch(toggleSidebar())}
        >
          <Menu className="h-4 w-4" />
        </Button>

        {/* Section info */}
        <div>
          <h1 className="text-xl font-semibold text-foreground">
            {getSectionTitle()}
          </h1>
          <p className="text-sm text-muted-foreground">
            {getSectionDescription()}
          </p>
        </div>
      </div>

      {/* Center section - Search */}
      <div className="flex-1 max-w-md mx-8">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search memories... (⌘K)"
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            onKeyPress={handleSearchKeyPress}
            onFocus={() => dispatch(setGlobalSearchFocused(true))}
            onBlur={() => dispatch(setGlobalSearchFocused(false))}
            className="pl-10 pr-4"
          />

          {/* Search shortcut hint */}
          <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
            <kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
              <span className="text-xs">⌘</span>K
            </kbd>
          </div>
        </div>

        {/* Recent searches dropdown */}
        {recentSearches.length > 0 && (
          <div className="mt-2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="text-xs text-muted-foreground">
                  Recent searches
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="center" className="w-64">
                <DropdownMenuLabel>Recent Searches</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {recentSearches.slice(0, 5).map((search, index) => (
                  <DropdownMenuItem
                    key={index}
                    onClick={() => {
                      setSearchValue(search)
                      handleSearch(search)
                    }}
                  >
                    <Search className="mr-2 h-3 w-3" />
                    {search}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </div>

      {/* Right section */}
      <div className="flex items-center space-x-2">
        {/* Quick actions */}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => dispatch(toggleCommandPalette())}
              >
                <Command className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Command Palette (⌘K)</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>

        {/* GraphQL Status */}
        <GraphQLStatus />

        {/* System status */}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="sm">
                <Activity className="h-4 w-4 text-success" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>System Status: Online</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>

        {/* Notifications */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="relative">
              <Bell className="h-4 w-4" />
              <Badge
                variant="destructive"
                className="absolute -top-1 -right-1 h-2 w-2 p-0"
              />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-80">
            <DropdownMenuLabel>Notifications</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <div className="flex flex-col space-y-1">
                <p className="text-sm">New pattern detected</p>
                <p className="text-xs text-muted-foreground">
                  Found similar bug-fix patterns across 3 repositories
                </p>
              </div>
            </DropdownMenuItem>
            <DropdownMenuItem>
              <div className="flex flex-col space-y-1">
                <p className="text-sm">Memory storage optimized</p>
                <p className="text-xs text-muted-foreground">
                  Cleaned up 150 outdated memory chunks
                </p>
              </div>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* User menu */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              <User className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>My Account</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => dispatch(setCurrentSection('settings'))}>
              <Settings className="mr-2 h-4 w-4" />
              <span>Settings</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => window.open(process.env.NEXT_PUBLIC_GITHUB_ISSUES_URL || 'https://github.com/lerianstudio/lerian-mcp-memory/issues', '_blank')}>
              <LifeBuoy className="mr-2 h-4 w-4" />
              <span>Support</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => window.open(process.env.NEXT_PUBLIC_GITHUB_REPO_URL || 'https://github.com/lerianstudio/lerian-mcp-memory', '_blank')}>
              <Github className="mr-2 h-4 w-4" />
              <span>GitHub</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => dispatch(setCurrentSection('performance'))}>
              <Zap className="mr-2 h-4 w-4" />
              <span>Performance</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => window.open('/api/health', '_blank')}>
              <Server className="mr-2 h-4 w-4" />
              <span>Server Status</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Theme toggle */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
              <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
              <span className="sr-only">Toggle theme</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setTheme('light')}>
              <Sun className="mr-2 h-4 w-4" />
              <span>Light</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTheme('dark')}>
              <Moon className="mr-2 h-4 w-4" />
              <span>Dark</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTheme('system')}>
              <Monitor className="mr-2 h-4 w-4" />
              <span>System</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}