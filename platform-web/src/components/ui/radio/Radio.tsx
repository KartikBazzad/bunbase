import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Radio component props
 */
export interface RadioProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size' | 'type'> {
  /**
   * Radio size
   * @default 'md'
   */
  size?: 'sm' | 'md' | 'lg'
  
  /**
   * Label text
   */
  label?: string
}

/**
 * Radio component for single choice selection
 */
const Radio = React.forwardRef<HTMLInputElement, RadioProps>(
  ({
    size = 'md',
    label,
    className,
    checked,
    onChange,
    disabled,
    ...props
  }, ref) => {
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
          ref={ref}
          type="radio"
          className={cn(
            'border-gray-300 text-primary-600 focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
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

Radio.displayName = 'Radio'

export { Radio }
