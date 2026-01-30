import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Button variant options
 */
export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'outline'

/**
 * Button size options
 */
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

/**
 * Button component props
 * 
 * @description
 * Extends native button attributes with variant, size, and loading state options.
 */
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /**
   * Button visual variant
   * @default 'primary'
   */
  variant?: ButtonVariant
  
  /**
   * Button size
   * @default 'md'
   */
  size?: ButtonSize
  
  /**
   * Show loading state with spinner
   * @default false
   */
  loading?: boolean
  
  /**
   * Icon to display before the button text
   */
  startIcon?: React.ReactNode
  
  /**
   * Icon to display after the button text
   */
  endIcon?: React.ReactNode
  
  /**
   * Button as a different element (Link, etc.)
   */
  asChild?: boolean
}

/**
 * Button component with multiple variants and sizes
 * 
 * @description
 * A versatile button component supporting primary, secondary, ghost, danger,
 * and outline variants with multiple sizes and loading states. Includes support
 * for icons and proper accessibility.
 * 
 * @example
 * ```tsx
 * // Primary button
 * <Button variant="primary" size="md">
 *   Click me
 * </Button>
 * 
 * // With loading state
 * <Button loading variant="primary">
 *   Loading...
 * </Button>
 * 
 * // With icons
 * <Button startIcon={<Icon />} endIcon={<Arrow />}>
 *   Continue
 * </Button>
 * 
 * // Disabled state
 * <Button disabled>
 *   Cannot click
 * </Button>
 * ```
 * 
 * @accessibility
 * - Supports keyboard navigation (Enter, Space)
 * - Proper focus indicators with ring
 * - ARIA attributes for screen readers
 * - Loading state sets aria-busy
 * - Disabled state properly handled
 * 
 * @since 1.0.0
 */
const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({
    variant = 'primary',
    size = 'md',
    loading = false,
    startIcon,
    endIcon,
    className,
    children,
    disabled,
    type = 'button',
    ...props
  }, ref) => {
    const isDisabled = disabled || loading

    const baseClasses = 'btn'
    const variantClasses = `btn-${variant}`
    const sizeClasses = `btn-${size}`

    return (
      <button
        ref={ref}
        type={type}
        className={cn(baseClasses, variantClasses, sizeClasses, className)}
        disabled={isDisabled}
        aria-busy={loading}
        {...props}
      >
        {loading && (
          <span className="spinner mr-2" aria-hidden="true" />
        )}
        {!loading && startIcon && (
          <span className="mr-2" aria-hidden="true">
            {startIcon}
          </span>
        )}
        <span>{children}</span>
        {!loading && endIcon && (
          <span className="ml-2" aria-hidden="true">
            {endIcon}
          </span>
        )}
      </button>
    )
  }
)

Button.displayName = 'Button'

export { Button }
