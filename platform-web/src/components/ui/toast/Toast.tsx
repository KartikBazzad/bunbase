import React, { useEffect, useState } from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Toast position
 */
export type ToastPosition = 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left' | 'top-center' | 'bottom-center'

/**
 * Toast variant
 */
export type ToastVariant = 'success' | 'error' | 'warning' | 'info'

/**
 * Toast item
 */
export interface Toast {
  id: string
  message: string
  variant?: ToastVariant
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
}

/**
 * Toast manager props
 */
export interface ToastManagerProps {
  /**
   * Toast container position
   * @default 'top-right'
   */
  position?: ToastPosition
  
  /**
   * Default toast duration (ms)
   * @default 5000
   */
  defaultDuration?: number
}

/**
 * Toast component context
 */
const ToastContext = React.createContext<{
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => string
  removeToast: (id: string) => void
} | null>(null)

/**
 * Toast provider hook
 */
export function useToast() {
  const context = React.useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within ToastProvider')
  }
  return context
}

/**
 * Toast provider component
 */
export function ToastProvider({
  children,
  position = 'top-right',
  defaultDuration = 5000,
}: ToastManagerProps & { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = (toast: Omit<Toast, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9)
    const newToast: Toast = {
      ...toast,
      id,
      duration: toast.duration ?? defaultDuration,
    }
    setToasts(prev => [...prev, newToast])
    return id
  }

  const removeToast = (id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }

  useEffect(() => {
    toasts.forEach(toast => {
      if (toast.duration) {
        const timeout = setTimeout(() => {
          removeToast(toast.id)
        }, toast.duration)
        return () => clearTimeout(timeout)
      }
    })
  }, [toasts])

  const positionClasses = {
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'top-center': 'top-4 left-1/2 -translate-x-1/2',
    'bottom-center': 'bottom-4 left-1/2 -translate-x-1/2',
  }

  const variantClasses = {
    success: 'border-success-500 bg-success-50 dark:bg-success-900/30',
    error: 'border-error-500 bg-error-50 dark:bg-error-900/30',
    warning: 'border-warning-500 bg-warning-50 dark:bg-warning-900/30',
    info: 'border-primary-500 bg-primary-50 dark:bg-primary-900/30',
  }

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast }}>
      {children}
      <div className={cn('fixed z-50 flex flex-col gap-2', positionClasses[position])}>
        {toasts.map(toast => (
          <div
            key={toast.id}
            className={cn(
              'flex w-full max-w-md items-start gap-3 rounded-lg border-2 p-4',
              'shadow-lg',
              'animate-in slide-in-from-right fade-in duration-200',
              variantClasses[toast.variant || 'info']
            )}
            role="alert"
          >
            <div className="flex-1">
              <p className="text-sm font-medium">
                {toast.message}
              </p>
              {toast.action && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={toast.action.onClick}
                  className="mt-2"
                >
                  {toast.action.label}
                </Button>
              )}
            </div>
            <button
              onClick={() => removeToast(toast.id)}
              className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              aria-label="Dismiss toast"
            >
              ✕
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

/**
 * Simple toast component (for quick usage)
 */
export function Toast({
  children,
}: { children: React.ReactNode }) {
  return <>{children}</>
}

Toast.displayName = 'Toast'

export { Toast, ToastProvider, useToast }
