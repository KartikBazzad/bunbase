import React, { useState } from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Sidebar menu item
 */
export interface MenuItem {
  id: string
  label: string
  icon?: React.ReactNode
  href?: string
  onClick?: () => void
  active?: boolean
  disabled?: boolean
  children?: MenuItem[]
}

/**
 * Sidebar component props
 */
export interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Menu items to display
   */
  items: MenuItem[]
  
  /**
   * Collapsed state (controlled)
   */
  collapsed?: boolean
  
  /**
   * Default collapsed state (uncontrolled)
   */
  defaultCollapsed?: boolean
  
  /**
   * On collapse toggle
   */
  onCollapseToggle?: (collapsed: boolean) => void
  
  /**
   * Sidebar width (pixels)
   * @default 250
   */
  width?: number
  
  /**
   * Collapsed width (pixels)
   * @default 64
   */
  collapsedWidth?: number
}

/**
 * Sidebar component for app navigation
 */
const Sidebar = React.forwardRef<HTMLDivElement, SidebarProps>(
  ({
    items,
    collapsed: controlledCollapsed,
    defaultCollapsed = false,
    onCollapseToggle,
    width = 250,
    collapsedWidth = 64,
    className,
  }, ref) => {
    const [internalCollapsed, setInternalCollapsed] = useState(defaultCollapsed)
    const collapsed = controlledCollapsed !== undefined ? controlledCollapsed : internalCollapsed

    const toggleCollapse = () => {
      const newState = !collapsed
      if (controlledCollapsed === undefined) {
        setInternalCollapsed(newState)
      }
      onCollapseToggle?.(newState)
    }

    const renderMenuItem = (item: MenuItem, depth = 0) => (
      <div key={item.id} className={depth > 0 ? 'ml-4' : ''}>
        <button
          onClick={() => {
            if (item.disabled) return
            item.onClick?.()
            if (item.href && window.location) {
              window.location.href = item.href
            }
          }}
          className={cn(
            'w-full flex items-center gap-3 rounded-lg px-3 py-2 text-left',
            'transition-all duration-200',
            'hover:bg-gray-100 dark:hover:bg-gray-800',
            item.active && 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300',
            item.disabled && 'opacity-50 cursor-not-allowed',
            !collapsed && 'justify-start',
            collapsed && 'justify-center'
          )}
          disabled={item.disabled}
          aria-current={item.active ? 'page' : undefined}
        >
          {item.icon && !collapsed && (
            <span className="flex-shrink-0" aria-hidden="true">
              {item.icon}
            </span>
          )}
          {item.icon && collapsed && (
            <span className="flex-shrink-0" aria-hidden="true">
              {item.icon}
            </span>
          )}
          {!collapsed && (
            <span className="flex-1 truncate">{item.label}</span>
          )}
        </button>

        {item.children && item.children.length > 0 && !collapsed && (
          <div className="mt-1 space-y-1">
            {item.children.map(child => renderMenuItem(child, depth + 1))}
          </div>
        )}
      </div>
    )

    return (
      <div
        ref={ref}
        className={cn(
          'flex h-screen flex-col border-r border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900',
          'transition-all duration-300 ease-in-out',
          className
        )}
        style={{
          width: collapsed ? collapsedWidth : width,
        }}
      >
        <div className="flex-1 overflow-y-auto px-3 py-4 space-y-1">
          {items.map(item => renderMenuItem(item))}
        </div>

        <div className="border-t border-gray-200 p-3 dark:border-gray-800">
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleCollapse}
            className="w-full"
          >
            {collapsed ? '→' : '←'}
          </Button>
        </div>
      </div>
    )
  }
)

Sidebar.displayName = 'Sidebar'

export { Sidebar }
