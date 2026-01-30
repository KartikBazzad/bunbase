import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Skeleton variant options
 */
export type SkeletonVariant = 'text' | 'circular' | 'rectangular'

/**
 * Skeleton component props
 * 
 * @description
 * Extends native div attributes with variant and animation options.
 */
export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Skeleton variant/shape
   * @default 'rectangular'
   */
  variant?: SkeletonVariant
  
  /**
   * Width of the skeleton
   */
  width?: string | number
  
  /**
   * Height of the skeleton
   */
  height?: string | number
  
  /**
   * Enable shimmer animation
   * @default true
   */
  animated?: boolean
  
  /**
   * Number of lines (for text variant)
   */
  lines?: number
}

/**
 * Skeleton component for loading states
 * 
 * @description
 * A skeleton loading component with multiple variants (text, circular, rectangular)
 * and optional animation. Perfect for indicating content is loading.
 * 
 * @example
 * ```tsx
 * // Rectangular skeleton
 * <Skeleton variant="rectangular" width={200} height={20} />
 * 
 * // Circular skeleton
 * <Skeleton variant="circular" width={40} height={40} />
 * 
 * // Text skeleton (single line)
 * <Skeleton variant="text" width="100%" />
 * 
 * // Text skeleton (multiple lines)
 * <Skeleton variant="text" lines={3} />
 * 
 * // Custom dimensions
 * <Skeleton width={300} height={150} />
 * 
 * // Disabled animation
 * <Skeleton animated={false} />
 * ```
 * 
 * @accessibility
 * - Marked with aria-busy="true"
 * - Proper semantic structure
 * - Announces loading state to screen readers
 * 
 * @since 1.0.0
 */
const Skeleton = React.forwardRef<HTMLDivElement, SkeletonProps>(
  ({
    variant = 'rectangular',
    width,
    height,
    animated = true,
    lines = 1,
    className,
    style,
    ...props
  }, ref) => {
    const variantClasses = {
      text: 'h-4 w-full',
      circular: 'rounded-full',
      rectangular: 'rounded-md',
    }

    const animationClasses = animated
      ? 'animate-pulse bg-gray-200 dark:bg-gray-800'
      : 'bg-gray-200 dark:bg-gray-800'

    const inlineStyles = {
      width: typeof width === 'number' ? `${width}px` : width,
      height: typeof height === 'number' ? `${height}px` : height,
      ...style,
    }

    if (variant === 'text' && lines > 1) {
      return (
        <div className="space-y-2" ref={ref} {...props}>
          {Array.from({ length: lines }).map((_, i) => (
            <Skeleton
              key={i}
              variant="text"
              width={i === lines - 1 ? '60%' : '100%'}
              animated={animated}
              className={className}
            />
          ))}
        </div>
      )
    }

    return (
      <div
        ref={ref}
        className={cn(
          variantClasses[variant],
          animationClasses,
          className
        )}
        style={inlineStyles}
        aria-busy="true"
        role="status"
        {...props}
      />
    )
  }
)

Skeleton.displayName = 'Skeleton'

export { Skeleton }
