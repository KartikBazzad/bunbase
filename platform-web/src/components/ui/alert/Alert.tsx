import React from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Alert variant
 */
export type AlertVariant = 'success' | 'error' | 'warning' | 'info'

/**
 * Alert component props
 */
export interface AlertProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Alert variant
   * @default 'info'
   */
  variant?: AlertVariant
  
  /**
   * Alert title
   */
  title?: string
  
  /**
   * Dismissible
   * @default false
   */
  dismissible?: boolean
  
  /**
   * On dismiss
   */
  onDismiss?: () => void
  
  /**
   * Action button
   */
  action?: {
    label: string
    onClick: () => void
  }
}

/**
 * Alert component for notifications
 */
const Alert = React.forwardRef<HTMLDivElement, AlertProps>(
  ({
    variant = 'info',
    title,
    dismissible = false,
    onDismiss,
    action,
    className,
    children,
    ...props
  }, ref) => {
    const variantClasses = {
      success: 'border-success-500 bg-success-50 text-success-900 dark:bg-success-900/30 dark:text-success-300',
      error: 'border-error-500 bg-error-50 text-error-900 dark:bg-error-900/30 dark:text-error-300',
      warning: 'border-warning-500 bg-warning-50 text-warning-900 dark:bg-warning-900/30 dark:text-warning-300',
      info: 'border-primary-500 bg-primary-50 text-primary-900 dark:bg-primary-900/30 dark:text-primary-300',
    }

    const iconMap = {
      success: '✓',
      error: '✕',
      warning: '⚠',
      info: 'ℹ',
    }

    return (
      <div
        ref={ref}
        className={cn(
          'flex gap-3 rounded-lg border-2 p-4',
          variantClasses[variant],
          className
        )}
        role="alert"
        {...props}
      >
        <span className="flex-shrink-0 text-xl" aria-hidden="true">
          {iconMap[variant]}
        </span>

        <div className="flex-1">
          {title && (
            <h4 className="mb-1 font-semibold">{title}</h4>
          )}
          <div className="text-sm">{children}</div>
          
          {action && (
            <Button
              variant="ghost"
              size="sm"
              onClick={action.onClick}
              className="mt-2"
            >
              {action.label}
            </Button>
          )}
        </div>

        {dismissible && (
          <button
            onClick={onDismiss}
            className="flex-shrink-0 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            aria-label="Dismiss alert"
          >
            ✕
          </button>
        )}
      </div>
    )
  }
)

Alert.displayName = 'Alert'

export { Alert }
