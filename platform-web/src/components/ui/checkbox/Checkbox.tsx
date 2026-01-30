import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Checkbox component props
 */
export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size' | 'type'> {
  /**
   * Indeterminate state
   * @default false
   */
  indeterminate?: boolean
  
  /**
   * Checkbox size
   * @default 'md'
   */
  size?: 'sm' | 'md' | 'lg'
  
  /**
   * Label text
   */
  label?: string
}

/**
 * Checkbox component for binary choices
 */
const Checkbox = React.forwardRef<HTMLInputElement, CheckboxProps>(
  ({
    indeterminate = false,
    size = 'md',
    label,
    className,
    checked,
    onChange,
    disabled,
    ...props
  }, ref) => {
    const inputRef = React.useRef<HTMLInputElement>(null)

    React.useEffect(() => {
      if (inputRef.current && inputRef.current.indeterminate !== indeterminate) {
        inputRef.current.indeterminate = indeterminate
      }
    }, [indeterminate])

    const sizeClasses = {
      sm: 'h-4 w-4',
      md: 'h-5 w-5',
      lg: 'h-6 w-6',
    }

    const labelSizeClasses = {
      sm: 'text-sm',
      md: 'text-base',
      lg: 'text-lg',
    }

    return (
      <label className={cn('inline-flex items-center gap-2', disabled && 'opacity-50 cursor-not-allowed')}>
        <input
          ref={(node) => {
            if (node) {
              inputRef.current = node
              if (typeof ref === 'function') ref(node)
              else if (ref) ref.current = node
            }
          }}
          type="checkbox"
          className={cn(
            'rounded border-gray-300 text-primary-600 focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            sizeClasses[size],
            disabled && 'cursor-not-allowed opacity-50',
            className
          )}
          checked={checked}
          onChange={onChange}
          disabled={disabled}
          {...props}
        />
        {label && (
          <span className={cn('text-gray-700 dark:text-gray-300', labelSizeClasses[size])}>
            {label}
          </span>
        )}
      </label>
    )
  }
)

Checkbox.displayName = 'Checkbox'

export { Checkbox }
