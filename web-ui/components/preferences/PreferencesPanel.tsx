/**
 * Preferences Panel Component
 * 
 * Provides UI for managing user preferences with
 * import/export functionality and real-time updates
 */

'use client'

import { useState } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectPreferences,
  selectLastSaved,
  setTheme,
  setLayout,
  setEnableAnimations,
  setEnableRealtime,
  setKeyboardShortcutsEnabled,
  setCacheEnabled,
  setDebugMode,
  setMemoryListLayout,
  setMemoriesPerPage,
  setAutoBackupEnabled,
  setBackupFrequency,
  setNotificationSound,
  setAutoReconnect,
  setReconnectDelay,
  resetPreferences,
  importPreferences,
  exportPreferencesThunk
} from '@/store/slices/preferencesSlice'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Slider } from '@/components/ui/slider'
import {
  Download,
  Upload,
  RefreshCw,
  Settings,
  Palette,
  Layout,
  Zap,
  Shield,
  Bell,
  CheckCircle
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useToast } from '@/components/ui/use-toast'

interface PreferencesPanelProps {
  className?: string
}

export function PreferencesPanel({ className }: PreferencesPanelProps) {
  const dispatch = useAppDispatch()
  const preferences = useAppSelector(selectPreferences)
  const lastSaved = useAppSelector(selectLastSaved)
  const { toast } = useToast()
  
  const [activeTab, setActiveTab] = useState('appearance')
  const [importFile, setImportFile] = useState<File | null>(null)

  const handleExport = () => {
    const json = exportPreferencesThunk()
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `mcp-memory-preferences-${new Date().toISOString().split('T')[0]}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    
    toast({
      title: 'Preferences Exported',
      description: 'Your preferences have been exported successfully',
    })
  }

  const handleImport = async () => {
    if (!importFile) return

    try {
      const text = await importFile.text()
      dispatch(importPreferences(text))
      setImportFile(null)
      
      toast({
        title: 'Preferences Imported',
        description: 'Your preferences have been imported successfully',
      })
    } catch {
      toast({
        title: 'Import Failed',
        description: 'Failed to import preferences. Please check the file format.',
        variant: 'destructive'
      })
    }
  }

  const handleReset = () => {
    dispatch(resetPreferences())
    toast({
      title: 'Preferences Reset',
      description: 'All preferences have been reset to defaults',
    })
  }

  return (
    <div className={cn("space-y-6", className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold flex items-center gap-2">
            <Settings className="h-6 w-6" />
            User Preferences
          </h2>
          <p className="text-sm text-muted-foreground">
            Customize your experience and manage settings
          </p>
        </div>

        <div className="flex items-center gap-4">
          {lastSaved && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <CheckCircle className="h-4 w-4 text-success" />
              <span>Auto-saved</span>
            </div>
          )}

          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleExport}>
              <Download className="mr-2 h-4 w-4" />
              Export
            </Button>
            
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <Upload className="mr-2 h-4 w-4" />
                  Import
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Import Preferences</AlertDialogTitle>
                  <AlertDialogDescription>
                    Select a preferences file to import. This will overwrite your current settings.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <div className="py-4">
                  <Input
                    type="file"
                    accept=".json"
                    onChange={(e) => setImportFile(e.target.files?.[0] || null)}
                  />
                </div>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction onClick={handleImport} disabled={!importFile}>
                    Import
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>

            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <RefreshCw className="mr-2 h-4 w-4" />
                  Reset
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Reset Preferences</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to reset all preferences to their default values?
                    This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction onClick={handleReset} className="bg-destructive text-destructive-foreground">
                    Reset All
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="appearance" className="flex items-center gap-2">
            <Palette className="h-4 w-4" />
            <span className="hidden sm:inline">Appearance</span>
          </TabsTrigger>
          <TabsTrigger value="layout" className="flex items-center gap-2">
            <Layout className="h-4 w-4" />
            <span className="hidden sm:inline">Layout</span>
          </TabsTrigger>
          <TabsTrigger value="performance" className="flex items-center gap-2">
            <Zap className="h-4 w-4" />
            <span className="hidden sm:inline">Performance</span>
          </TabsTrigger>
          <TabsTrigger value="advanced" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            <span className="hidden sm:inline">Advanced</span>
          </TabsTrigger>
          <TabsTrigger value="notifications" className="flex items-center gap-2">
            <Bell className="h-4 w-4" />
            <span className="hidden sm:inline">Notifications</span>
          </TabsTrigger>
        </TabsList>

        {/* Appearance Tab */}
        <TabsContent value="appearance" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Theme & Colors</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="theme">Theme</Label>
                <Select
                  value={preferences.theme}
                  onValueChange={(value: 'light' | 'dark' | 'system') => dispatch(setTheme(value))}
                >
                  <SelectTrigger id="theme">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="light">Light</SelectItem>
                    <SelectItem value="dark">Dark</SelectItem>
                    <SelectItem value="system">System</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label>Enable Animations</Label>
                  <p className="text-xs text-muted-foreground">
                    Smooth transitions and hover effects
                  </p>
                </div>
                <Switch
                  checked={preferences.enableAnimations}
                  onCheckedChange={(checked) => dispatch(setEnableAnimations(checked))}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Layout Tab */}
        <TabsContent value="layout" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Display Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="layout">Layout Style</Label>
                <Select
                  value={preferences.layout}
                  onValueChange={(value: 'default' | 'compact' | 'comfortable') => dispatch(setLayout(value))}
                >
                  <SelectTrigger id="layout">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="default">Default</SelectItem>
                    <SelectItem value="compact">Compact</SelectItem>
                    <SelectItem value="comfortable">Comfortable</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <Label htmlFor="memoryLayout">Memory List Layout</Label>
                <Select
                  value={preferences.memoryListLayout}
                  onValueChange={(value: 'grid' | 'list') => dispatch(setMemoryListLayout(value))}
                >
                  <SelectTrigger id="memoryLayout">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="list">List View</SelectItem>
                    <SelectItem value="grid">Grid View</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <Label htmlFor="memoriesPerPage">Memories Per Page</Label>
                <div className="flex items-center gap-4">
                  <Slider
                    id="memoriesPerPage"
                    min={10}
                    max={50}
                    step={5}
                    value={[preferences.memoriesPerPage]}
                    onValueChange={([value]) => dispatch(setMemoriesPerPage(value))}
                    className="flex-1"
                  />
                  <Badge variant="secondary" className="w-12 text-center">
                    {preferences.memoriesPerPage}
                  </Badge>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Performance Tab */}
        <TabsContent value="performance" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Performance Optimization</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Enable Cache</Label>
                  <p className="text-xs text-muted-foreground">
                    Cache search results for faster loading
                  </p>
                </div>
                <Switch
                  checked={preferences.cacheEnabled}
                  onCheckedChange={(checked) => dispatch(setCacheEnabled(checked))}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label>Real-time Updates</Label>
                  <p className="text-xs text-muted-foreground">
                    Enable WebSocket for live memory updates
                  </p>
                </div>
                <Switch
                  checked={preferences.enableRealtime}
                  onCheckedChange={(checked) => dispatch(setEnableRealtime(checked))}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label>Debug Mode</Label>
                  <p className="text-xs text-muted-foreground">
                    Show detailed logging in console
                  </p>
                </div>
                <Switch
                  checked={preferences.debugMode}
                  onCheckedChange={(checked) => dispatch(setDebugMode(checked))}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>WebSocket Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Auto-Reconnect</Label>
                  <p className="text-xs text-muted-foreground">
                    Automatically reconnect on connection loss
                  </p>
                </div>
                <Switch
                  checked={preferences.autoReconnect}
                  onCheckedChange={(checked) => dispatch(setAutoReconnect(checked))}
                />
              </div>

              {preferences.autoReconnect && (
                <div>
                  <Label htmlFor="reconnectDelay">Reconnect Delay (ms)</Label>
                  <div className="flex items-center gap-4">
                    <Slider
                      id="reconnectDelay"
                      min={1000}
                      max={30000}
                      step={1000}
                      value={[preferences.reconnectDelay]}
                      onValueChange={([value]) => dispatch(setReconnectDelay(value))}
                      className="flex-1"
                    />
                    <Badge variant="secondary" className="min-w-[80px] text-center">
                      {(preferences.reconnectDelay / 1000).toFixed(0)}s
                    </Badge>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Advanced Tab */}
        <TabsContent value="advanced" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Backup Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Auto-Backup</Label>
                  <p className="text-xs text-muted-foreground">
                    Automatically backup your memories
                  </p>
                </div>
                <Switch
                  checked={preferences.autoBackupEnabled}
                  onCheckedChange={(checked) => dispatch(setAutoBackupEnabled(checked))}
                />
              </div>

              {preferences.autoBackupEnabled && (
                <div>
                  <Label htmlFor="backupFrequency">Backup Frequency</Label>
                  <Select
                    value={preferences.backupFrequency}
                    onValueChange={(value: 'daily' | 'weekly' | 'monthly') => dispatch(setBackupFrequency(value))}
                  >
                    <SelectTrigger id="backupFrequency">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="daily">Daily</SelectItem>
                      <SelectItem value="weekly">Weekly</SelectItem>
                      <SelectItem value="monthly">Monthly</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Keyboard Shortcuts</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div>
                  <Label>Enable Keyboard Shortcuts</Label>
                  <p className="text-xs text-muted-foreground">
                    Use Cmd/Ctrl+K for command palette and more
                  </p>
                </div>
                <Switch
                  checked={preferences.keyboardShortcutsEnabled}
                  onCheckedChange={(checked) => dispatch(setKeyboardShortcutsEnabled(checked))}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Notifications Tab */}
        <TabsContent value="notifications" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Notification Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Notification Sounds</Label>
                  <p className="text-xs text-muted-foreground">
                    Play sounds for important notifications
                  </p>
                </div>
                <Switch
                  checked={preferences.notificationSound}
                  onCheckedChange={(checked) => dispatch(setNotificationSound(checked))}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}