'use client'

import { useAppSelector, useAppDispatch } from '@/store/store'
import { 
  selectSelectedMemory,
  selectRelatedMemories,
  setSelectedMemory 
} from '@/store/slices/memoriesSlice'
import { cn, formatDate, getMemoryTypeIcon, getConfidenceColor, formatConfidence } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  X,
  GitBranch,
  Clock,
  User,
  Tag,
  Link,
  Star,
  Share,
  Download,
  Edit,
  Trash2,
  ArrowUpRight,
  Brain,
  Activity
} from 'lucide-react'

interface MemoryDetailsProps {
  className?: string
}

export function MemoryDetails({ className }: MemoryDetailsProps) {
  const dispatch = useAppDispatch()
  const selectedMemory = useAppSelector(selectSelectedMemory)
  const relatedMemories = useAppSelector(selectRelatedMemories)

  if (!selectedMemory) {
    return (
      <div className={cn(
        "flex items-center justify-center h-full text-center",
        className
      )}>
        <div className="max-w-md space-y-4">
          <div className="text-6xl">🧠</div>
          <h3 className="text-lg font-medium text-foreground">
            Select a memory
          </h3>
          <p className="text-sm text-muted-foreground">
            Choose a memory from the list to view its details and relationships
          </p>
        </div>
      </div>
    )
  }

  const handleClose = () => {
    dispatch(setSelectedMemory(undefined))
  }

  const handleRelatedMemoryClick = (memory: any) => {
    dispatch(setSelectedMemory(memory))
  }

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="flex items-center justify-between p-6 border-b border-border">
        <div className="flex items-center space-x-3">
          <span className="text-2xl">
            {getMemoryTypeIcon(selectedMemory.type)}
          </span>
          <div>
            <h2 className="text-xl font-semibold text-foreground">
              Memory Details
            </h2>
            <p className="text-sm text-muted-foreground">
              {formatDate(selectedMemory.timestamp)}
            </p>
          </div>
        </div>
        
        <div className="flex items-center space-x-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="sm">
                  <Star className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Add to favorites</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="sm">
                  <Share className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Share memory</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="sm">
                  <Download className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Export memory</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          
          <Button variant="ghost" size="sm" onClick={handleClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-6 space-y-6">
          {/* Memory metadata */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center space-x-2">
                <Activity className="h-5 w-5" />
                <span>Metadata</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Type</label>
                  <div className="flex items-center space-x-2 mt-1">
                    <Badge 
                      variant="outline" 
                      className={cn("text-xs", getConfidenceColor(selectedMemory.metadata.confidence?.score || 0))}
                    >
                      {selectedMemory.type.replace('_', ' ')}
                    </Badge>
                  </div>
                </div>
                
                {selectedMemory.metadata.confidence && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Confidence</label>
                    <div className="mt-1">
                      <Badge 
                        variant="secondary" 
                        className={cn("text-xs", getConfidenceColor(selectedMemory.metadata.confidence.score))}
                      >
                        {formatConfidence(selectedMemory.metadata.confidence.score)}
                      </Badge>
                    </div>
                  </div>
                )}
                
                {selectedMemory.metadata.repository && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Repository</label>
                    <div className="flex items-center space-x-1 mt-1">
                      <GitBranch className="h-3 w-3 text-muted-foreground" />
                      <span className="text-sm">{selectedMemory.metadata.repository}</span>
                    </div>
                  </div>
                )}
                
                {selectedMemory.session_id && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Session</label>
                    <div className="flex items-center space-x-1 mt-1">
                      <User className="h-3 w-3 text-muted-foreground" />
                      <span className="text-sm font-mono">{selectedMemory.session_id}</span>
                    </div>
                  </div>
                )}
              </div>
              
              <div>
                <label className="text-sm font-medium text-muted-foreground">Created</label>
                <div className="flex items-center space-x-1 mt-1">
                  <Clock className="h-3 w-3 text-muted-foreground" />
                  <span className="text-sm">{formatDate(selectedMemory.timestamp)}</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Tags */}
          {selectedMemory.metadata.tags && selectedMemory.metadata.tags.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Tag className="h-5 w-5" />
                  <span>Tags</span>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {selectedMemory.metadata.tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Content */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center space-x-2">
                <Brain className="h-5 w-5" />
                <span>Content</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="prose prose-sm dark:prose-invert max-w-none">
                <p className="whitespace-pre-wrap leading-relaxed">
                  {selectedMemory.content}
                </p>
              </div>
            </CardContent>
          </Card>

          {/* Related memories */}
          {relatedMemories.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Link className="h-5 w-5" />
                  <span>Related Memories</span>
                  <Badge variant="secondary" className="ml-auto">
                    {relatedMemories.length}
                  </Badge>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {relatedMemories.slice(0, 5).map((memory) => (
                    <div
                      key={memory.id}
                      className="flex items-start space-x-3 p-3 border border-border rounded-lg hover:bg-muted/50 cursor-pointer transition-colors"
                      onClick={() => handleRelatedMemoryClick(memory)}
                    >
                      <span className="text-sm">
                        {getMemoryTypeIcon(memory.type)}
                      </span>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center space-x-2 mb-1">
                          <Badge variant="outline" className="text-xs">
                            {memory.type.replace('_', ' ')}
                          </Badge>
                          <span className="text-xs text-muted-foreground">
                            {formatDate(memory.timestamp)}
                          </span>
                        </div>
                        <p className="text-sm text-foreground line-clamp-2">
                          {memory.content.slice(0, 150)}...
                        </p>
                      </div>
                      <ArrowUpRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    </div>
                  ))}
                  
                  {relatedMemories.length > 5 && (
                    <Button variant="ghost" className="w-full">
                      View all {relatedMemories.length} related memories
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Actions */}
          <Card>
            <CardHeader>
              <CardTitle>Actions</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm">
                  <Edit className="mr-2 h-4 w-4" />
                  Edit Memory
                </Button>
                <Button variant="outline" size="sm">
                  <Link className="mr-2 h-4 w-4" />
                  Create Relationship
                </Button>
                <Button variant="outline" size="sm">
                  <Download className="mr-2 h-4 w-4" />
                  Export
                </Button>
                <Button variant="destructive" size="sm">
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </ScrollArea>
    </div>
  )
}