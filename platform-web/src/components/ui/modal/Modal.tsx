import React, { useEffect, useRef } from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Modal size
 */
export type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | 'full'

/**
 * Modal component props
 */
export interface ModalProps {
  /**
   * Modal open state
   */
  open: boolean
  
  /**
   * On close handler
   */
  onClose: () => void
  
  /**
   * Modal size
   * @default 'md'
   */
  size?: ModalSize
  
  /**
   * Show close button
   * @default true
   */
  showCloseButton?: boolean
  
  /**
   * Close on backdrop click
   * @default true
   */
  closeOnBackdropClick?: boolean
  
  /**
   * Close on escape key
   * @default true
   */
  closeOnEscape?: boolean
  
  /**
   * Modal title
   */
  title?: string
  
  /**
   * Modal footer content
   */
  footer?: React.ReactNode
  
  /**
   * Prevent body scroll when open
   * @default true
   */
  preventScroll?: boolean
}

/**
 * Modal component for dialogs
 */
const Modal = ({
  open,
  onClose,
  size = 'md',
  showCloseButton = true,
  closeOnBackdropClick = true,
  closeOnEscape = true,
  title,
  footer,
  children,
  preventScroll = true,
}: ModalProps) => {
  const modalRef = useRef<HTMLDivElement>(null)
  const previousActiveElement = useRef<HTMLElement | null>(null)

  const sizeClasses = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    full: 'max-w-full mx-4',
  }

  useEffect(() => {
    if (open) {
      previousActiveElement.current = document.activeElement as HTMLElement
      modalRef.current?.focus()
      
      if (preventScroll) {
        document.body.style.overflow = 'hidden'
      }
    }

    return () => {
      if (previousActiveElement.current) {
        previousActiveElement.current.focus()
      }
      if (preventScroll) {
        document.body.style.overflow = ''
      }
    }
  }, [open, preventScroll])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (closeOnEscape && e.key === 'Escape' && open) {
        onClose()
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [closeOnEscape, open, onClose])

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (closeOnBackdropClick && e.target === e.currentTarget) {
      onClose()
    }
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      onClick={handleBackdropClick}
    >
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm"
        aria-hidden="true"
      />
      <div
        ref={modalRef}
        className={cn(
          'relative w-full rounded-xl bg-white shadow-xl dark:bg-gray-900',
          'max-h-[90vh] overflow-hidden',
          sizeClasses[size],
          'animate-in fade-in zoom-in-95 duration-200'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? 'modal-title' : undefined}
      >
        {title && (
          <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-800">
            <h2 id="modal-title" className="text-lg font-semibold">
              {title}
            </h2>
            {showCloseButton && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onClose}
                aria-label="Close modal"
              >
                ✕
              </Button>
            )}
          </div>
        )}

        <div className="overflow-y-auto px-6 py-4">{children}</div>

        {footer && (
          <div className="flex items-center justify-end gap-3 border-t border-gray-200 px-6 py-4 dark:border-gray-800">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}

Modal.displayName = 'Modal'

export { Modal }
