import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Tab item
 */
export interface TabItem {
  id: string
  label: string
  icon?: React.ReactNode
  disabled?: boolean
}

/**
 * Tabs variant
 */
export type TabsVariant = 'underline' | 'pill'

/**
 * Tabs component props
 */
export interface TabsProps extends Omit<React.HTMLAttributes<HTMLDivElement>, 'onChange'> {
  /**
   * Tab items
   */
  tabs: TabItem[]
  
  /**
   * Active tab ID (controlled)
   */
  activeTab?: string
  
  /**
   * Default active tab ID (uncontrolled)
   */
  defaultActiveTab?: string
  
  /**
   * On tab change
   */
  onTabChange?: (tabId: string) => void
  
  /**
   * Tabs variant
   * @default 'underline'
   */
  variant?: TabsVariant
  
  /**
   * Full width
   * @default false
   */
  fullWidth?: boolean
}

/**
 * Tabs component for content organization
 */
const Tabs = React.forwardRef<HTMLDivElement, TabsProps>(
  ({
    tabs,
    activeTab: controlledActiveTab,
    defaultActiveTab = tabs[0]?.id,
    onTabChange,
    variant = 'underline',
    fullWidth = false,
    className,
    children,
  }, ref) => {
    const [internalActiveTab, setInternalActiveTab] = React.useState(defaultActiveTab)
    const activeTab = controlledActiveTab !== undefined ? controlledActiveTab : internalActiveTab

    const handleTabClick = (tab: TabItem) => {
      if (tab.disabled) return
      
      if (controlledActiveTab === undefined) {
        setInternalActiveTab(tab.id)
      }
      onTabChange?.(tab.id)
    }

    const variantClasses = {
      underline: 'border-b border-gray-200 dark:border-gray-800',
      pill: 'bg-gray-100 p-1 rounded-lg dark:bg-gray-800',
    }

    const tabVariantClasses = {
      underline: (isActive: boolean) => cn(
        'relative pb-3 px-4 text-sm font-medium transition-colors',
        'after:content-[""] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:transition-all',
        isActive
          ? 'text-primary-600 after:bg-primary-600 dark:text-primary-400 dark:after:bg-primary-400'
          : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100 after:bg-transparent',
        tab.disabled && 'opacity-50 cursor-not-allowed'
      ),
      pill: (isActive: boolean) => cn(
        'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all',
        isActive
          ? 'bg-white text-primary-700 shadow-sm dark:bg-gray-900 dark:text-primary-300'
          : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100',
        tab.disabled && 'opacity-50 cursor-not-allowed'
      ),
    }

    return (
      <div ref={ref} className={cn('space-y-4', className)}>
        <div className={cn('flex', fullWidth && 'w-full', variantClasses[variant])}>
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => handleTabClick(tab)}
              className={cn(
                tabVariantClasses[variant](tab.id === activeTab),
                !fullWidth && !variant.includes('pill') && 'flex-shrink-0'
              )}
              role="tab"
              aria-selected={tab.id === activeTab}
              aria-controls={`panel-${tab.id}`}
              disabled={tab.disabled}
            >
              {tab.icon && (
                <span className="mr-2" aria-hidden="true">
                  {tab.icon}
                </span>
              )}
              {tab.label}
            </button>
          ))}
        </div>

        {children && (
          <div>
            {React.Children.map(children, (child, index) => {
              if (React.isValidElement(child)) {
                const tabId = tabs[index]?.id
                const isActive = tabId === activeTab
                return React.cloneElement(child as React.ReactElement, {
                  id: tabId,
                  active: isActive,
                })
              }
              return child
            })}
          </div>
        )}
      </div>
    )
  }
)

Tabs.displayName = 'Tabs'

/**
 * TabPanel component
 */
export interface TabPanelProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Panel ID (must match tab ID)
   */
  id: string
  
  /**
   * Active state (managed by Tabs)
   */
  active?: boolean
}

/**
 * TabPanel component for tab content
 */
const TabPanel = React.forwardRef<HTMLDivElement, TabPanelProps>(
  ({ id, active = false, className, children, ...props }, ref) => {
    if (!active) return null

    return (
      <div
        ref={ref}
        id={`panel-${id}`}
        role="tabpanel"
        aria-labelledby={id}
        className={className}
        {...props}
      >
        {children}
      </div>
    )
  }
)

TabPanel.displayName = 'TabPanel'

export { Tabs, TabPanel }
