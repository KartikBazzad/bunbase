import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Separator orientation options
 */
export type SeparatorOrientation = 'horizontal' | 'vertical'

/**
 * Separator component props
 * 
 * @description
 * Extends native div attributes with orientation and decorative options.
 */
export interface SeparatorProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Separator orientation
   * @default 'horizontal'
   */
  orientation?: SeparatorOrientation
  
  /**
   * Text label to display in the separator
   */
  label?: string
  
  /**
   * Mark as decorative (no semantic meaning)
   * @default true
   */
  decorative?: boolean
}

/**
 * Separator component for visual separation
 * 
 * @description
 * A horizontal or vertical separator line that can optionally display
 * a text label. Useful for dividing content sections.
 * 
 * @example
 * ```tsx
 * // Horizontal separator
 * <Separator />
 * 
 * // Vertical separator
 * <Separator orientation="vertical" className="h-8" />
 * 
 * // With label
 * <Separator label="OR" />
 * 
 * // With styled label
 * <Separator label="Section Title" />
 * ```
 * 
 * @accessibility
 * - Marked as decorative by default (role="separator", aria-orientation)
 * - Proper semantic structure
 * 
 * @since 1.0.0
 */
const Separator = React.forwardRef<HTMLDivElement, SeparatorProps>(
  ({
    orientation = 'horizontal',
    label,
    decorative = true,
    className,
    ...props
  }, ref) => {
    const orientationClasses = {
      horizontal: 'w-full border-t border-gray-200 dark:border-gray-800',
      vertical: 'h-full border-l border-gray-200 dark:border-gray-800',
    }

    const withLabelClasses = label
      ? orientation === 'horizontal'
        ? 'flex items-center border-0 gap-4'
        : 'flex flex-col items-center border-0 gap-4'
      : ''

    const labelWrapperClasses = label
      ? 'relative flex shrink-0 items-center'
      : ''

    return (
      <div
        ref={ref}
        role={decorative ? 'separator' : undefined}
        aria-orientation={orientation}
        className={cn(
          orientationClasses[orientation],
          withLabelClasses,
          className
        )}
        {...props}
      >
        {label && (
          <>
            <div className={cn(
              'border-gray-200 dark:border-gray-800',
              orientation === 'horizontal' ? 'flex-1 border-t' : 'flex-1 border-l'
            )} />
            <span className={cn(
              labelWrapperClasses,
              'px-2 text-sm font-medium text-gray-500 dark:text-gray-400'
            )}>
              {label}
            </span>
            <div className={cn(
              'border-gray-200 dark:border-gray-800',
              orientation === 'horizontal' ? 'flex-1 border-t' : 'flex-1 border-l'
            )} />
          </>
        )}
      </div>
    )
  }
)

Separator.displayName = 'Separator'

export { Separator }
