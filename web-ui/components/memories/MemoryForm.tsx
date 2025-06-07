/**
 * Memory Form Component with CSRF Protection
 * 
 * Example form for creating/updating memory entries with built-in CSRF protection.
 * Demonstrates proper integration of ProtectedForm component.
 */

'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { ProtectedForm, CSRFTokenDisplay } from '@/components/shared/ProtectedForm'
import { logger } from '@/lib/logger'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { X, Plus, Save, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

// Form validation schema
const memoryFormSchema = z.object({
  content: z.string().min(10, 'Content must be at least 10 characters'),
  type: z.enum(['problem', 'solution', 'architecture_decision', 'session_summary', 'code_change', 'discussion', 'analysis', 'verification', 'question']),
  repository: z.string().min(1, 'Repository is required'),
  sessionId: z.string().optional(),
  tags: z.array(z.string()).default([]),
  isPublic: z.boolean().default(false),
  priority: z.enum(['low', 'medium', 'high']).default('medium')
})

type MemoryFormData = z.infer<typeof memoryFormSchema>

interface MemoryFormProps {
  initialData?: Partial<MemoryFormData>
  onSuccess?: (data: MemoryFormData) => void
  onCancel?: () => void
  className?: string
}

const MEMORY_TYPES = [
  { value: 'discussion', label: 'Discussion', icon: '💬' },
  { value: 'solution', label: 'Solution', icon: '💡' },
  { value: 'problem', label: 'Problem', icon: '❓' },
  { value: 'architecture_decision', label: 'Architecture Decision', icon: '🏗️' },
  { value: 'bug_report', label: 'Bug Report', icon: '🐛' },
  { value: 'feature_request', label: 'Feature Request', icon: '✨' }
] as const

export function MemoryForm({ 
  initialData, 
  onSuccess, 
  onCancel, 
  className 
}: MemoryFormProps) {
  const [tags, setTags] = useState<string[]>(initialData?.tags || [])
  const [newTag, setNewTag] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const {
    register,
    setValue,
    watch,
    formState: { errors }
  } = useForm<MemoryFormData>({
    resolver: zodResolver(memoryFormSchema),
    defaultValues: {
      content: initialData?.content || '',
      type: initialData?.type || 'discussion',
      repository: initialData?.repository || '',
      sessionId: initialData?.sessionId || '',
      tags: initialData?.tags || [],
      isPublic: initialData?.isPublic || false,
      priority: initialData?.priority || 'medium'
    }
  })

  const addTag = () => {
    if (newTag.trim() && !tags.includes(newTag.trim())) {
      const updatedTags = [...tags, newTag.trim()]
      setTags(updatedTags)
      setValue('tags', updatedTags)
      setNewTag('')
    }
  }

  const removeTag = (tagToRemove: string) => {
    const updatedTags = tags.filter(tag => tag !== tagToRemove)
    setTags(updatedTags)
    setValue('tags', updatedTags)
  }

  const handleProtectedSubmit = async (
    formData: FormData, 
    submitProtected: (url: string, data: FormData | Record<string, unknown>) => Promise<Response>
  ) => {
    setIsSubmitting(true)
    
    try {
      // Extract form data
      const data: MemoryFormData = {
        content: formData.get('content') as string,
        type: formData.get('type') as MemoryFormData['type'],
        repository: formData.get('repository') as string,
        sessionId: formData.get('sessionId') as string || undefined,
        tags,
        isPublic: formData.get('isPublic') === 'on',
        priority: formData.get('priority') as MemoryFormData['priority']
      }

      // Submit via protected request
      const response = await submitProtected('/api/memories', data)
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to save memory')
      }

      const result = await response.json()
      
      if (onSuccess) {
        onSuccess(data)
      }
      
      logger.info('Memory saved successfully:', result)
    } catch (error) {
      logger.error('Failed to save memory:', error)
      throw error // Re-throw to let ProtectedForm handle the error display
    } finally {
      setIsSubmitting(false)
    }
  }

  const validateForm = (formData: FormData): string | null => {
    const content = formData.get('content') as string
    const repository = formData.get('repository') as string

    if (!content || content.trim().length < 10) {
      return 'Content must be at least 10 characters long'
    }

    if (!repository || repository.trim().length === 0) {
      return 'Repository is required'
    }

    return null
  }

  const watchedType = watch('type')
  const selectedTypeInfo = MEMORY_TYPES.find(t => t.value === watchedType)

  return (
    <div className={cn('space-y-6', className)}>
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold">
          {initialData ? 'Edit Memory' : 'Create Memory'}
        </h2>
        {onCancel && (
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        )}
      </div>

      <ProtectedForm
        onSubmit={handleProtectedSubmit}
        disabled={isSubmitting}
        validateBeforeSubmit={validateForm}
        className="space-y-6"
      >
        {/* Content */}
        <div className="space-y-2">
          <Label htmlFor="content">Content *</Label>
          <Textarea
            id="content"
            {...register('content')}
            placeholder="Describe your memory, solution, or discussion..."
            rows={6}
            className={cn(errors.content && 'border-destructive')}
          />
          {errors.content && (
            <p className="text-sm text-destructive flex items-center gap-1">
              <AlertCircle className="h-3 w-3" />
              {errors.content.message}
            </p>
          )}
        </div>

        {/* Type and Repository */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="type">Memory Type *</Label>
            <Select 
              defaultValue={watchedType}
              onValueChange={(value) => setValue('type', value as MemoryFormData['type'])}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select memory type" />
              </SelectTrigger>
              <SelectContent>
                {MEMORY_TYPES.map(type => (
                  <SelectItem key={type.value} value={type.value}>
                    <span className="flex items-center gap-2">
                      <span>{type.icon}</span>
                      <span>{type.label}</span>
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <input type="hidden" {...register('type')} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="repository">Repository *</Label>
            <Input
              id="repository"
              {...register('repository')}
              placeholder="e.g., github.com/user/repo"
              className={cn(errors.repository && 'border-destructive')}
            />
            {errors.repository && (
              <p className="text-sm text-destructive flex items-center gap-1">
                <AlertCircle className="h-3 w-3" />
                {errors.repository.message}
              </p>
            )}
          </div>
        </div>

        {/* Session ID and Priority */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="sessionId">Session ID</Label>
            <Input
              id="sessionId"
              {...register('sessionId')}
              placeholder="Optional session identifier"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="priority">Priority</Label>
            <Select 
              defaultValue={watch('priority')}
              onValueChange={(value) => setValue('priority', value as MemoryFormData['priority'])}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="low">Low</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="high">High</SelectItem>
              </SelectContent>
            </Select>
            <input type="hidden" {...register('priority')} />
          </div>
        </div>

        {/* Tags */}
        <div className="space-y-2">
          <Label>Tags</Label>
          <div className="flex gap-2">
            <Input
              value={newTag}
              onChange={(e) => setNewTag(e.target.value)}
              placeholder="Add a tag..."
              onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), addTag())}
            />
            <Button 
              type="button" 
              variant="outline" 
              size="sm"
              onClick={addTag}
              disabled={!newTag.trim()}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          
          {tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mt-2">
              {tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="flex items-center gap-1">
                  {tag}
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-4 w-4 p-0 hover:bg-transparent"
                    onClick={() => removeTag(tag)}
                  >
                    <X className="h-3 w-3" />
                  </Button>
                </Badge>
              ))}
            </div>
          )}
        </div>

        {/* Public checkbox */}
        <div className="flex items-center space-x-2">
          <Checkbox
            id="isPublic"
            {...register('isPublic')}
          />
          <Label htmlFor="isPublic" className="text-sm">
            Make this memory public (visible to other users)
          </Label>
        </div>

        {/* Submit Button */}
        <div className="flex gap-3">
          <Button
            type="submit"
            disabled={isSubmitting}
            className="flex-1"
          >
            {isSubmitting ? (
              <>
                <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                Saving...
              </>
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                {initialData ? 'Update Memory' : 'Create Memory'}
              </>
            )}
          </Button>
          
          {selectedTypeInfo && (
            <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded">
              <span className="text-lg">{selectedTypeInfo.icon}</span>
              <span className="text-sm text-muted-foreground">{selectedTypeInfo.label}</span>
            </div>
          )}
        </div>
      </ProtectedForm>

      {/* CSRF Debug Info (development only) */}
      <CSRFTokenDisplay />
    </div>
  )
}