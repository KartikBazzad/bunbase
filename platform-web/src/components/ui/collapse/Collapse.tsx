import React, { useState } from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Collapse component props
 */
export interface CollapseProps {
  /**
   * Collapse title
   */
  title: React.ReactNode
  
  /**
   * Open state (controlled)
   */
  open?: boolean
  
  /**
   * Default open state (uncontrolled)
   * @default false
   */
  defaultOpen?: boolean
  
  /**
   * On toggle
   */
  onToggle?: (open: boolean) => void
  
  /**
   * Disabled state
   * @default false
   */
  disabled?: boolean
  
  /**
   * Variant
   * @default 'default'
   */
  variant?: 'default' | 'bordered'
  
  /**
   * Size
   * @default 'md'
   */
  size?: 'sm' | 'md' | 'lg'
}

/**
 * Collapse component for accordion behavior
 */
const Collapse = ({
  title,
  open: controlledOpen,
  defaultOpen = false,
  onToggle,
  disabled = false,
  variant = 'default',
  size = 'md',
  children,
}: CollapseProps) => {
  const [internalOpen, setInternalOpen] = useState(defaultOpen)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen

  const handleToggle = () => {
    if (disabled) return
    const newState = !open
    if (controlledOpen === undefined) {
      setInternalOpen(newState)
    }
    onToggle?.(newState)
  }

  const variantClasses = {
    default: 'bg-white dark:bg-gray-900',
    bordered: 'border border-gray-200 dark:border-gray-800',
  }

  const sizeClasses = {
    sm: 'p-3',
    md: 'p-4',
    lg: 'p-5',
  }

  return (
    <div
      className={cn(
        'overflow-hidden rounded-lg',
        variantClasses[variant],
        'transition-all duration-200'
      )}
    >
      <Button
        variant="ghost"
        onClick={handleToggle}
        disabled={disabled}
        className={cn(
          'w-full justify-start',
          sizeClasses[size]
        )}
      >
        <div className="flex flex-1 items-center gap-3">
          <span className="text-lg transition-transform" aria-hidden="true">
            {open ? '−' : '+'}
          </span>
          <span className="font-medium">{title}</span>
        </div>
      </Button>

      <div
        className={cn(
          'overflow-hidden transition-all duration-200 ease-in-out',
          open ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
        )}
      >
        <div className={cn('p-4', sizeClasses[size])}>
          {children}
        </div>
      </div>
    </div>
  )
}

Collapse.displayName = 'Collapse'

export { Collapse }
