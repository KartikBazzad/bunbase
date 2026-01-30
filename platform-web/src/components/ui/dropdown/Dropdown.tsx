import React, { useState, useRef, useEffect } from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Dropdown item
 */
export interface DropdownItem {
  id: string
  label: string
  icon?: React.ReactNode
  onClick?: () => void
  disabled?: boolean
  danger?: boolean
}

/**
 * Dropdown component props
 */
export interface DropdownProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Dropdown items
   */
  items: DropdownItem[]
  
  /**
   * Trigger button content
   */
  trigger: React.ReactNode
  
  /**
   * Dropdown align
   * @default 'left'
   */
  align?: 'left' | 'right'
  
  /**
   * Disabled state
   * @default false
   */
  disabled?: boolean
}

/**
 * Dropdown component for menus
 */
const Dropdown = React.forwardRef<HTMLDivElement, DropdownProps>(
  ({
    items,
    trigger,
    align = 'left',
    disabled = false,
    className,
  }, ref) => {
    const [open, setOpen] = useState(false)
    const dropdownRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
          setOpen(false)
        }
      }

      if (open) {
        document.addEventListener('mousedown', handleClickOutside)
      }

      return () => {
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }, [open])

    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
      }
    }

    return (
      <div
        ref={ref}
        className={cn('relative', className)}
        onKeyDown={handleKeyDown}
      >
        <Button
          variant="ghost"
          onClick={() => !disabled && setOpen(!open)}
          disabled={disabled}
          aria-expanded={open}
          aria-haspopup="true"
        >
          {trigger}
          <span className="ml-2" aria-hidden="true">▼</span>
        </Button>

        {open && (
          <div
            ref={dropdownRef}
            className={cn(
              'absolute z-50 mt-2 w-56 rounded-lg border border-gray-200 bg-white py-1 shadow-lg',
              'animate-in fade-in zoom-in-95 duration-150',
              align === 'left' ? 'left-0' : 'right-0',
              'dark:border-gray-700 dark:bg-gray-900'
            )}
            role="menu"
          >
            {items.map((item) => (
              <button
                key={item.id}
                onClick={() => {
                  if (item.disabled) return
                  item.onClick?.()
                  setOpen(false)
                }}
                className={cn(
                  'flex w-full items-center gap-3 px-3 py-2 text-left text-sm',
                  'transition-colors',
                  'hover:bg-gray-100 dark:hover:bg-gray-800',
                  item.disabled && 'opacity-50 cursor-not-allowed',
                  item.danger && 'text-error-600 hover:text-error-700 dark:text-error-400'
                )}
                role="menuitem"
                disabled={item.disabled}
              >
                {item.icon && (
                  <span className="flex-shrink-0" aria-hidden="true">
                    {item.icon}
                  </span>
                )}
                <span className="flex-1">{item.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    )
  }
)

Dropdown.displayName = 'Dropdown'

export { Dropdown }
