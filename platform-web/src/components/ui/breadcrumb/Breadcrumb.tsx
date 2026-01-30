import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Breadcrumb item
 */
export interface BreadcrumbItem {
  label: string
  href?: string
  onClick?: () => void
}

/**
 * Breadcrumb component props
 */
export interface BreadcrumbProps extends React.HTMLAttributes<HTMLElement> {
  /**
   * Breadcrumb items
   */
  items: BreadcrumbItem[]
  
  /**
   * Separator between items
   * @default '/'
   */
  separator?: string
}

/**
 * Breadcrumb component for navigation hierarchy
 */
const Breadcrumb = React.forwardRef<HTMLElement, BreadcrumbProps>(
  ({
    items,
    separator = '/',
    className,
  }, ref) => {
    return (
      <nav
        ref={ref}
        className={cn('flex items-center text-sm', className)}
        aria-label="Breadcrumb"
      >
        {items.map((item, index) => {
          const isLast = index === items.length - 1
          return (
            <React.Fragment key={index}>
              {item.href || item.onClick ? (
                <a
                  href={item.href}
                  onClick={item.onClick}
                  className={cn(
                    'text-gray-600 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-400',
                    'transition-colors',
                    !isLast && 'hover:underline'
                  )}
                >
                  {item.label}
                </a>
              ) : (
                <span className="text-gray-900 dark:text-gray-100">
                  {item.label}
                </span>
              )}
              {!isLast && (
                <span className="mx-2 text-gray-400" aria-hidden="true">
                  {separator}
                </span>
              )}
            </React.Fragment>
          )
        })}
      </nav>
    )
  }
)

Breadcrumb.displayName = 'Breadcrumb'

export { Breadcrumb }
