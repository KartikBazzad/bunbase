import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Navbar component props
 */
export interface NavbarProps extends React.HTMLAttributes<HTMLElement> {
  /**
   * Logo to display
   */
  logo?: React.ReactNode
  
  /**
   * Actions to display on the right
   */
  actions?: React.ReactNode
  
  /**
   * Fixed positioning
   * @default false
   */
  fixed?: boolean
  
  /**
   * Transparent background
   * @default false
   */
  transparent?: boolean
}

/**
 * Navbar component for app navigation
 */
const Navbar = React.forwardRef<HTMLElement, NavbarProps>(
  ({
    logo,
    actions,
    fixed = false,
    transparent = false,
    className,
    children,
    ...props
  }, ref) => {
    return (
      <nav
        ref={ref}
        className={cn(
          'border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900',
          'transition-all duration-200',
          fixed && 'fixed top-0 left-0 right-0 z-50',
          transparent && 'border-transparent bg-transparent',
          className
        )}
        {...props}
      >
        <div className="container-custom mx-auto">
          <div className="flex h-16 items-center justify-between">
            {logo && (
              <div className="flex-shrink-0">{logo}</div>
            )}
            {children && (
              <div className="flex-1 flex items-center justify-center">
                {children}
              </div>
            )}
            {actions && (
              <div className="flex items-center gap-2">{actions}</div>
            )}
          </div>
        </div>
      </nav>
    )
  }
)

Navbar.displayName = 'Navbar'

export { Navbar }
