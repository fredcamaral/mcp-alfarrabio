/**
 * Memory Form Dialog Component
 * 
 * Dialog wrapper for the memory form, controlled by global UI state
 */

'use client'

import { useAppSelector, useAppDispatch } from '@/store/store'
import { selectShowMemoryForm, setShowMemoryForm } from '@/store/slices/uiSlice'
import { addMemories } from '@/store/slices/memoriesSlice'
import { addNotification } from '@/store/slices/uiSlice'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { MemoryForm } from './MemoryForm'
import { useStoreMemory } from '@/hooks/useMemoryAPI'
import { ConversationChunk } from '@/types/memory'
import { v4 as uuidv4 } from 'uuid'

// Import the type from MemoryForm
type MemoryFormData = {
  content: string
  type: 'problem' | 'solution' | 'architecture_decision' | 'session_summary' | 'code_change' | 'discussion' | 'analysis' | 'verification' | 'question'
  repository: string
  sessionId?: string
  tags: string[]
  isPublic: boolean
  priority: 'low' | 'medium' | 'high'
}

export function MemoryFormDialog() {
  const dispatch = useAppDispatch()
  const showMemoryForm = useAppSelector(selectShowMemoryForm)
  const { store } = useStoreMemory()

  const handleClose = () => {
    dispatch(setShowMemoryForm(false))
  }

  const handleSuccess = async (data: MemoryFormData) => {
    try {
      // Create a memory chunk from form data
      const chunk: Partial<ConversationChunk> = {
        id: uuidv4(),
        session_id: data.sessionId || uuidv4(),
        content: data.content,
        type: data.type,
        timestamp: new Date().toISOString(),
        metadata: {
          repository: data.repository,
          tags: data.tags,
          extended_metadata: {
            priority: data.priority,
            is_public: data.isPublic
          }
        }
      }

      // Store via API
      await store(chunk)
      
      // Add to Redux store
      dispatch(addMemories([chunk as ConversationChunk]))
      
      // Show success notification
      dispatch(addNotification({
        type: 'success',
        title: 'Memory Created',
        message: 'Your memory has been successfully stored',
        duration: 3000
      }))
      
      // Close dialog
      handleClose()
    } catch (error) {
      dispatch(addNotification({
        type: 'error',
        title: 'Failed to Create Memory',
        message: error instanceof Error ? error.message : 'An error occurred',
        duration: 5000
      }))
    }
  }

  return (
    <Dialog open={showMemoryForm} onOpenChange={setShowMemoryForm}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create New Memory</DialogTitle>
        </DialogHeader>
        <MemoryForm 
          onSuccess={handleSuccess}
          onCancel={handleClose}
          className="mt-4"
        />
      </DialogContent>
    </Dialog>
  )
}