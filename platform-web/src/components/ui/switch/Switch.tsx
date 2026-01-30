import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Switch component props
 */
export interface SwitchProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'size'> {
  /**
   * Switch size
   * @default 'md'
   */
  size?: 'sm' | 'md' | 'lg'
  
  /**
   * Checked state (controlled)
   */
  checked?: boolean
  
  /**
   * Default checked state (uncontrolled)
   */
  defaultChecked?: boolean
  
  /**
   * Change handler
   */
  onChange?: (checked: boolean) => void
}

/**
 * Switch component for toggle controls
 */
const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(
  ({
    size = 'md',
    checked,
    defaultChecked,
    onChange,
    disabled,
    className,
    ...props
  }, ref) => {
    const [internalChecked, setInternalChecked] = React.useState(defaultChecked || false)
    const isChecked = checked !== undefined ? checked : internalChecked

    const sizeClasses = {
      sm: 'h-5 w-9',
      md: 'h-6 w-11',
      lg: 'h-7 w-13',
    }

    const thumbClasses = {
      sm: 'h-3 w-3',
      md: 'h-4 w-4',
      lg: 'h-5 w-5',
    }

    const thumbPositionClasses = isChecked
      ? 'translate-x-full'
      : 'translate-x-0'

    const toggleSwitch = () => {
      const newState = !isChecked
      if (checked === undefined) {
        setInternalChecked(newState)
      }
      onChange?.(newState)
    }

    return (
      <button
        ref={ref}
        type="button"
        onClick={toggleSwitch}
        disabled={disabled}
        className={cn(
          'relative inline-flex flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent',
          'transition-colors duration-200 ease-in-out',
          'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          isChecked ? 'bg-primary-600' : 'bg-gray-200',
          disabled && 'opacity-50 cursor-not-allowed',
          sizeClasses[size],
          className
        )}
        role="switch"
        aria-checked={isChecked}
        {...props}
      >
        <span
          className={cn(
            'pointer-events-none inline-block rounded-full bg-white shadow',
            'transition-transform duration-200 ease-in-out',
            thumbClasses[size],
            thumbPositionClasses
          )}
        />
      </button>
    )
  }
)

Switch.displayName = 'Switch'

export { Switch }
