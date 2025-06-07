/**
 * Backup Manager Component
 * 
 * Provides UI for creating, listing, restoring, and managing memory backups.
 * Integrates with the backup API endpoints and Go backend.
 */

'use client'

import { useState, useEffect } from 'react'
import { useCSRF } from '@/providers/CSRFProvider'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Download,
  Upload,
  Trash2,
  RefreshCw,
  Database,
  Calendar,
  FileText,
  AlertTriangle,
  CheckCircle,
  MoreHorizontal,
  HardDrive,
  Archive
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface BackupMetadata {
  version: string
  created_at: string
  chunk_count: number
  size: number
  repository?: string
  metadata?: {
    backup_file?: string
    compression?: string
    format?: string
  }
}

interface BackupManagerProps {
  className?: string
}

export function BackupManager({ className }: BackupManagerProps) {
  const { makeProtectedRequest } = useCSRF()
  
  const [backups, setBackups] = useState<BackupMetadata[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  
  // Create backup state
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [backupName, setBackupName] = useState('')
  const [backupRepository, setBackupRepository] = useState('')
  const [isCreating, setIsCreating] = useState(false)
  
  // Restore backup state
  const [restoreDialogOpen, setRestoreDialogOpen] = useState(false)
  const [selectedBackup, setSelectedBackup] = useState<BackupMetadata | null>(null)
  const [overwriteData, setOverwriteData] = useState(false)
  const [isRestoring, setIsRestoring] = useState(false)

  // Load backups on component mount
  useEffect(() => {
    loadBackups()
  }, [])

  const loadBackups = async () => {
    setIsLoading(true)
    setError(null)
    
    try {
      const response = await fetch('/api/backup', {
        method: 'GET',
        credentials: 'include'
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to load backups')
      }
      
      const data = await response.json()
      setBackups(data.backups || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load backups')
    } finally {
      setIsLoading(false)
    }
  }

  const createBackup = async () => {
    setIsCreating(true)
    setError(null)
    setSuccess(null)
    
    try {
      const response = await makeProtectedRequest('/api/backup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: backupName.trim() || undefined,
          repository: backupRepository.trim() || undefined
        })
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create backup')
      }
      
      const result = await response.json()
      setSuccess(`Backup created successfully: ${result.chunk_count} chunks backed up`)
      setCreateDialogOpen(false)
      setBackupName('')
      setBackupRepository('')
      await loadBackups()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create backup')
    } finally {
      setIsCreating(false)
    }
  }

  const restoreBackup = async () => {
    if (!selectedBackup?.metadata?.backup_file) return
    
    setIsRestoring(true)
    setError(null)
    setSuccess(null)
    
    try {
      const response = await makeProtectedRequest('/api/backup', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          file: selectedBackup.metadata.backup_file,
          overwrite: overwriteData,
          validateIntegrity: true
        })
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to restore backup')
      }
      
      setSuccess(`Backup restored successfully: ${selectedBackup.chunk_count} chunks restored`)
      setRestoreDialogOpen(false)
      setSelectedBackup(null)
      setOverwriteData(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to restore backup')
    } finally {
      setIsRestoring(false)
    }
  }

  const cleanupBackups = async () => {
    setIsLoading(true)
    setError(null)
    setSuccess(null)
    
    try {
      const response = await makeProtectedRequest('/api/backup', {
        method: 'PATCH'
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to cleanup backups')
      }
      
      setSuccess('Old backups cleaned up successfully')
      await loadBackups()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cleanup backups')
    } finally {
      setIsLoading(false)
    }
  }

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = (dateString: string): string => {
    return new Date(dateString).toLocaleString()
  }

  const renderBackupCard = (backup: BackupMetadata, index: number) => (
    <Card key={index} className="w-full">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-lg flex items-center gap-2">
              <Archive className="h-5 w-5" />
              Backup #{index + 1}
            </CardTitle>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Calendar className="h-4 w-4" />
              {formatDate(backup.created_at)}
            </div>
          </div>
          
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={() => {
                  setSelectedBackup(backup)
                  setRestoreDialogOpen(true)
                }}
              >
                <Upload className="mr-2 h-4 w-4" />
                Restore
              </DropdownMenuItem>
              <DropdownMenuItem className="text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div className="flex items-center gap-2">
            <Database className="h-4 w-4 text-muted-foreground" />
            <span>{backup.chunk_count} chunks</span>
          </div>
          <div className="flex items-center gap-2">
            <HardDrive className="h-4 w-4 text-muted-foreground" />
            <span>{formatFileSize(backup.size)}</span>
          </div>
        </div>
        
        {backup.repository && (
          <div className="flex items-center gap-2 text-sm">
            <FileText className="h-4 w-4 text-muted-foreground" />
            <span className="truncate">{backup.repository}</span>
          </div>
        )}
        
        <div className="flex gap-1">
          <Badge variant="secondary" className="text-xs">
            v{backup.version}
          </Badge>
          {backup.metadata?.compression && (
            <Badge variant="outline" className="text-xs">
              {backup.metadata.compression}
            </Badge>
          )}
          {backup.metadata?.format && (
            <Badge variant="outline" className="text-xs">
              {backup.metadata.format}
            </Badge>
          )}
        </div>
      </CardContent>
    </Card>
  )

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Backup Manager</h2>
          <p className="text-muted-foreground">
            Create, restore, and manage memory data backups
          </p>
        </div>
        
        <div className="flex gap-2">
          <Button onClick={loadBackups} disabled={isLoading} variant="outline">
            <RefreshCw className={cn("h-4 w-4 mr-2", isLoading && "animate-spin")} />
            Refresh
          </Button>
          
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Download className="h-4 w-4 mr-2" />
                Create Backup
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Backup</DialogTitle>
                <DialogDescription>
                  Create a backup of your memory data. You can optionally filter by repository.
                </DialogDescription>
              </DialogHeader>
              
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="backup-name">Backup Name (optional)</Label>
                  <Input
                    id="backup-name"
                    value={backupName}
                    onChange={(e) => setBackupName(e.target.value)}
                    placeholder="my-backup"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label htmlFor="backup-repository">Repository Filter (optional)</Label>
                  <Input
                    id="backup-repository"
                    value={backupRepository}
                    onChange={(e) => setBackupRepository(e.target.value)}
                    placeholder="github.com/user/repo"
                  />
                </div>
              </div>
              
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setCreateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button onClick={createBackup} disabled={isCreating}>
                  {isCreating ? (
                    <>
                      <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Download className="h-4 w-4 mr-2" />
                      Create Backup
                    </>
                  )}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Status Messages */}
      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      
      {success && (
        <Alert>
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}

      {/* Backup Actions */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Quick Actions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2">
            <Button variant="outline" onClick={cleanupBackups} disabled={isLoading}>
              <Trash2 className="h-4 w-4 mr-2" />
              Cleanup Old Backups
            </Button>
          </div>
        </CardContent>
      </Card>

      <Separator />

      {/* Backup List */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-xl font-semibold">Available Backups</h3>
          <Badge variant="secondary">
            {backups.length} {backups.length === 1 ? 'backup' : 'backups'}
          </Badge>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <Card key={i} className="animate-pulse">
                <CardHeader>
                  <div className="h-4 bg-muted rounded w-3/4" />
                  <div className="h-3 bg-muted rounded w-1/2" />
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="h-3 bg-muted rounded" />
                    <div className="h-3 bg-muted rounded w-2/3" />
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : backups.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12">
              <Archive className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium text-center mb-2">No backups found</h3>
              <p className="text-muted-foreground text-center mb-4">
                Create your first backup to protect your memory data
              </p>
              <Button onClick={() => setCreateDialogOpen(true)}>
                <Download className="h-4 w-4 mr-2" />
                Create First Backup
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {backups.map((backup, index) => renderBackupCard(backup, index))}
          </div>
        )}
      </div>

      {/* Restore Dialog */}
      <Dialog open={restoreDialogOpen} onOpenChange={setRestoreDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Restore Backup</DialogTitle>
            <DialogDescription>
              This will restore data from the selected backup. Choose your options carefully.
            </DialogDescription>
          </DialogHeader>
          
          {selectedBackup && (
            <div className="space-y-4">
              <div className="p-4 bg-muted rounded-lg space-y-2">
                <div className="font-medium">Backup Details:</div>
                <div className="text-sm space-y-1">
                  <div>Created: {formatDate(selectedBackup.created_at)}</div>
                  <div>Chunks: {selectedBackup.chunk_count}</div>
                  <div>Size: {formatFileSize(selectedBackup.size)}</div>
                  {selectedBackup.repository && (
                    <div>Repository: {selectedBackup.repository}</div>
                  )}
                </div>
              </div>
              
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="overwrite"
                  checked={overwriteData}
                  onCheckedChange={(checked) => setOverwriteData(checked as boolean)}
                />
                <Label htmlFor="overwrite" className="text-sm">
                  Overwrite existing data (if any conflicts occur)
                </Label>
              </div>
              
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  Restoring a backup will add the backed up data to your current memory store.
                  {overwriteData && " Existing conflicting data will be overwritten."}
                </AlertDescription>
              </Alert>
            </div>
          )}
          
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setRestoreDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button onClick={restoreBackup} disabled={isRestoring}>
              {isRestoring ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Restoring...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4 mr-2" />
                  Restore Backup
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}