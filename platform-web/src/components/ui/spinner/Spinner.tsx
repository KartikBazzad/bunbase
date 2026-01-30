import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Spinner size
 */
export type SpinnerSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

/**
 * Spinner variant
 */
export type SpinnerVariant = 'default' | 'primary' | 'success' | 'error'

/**
 * Spinner component props
 */
export interface SpinnerProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Spinner size
   * @default 'md'
   */
  size?: SpinnerSize
  
  /**
   * Spinner color variant
   * @default 'default'
   */
  variant?: SpinnerVariant
  
  /**
   * Overlay mode (shows backdrop)
   * @default false
   */
  overlay?: boolean
  
  /**
   * Label text
   */
  label?: string
}

/**
 * Spinner component for loading states
 */
const Spinner = React.forwardRef<HTMLDivElement, SpinnerProps>(
  ({
    size = 'md',
    variant = 'default',
    overlay = false,
    label,
    className,
    ...props
  }, ref) => {
    const sizeClasses = {
      xs: 'h-3 w-3 border-2',
      sm: 'h-4 w-4 border-2',
      md: 'h-6 w-6 border-2',
      lg: 'h-8 w-8 border-3',
      xl: 'h-12 w-12 border-4',
    }

    const variantClasses = {
      default: 'border-gray-300 border-t-gray-600 dark:border-gray-700 dark:border-t-gray-400',
      primary: 'border-primary-300 border-t-primary-600',
      success: 'border-success-300 border-t-success-600',
      error: 'border-error-300 border-t-error-600',
    }

    const spinner = (
      <div
        ref={ref}
        className={cn(
          'animate-spin rounded-full',
          sizeClasses[size],
          variantClasses[variant],
          className
        )}
        role="status"
        aria-busy="true"
        aria-label={label || 'Loading'}
        {...props}
      />
    )

    if (overlay) {
      return (
        <div
          className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-black/50"
          role="presentation"
        >
          {spinner}
          {label && (
            <p className="mt-4 text-sm font-medium text-white">
              {label}
            </p>
          )}
        </div>
      )
    }

    return (
      <div className="flex flex-col items-center justify-center">
        {spinner}
        {label && (
          <p className="mt-2 text-sm font-medium text-gray-600 dark:text-gray-400">
            {label}
          </p>
        )}
      </div>
    )
  }
)

Spinner.displayName = 'Spinner'

export { Spinner }
