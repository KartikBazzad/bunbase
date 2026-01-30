import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Select component props
 */
export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  /**
   * Error state
   * @default false
   */
  error?: boolean
  
  /**
   * Error message to display
   */
  errorMessage?: string
  
  /**
   * Select size
   * @default 'md'
   */
  size?: 'sm' | 'md' | 'lg'
}

/**
 * Select component for dropdown choices
 */
const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  ({
    error = false,
    errorMessage,
    size = 'md',
    className,
    disabled,
    children,
    ...props
  }, ref) => {
    const sizeClasses = {
      sm: 'px-3 py-1.5 text-sm',
      md: 'px-4 py-2.5 text-base',
      lg: 'px-5 py-3 text-lg',
    }

    const errorClasses = error
      ? 'border-error-500 focus:border-error-500 focus:ring-error-100'
      : 'border-gray-300 focus:border-primary-500 focus:ring-primary-100'

    return (
      <div className="relative w-full">
        <select
          ref={ref}
          className={cn(
            'input w-full appearance-none pr-10',
            sizeClasses[size],
            errorClasses,
            disabled && 'cursor-not-allowed',
            className
          )}
          disabled={disabled}
          aria-invalid={error}
          aria-describedby={errorMessage ? `${props.id || ''}-error` : undefined}
          {...props}
        >
          {children}
        </select>

        <div
          className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-gray-500"
          aria-hidden="true"
        >
          <svg
            className={`h-4 w-4 ${
              size === 'sm' ? 'h-3 w-3' : size === 'lg' ? 'h-5 w-5' : ''
            }`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </div>

        {error && errorMessage && (
          <p
            id={`${props.id || ''}-error`}
            className="mt-1 text-sm text-error-600"
            role="alert"
          >
            {errorMessage}
          </p>
        )}
      </div>
    )
  }
)

Select.displayName = 'Select'

export { Select }
