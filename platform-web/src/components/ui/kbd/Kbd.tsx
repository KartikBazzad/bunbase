import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Kbd component props
 */
export interface KbdProps extends React.HTMLAttributes<HTMLElement> {
  /**
   * Keyboard shortcut to display
   */
  shortcut: string | string[]
}

/**
 * Kbd component for keyboard shortcuts
 */
const Kbd = React.forwardRef<HTMLElement, KbdProps>(
  ({ shortcut, className, ...props }, ref) => {
    const shortcuts = Array.isArray(shortcut) ? shortcut : [shortcut]
    
    return (
      <div
        ref={ref}
        className={cn(
          'inline-flex items-center gap-1 rounded-md border border-gray-300 bg-gray-100 px-2 py-1',
          'text-xs font-mono font-medium text-gray-700',
          'shadow-sm',
          'dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300',
          className
        )}
        {...props}
      >
        {shortcuts.map((key, index) => (
          <React.Fragment key={index}>
            {index > 0 && <span className="mx-1">+</span>}
            <kbd className="border-none bg-transparent p-0 shadow-none">
              {key}
            </kbd>
          </React.Fragment>
        ))}
      </div>
    )
  }
)

Kbd.displayName = 'Kbd'

export { Kbd }
