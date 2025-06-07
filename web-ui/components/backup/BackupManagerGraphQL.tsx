/**
 * Backup Manager Component (GraphQL Version)
 * 
 * Provides UI for creating, listing, restoring, and managing memory backups.
 * Uses GraphQL instead of REST API.
 */

'use client'

import { useState } from 'react'
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
  MoreHorizontal,
  HardDrive,
  Archive
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { 
  useListBackups, 
  useCreateBackup, 
  useRestoreBackup, 
  useCleanupBackups,
  type Backup 
} from '@/lib/graphql/hooks-extended'
import { useAppDispatch } from '@/store/store'
import { addNotification } from '@/store/slices/uiSlice'

interface BackupManagerProps {
  className?: string
}

export function BackupManager({ className }: BackupManagerProps) {
  const dispatch = useAppDispatch()
  
  // GraphQL hooks
  const { data, loading: isLoading, error: queryError, refetch: loadBackups } = useListBackups()
  const [createBackup, { loading: isCreating }] = useCreateBackup()
  const [restoreBackupMutation, { loading: isRestoring }] = useRestoreBackup()
  const [cleanupBackupsMutation, { loading: isCleaningUp }] = useCleanupBackups()
  
  const backups = data?.listBackups || []
  
  // Create backup state
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [backupName, setBackupName] = useState('')
  const [backupRepository, setBackupRepository] = useState('')
  const [includeVectors, setIncludeVectors] = useState(true)
  const [compressBackup, setCompressBackup] = useState(true)
  
  // Restore backup state
  const [restoreDialogOpen, setRestoreDialogOpen] = useState(false)
  const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null)
  const [overwriteData, setOverwriteData] = useState(false)

  const handleCreateBackup = async () => {
    try {
      const result = await createBackup({
        variables: {
          name: backupName.trim() || undefined,
          repository: backupRepository.trim() || undefined,
          includeVectors,
          format: 'json',
          compress: compressBackup
        }
      })
      
      if (result.data?.createBackup) {
        dispatch(addNotification({
          type: 'success',
          title: 'Backup Created',
          message: `Successfully backed up ${result.data.createBackup.chunkCount} chunks`,
          duration: 5000
        }))
        
        setCreateDialogOpen(false)
        setBackupName('')
        setBackupRepository('')
        await loadBackups()
      }
    } catch (error) {
      dispatch(addNotification({
        type: 'error',
        title: 'Backup Failed',
        message: error instanceof Error ? error.message : 'Failed to create backup',
        duration: 5000
      }))
    }
  }

  const handleRestoreBackup = async () => {
    if (!selectedBackup?.metadata?.backupFile) return
    
    try {
      const result = await restoreBackupMutation({
        variables: {
          backupFile: selectedBackup.metadata.backupFile,
          overwrite: overwriteData
        }
      })
      
      if (result.data?.restoreBackup.success) {
        dispatch(addNotification({
          type: 'success',
          title: 'Backup Restored',
          message: `Successfully restored ${result.data.restoreBackup.restoredCount || selectedBackup.chunkCount} chunks`,
          duration: 5000
        }))
        
        setRestoreDialogOpen(false)
        setSelectedBackup(null)
        setOverwriteData(false)
      }
    } catch (error) {
      dispatch(addNotification({
        type: 'error',
        title: 'Restore Failed',
        message: error instanceof Error ? error.message : 'Failed to restore backup',
        duration: 5000
      }))
    }
  }

  const handleCleanupBackups = async () => {
    try {
      const result = await cleanupBackupsMutation({
        variables: {
          retentionDays: 30 // Keep backups for 30 days
        }
      })
      
      if (result.data?.cleanupBackups.success) {
        dispatch(addNotification({
          type: 'success',
          title: 'Cleanup Complete',
          message: `Removed ${result.data.cleanupBackups.deletedCount} old backups, freed ${formatFileSize(result.data.cleanupBackups.freedSpace)}`,
          duration: 5000
        }))
        
        await loadBackups()
      }
    } catch (error) {
      dispatch(addNotification({
        type: 'error',
        title: 'Cleanup Failed',
        message: error instanceof Error ? error.message : 'Failed to cleanup backups',
        duration: 5000
      }))
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

  const renderBackupCard = (backup: Backup, index: number) => (
    <Card key={backup.id} className="w-full">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-lg flex items-center gap-2">
              <Archive className="h-5 w-5" />
              {backup.metadata?.backupFile || `Backup #${index + 1}`}
            </CardTitle>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Calendar className="h-4 w-4" />
              {formatDate(backup.createdAt)}
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
            <span>{backup.chunkCount} chunks</span>
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
          <Button onClick={() => loadBackups()} disabled={isLoading} variant="outline">
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
                
                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="include-vectors"
                    checked={includeVectors}
                    onCheckedChange={(checked) => setIncludeVectors(checked as boolean)}
                  />
                  <Label htmlFor="include-vectors" className="text-sm">
                    Include vector embeddings
                  </Label>
                </div>
                
                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="compress"
                    checked={compressBackup}
                    onCheckedChange={(checked) => setCompressBackup(checked as boolean)}
                  />
                  <Label htmlFor="compress" className="text-sm">
                    Compress backup file
                  </Label>
                </div>
              </div>
              
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setCreateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button onClick={handleCreateBackup} disabled={isCreating}>
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
      {queryError && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{queryError.message}</AlertDescription>
        </Alert>
      )}

      {/* Backup Actions */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Quick Actions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2">
            <Button 
              variant="outline" 
              onClick={handleCleanupBackups} 
              disabled={isCleaningUp || isLoading}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Cleanup Old Backups (30+ days)
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
                  <div>Created: {formatDate(selectedBackup.createdAt)}</div>
                  <div>Chunks: {selectedBackup.chunkCount}</div>
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
            <Button onClick={handleRestoreBackup} disabled={isRestoring}>
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