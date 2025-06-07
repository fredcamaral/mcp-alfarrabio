'use client';

import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useDispatch, useSelector } from 'react-redux';
import { X, Search, Home, Brain, Settings, Database, Activity, FileText, GitBranch, RefreshCw, Download, Upload, Moon, Sun } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { setCommandPaletteOpen, toggleTheme, addNotification } from '@/store/slices/uiSlice';
import { RootState } from '@/store/store';

interface Command {
  id: string;
  title: string;
  description?: string;
  icon: React.ComponentType<{ className?: string }>;
  shortcut?: string;
  category: 'navigation' | 'actions' | 'settings';
  action: () => void;
}

export function CommandPalette() {
  const router = useRouter();
  const dispatch = useDispatch();
  const isOpen = useSelector((state: RootState) => state.ui.commandPaletteOpen);
  const theme = useSelector((state: RootState) => state.ui.theme);
  
  const [search, setSearch] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const commands: Command[] = [
    // Navigation commands
    {
      id: 'home',
      title: 'Go to Home',
      description: 'Navigate to the main dashboard',
      icon: Home,
      shortcut: 'G H',
      category: 'navigation',
      action: () => {
        router.push('/');
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'memories',
      title: 'Go to Memories',
      description: 'View and manage stored memories',
      icon: Brain,
      shortcut: 'G M',
      category: 'navigation',
      action: () => {
        router.push('/');
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'patterns',
      title: 'Go to Patterns',
      description: 'View learned patterns and insights',
      icon: Activity,
      shortcut: 'G P',
      category: 'navigation',
      action: () => {
        router.push('/?view=patterns');
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'repositories',
      title: 'Go to Repositories',
      description: 'Manage repository connections',
      icon: GitBranch,
      shortcut: 'G R',
      category: 'navigation',
      action: () => {
        router.push('/?view=repositories');
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'config',
      title: 'Go to Configuration',
      description: 'System configuration and settings',
      icon: Settings,
      shortcut: 'G C',
      category: 'navigation',
      action: () => {
        router.push('/?view=config');
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'backup',
      title: 'Go to Backup',
      description: 'Backup and restore data',
      icon: Database,
      shortcut: 'G B',
      category: 'navigation',
      action: () => {
        router.push('/?view=backup');
        dispatch(setCommandPaletteOpen(false));
      }
    },
    
    // Action commands
    {
      id: 'refresh',
      title: 'Refresh Data',
      description: 'Reload memories and patterns',
      icon: RefreshCw,
      shortcut: 'R',
      category: 'actions',
      action: () => {
        window.location.reload();
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'export',
      title: 'Export Memories',
      description: 'Export memories to JSON file',
      icon: Download,
      category: 'actions',
      action: () => {
        // TODO: Implement export functionality via Redux action or API
        // For now, just close the palette and show a notification
        dispatch(addNotification({
          type: 'info',
          title: 'Export Feature',
          message: 'Export functionality will be available soon',
          duration: 3000
        }));
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'import',
      title: 'Import Memories',
      description: 'Import memories from file',
      icon: Upload,
      category: 'actions',
      action: () => {
        // TODO: Implement import functionality via Redux action or API
        // For now, just close the palette and show a notification
        dispatch(addNotification({
          type: 'info',
          title: 'Import Feature',
          message: 'Import functionality will be available soon',
          duration: 3000
        }));
        dispatch(setCommandPaletteOpen(false));
      }
    },
    {
      id: 'new-memory',
      title: 'Create New Memory',
      description: 'Add a new memory entry',
      icon: FileText,
      shortcut: 'N',
      category: 'actions',
      action: () => {
        // Trigger new memory dialog
        const newMemoryButton = document.querySelector('[data-new-memory-button]') as HTMLButtonElement;
        if (newMemoryButton) {
          newMemoryButton.click();
        }
        dispatch(setCommandPaletteOpen(false));
      }
    },
    
    // Settings commands
    {
      id: 'toggle-theme',
      title: `Switch to ${theme === 'dark' ? 'Light' : 'Dark'} Mode`,
      description: 'Toggle between light and dark theme',
      icon: theme === 'dark' ? Sun : Moon,
      shortcut: 'T',
      category: 'settings',
      action: () => {
        dispatch(toggleTheme());
        dispatch(setCommandPaletteOpen(false));
      }
    }
  ];

  const filteredCommands = commands.filter(command => {
    const searchLower = search.toLowerCase();
    return (
      command.title.toLowerCase().includes(searchLower) ||
      command.description?.toLowerCase().includes(searchLower) ||
      command.category.toLowerCase().includes(searchLower)
    );
  });

  const groupedCommands = filteredCommands.reduce((acc, command) => {
    if (!acc[command.category]) {
      acc[command.category] = [];
    }
    acc[command.category].push(command);
    return acc;
  }, {} as Record<string, Command[]>);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!isOpen) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex(prev => 
          prev < filteredCommands.length - 1 ? prev + 1 : 0
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex(prev => 
          prev > 0 ? prev - 1 : filteredCommands.length - 1
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (filteredCommands[selectedIndex]) {
          filteredCommands[selectedIndex].action();
        }
        break;
      case 'Escape':
        e.preventDefault();
        dispatch(setCommandPaletteOpen(false));
        break;
    }
  }, [isOpen, filteredCommands, selectedIndex, dispatch]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  useEffect(() => {
    if (isOpen) {
      setSearch('');
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [isOpen]);

  useEffect(() => {
    // Scroll selected item into view
    if (listRef.current) {
      const selectedElement = listRef.current.querySelector(`[data-index="${selectedIndex}"]`);
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: 'nearest' });
      }
    }
  }, [selectedIndex]);

  const categoryLabels = {
    navigation: 'Navigation',
    actions: 'Actions',
    settings: 'Settings'
  };

  let commandIndex = 0;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => dispatch(setCommandPaletteOpen(open))}>
      <DialogContent className="p-0 gap-0 max-w-2xl">
        <DialogHeader className="px-4 py-3 border-b">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              ref={inputRef}
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setSelectedIndex(0);
              }}
              placeholder="Type a command or search..."
              className="pl-10 pr-10 h-12 text-base border-0 focus-visible:ring-0"
            />
            <button
              onClick={() => dispatch(setCommandPaletteOpen(false))}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </DialogHeader>
        
        <ScrollArea className="max-h-[400px]">
          <div ref={listRef} className="px-2 py-2">
            {Object.entries(groupedCommands).map(([category, categoryCommands]) => (
              <div key={category} className="mb-4">
                <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                  {categoryLabels[category as keyof typeof categoryLabels]}
                </div>
                {categoryCommands.map((command) => {
                  const currentIndex = commandIndex++;
                  const Icon = command.icon;
                  
                  return (
                    <button
                      key={command.id}
                      data-index={currentIndex}
                      onClick={() => command.action()}
                      className={cn(
                        "w-full px-2 py-2 rounded-md flex items-center gap-3 text-left transition-colors",
                        currentIndex === selectedIndex
                          ? "bg-accent text-accent-foreground"
                          : "hover:bg-accent/50 text-foreground"
                      )}
                    >
                      <Icon className="h-4 w-4 flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm">{command.title}</div>
                        {command.description && (
                          <div className="text-xs text-muted-foreground truncate">
                            {command.description}
                          </div>
                        )}
                      </div>
                      {command.shortcut && (
                        <Badge variant="outline" className="ml-auto flex-shrink-0">
                          {command.shortcut}
                        </Badge>
                      )}
                    </button>
                  );
                })}
              </div>
            ))}
            
            {filteredCommands.length === 0 && (
              <div className="py-8 text-center text-sm text-muted-foreground">
                No commands found
              </div>
            )}
          </div>
        </ScrollArea>
        
        <div className="px-4 py-2 border-t bg-muted/50 text-xs text-muted-foreground flex items-center gap-4">
          <span className="flex items-center gap-1">
            <kbd className="px-1.5 py-0.5 bg-background rounded border">↑</kbd>
            <kbd className="px-1.5 py-0.5 bg-background rounded border">↓</kbd>
            to navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1.5 py-0.5 bg-background rounded border">↵</kbd>
            to select
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1.5 py-0.5 bg-background rounded border">esc</kbd>
            to close
          </span>
        </div>
      </DialogContent>
    </Dialog>
  );
}