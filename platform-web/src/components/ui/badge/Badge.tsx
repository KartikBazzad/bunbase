import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Badge color options
 */
export type BadgeColor = 'primary' | 'secondary' | 'success' | 'warning' | 'error'

/**
 * Badge variant options
 */
export type BadgeVariant = 'solid' | 'outline' | 'dot'

/**
 * Badge component props
 * 
 * @description
 * Extends native span attributes with color, variant, and dot options.
 */
export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  /**
   * Badge color
   * @default 'primary'
   */
  color?: BadgeColor
  
  /**
   * Badge visual variant
   * @default 'solid'
   */
  variant?: BadgeVariant
  
  /**
   * Show a dot indicator
   * @default false
   */
  dot?: boolean
  
  /**
   * Icon to display
   */
  icon?: React.ReactNode
}

/**
 * Badge component for status indicators and labels
 * 
 * @description
 * A versatile badge component supporting multiple colors, variants (solid, outline, dot),
 * and optional icon. Perfect for status indicators, tags, and labels.
 * 
 * @example
 * ```tsx
 * // Basic badge
 * <Badge>Default</Badge>
 * 
 * // Colored badges
 * <Badge color="success">Success</Badge>
 * <Badge color="error">Error</Badge>
 * <Badge color="warning">Warning</Badge>
 * 
 * // Outline variant
 * <Badge variant="outline">Outlined</Badge>
 * 
 * // Dot variant
 * <Badge variant="dot" color="success">
 *   Active
 * </Badge>
 * 
 * // With icon
 * <Badge icon={<Icon />}>With Icon</Badge>
 * 
 * // All combined
 * <Badge color="error" variant="dot" icon={<AlertIcon />}>
 *   Alert
 * </Badge>
 * ```
 * 
 * @accessibility
 * - Proper aria-label when used as status indicator
 * - Icon hidden from screen readers when decorative
 * - Can be used with appropriate ARIA roles
 * 
 * @since 1.0.0
 */
const Badge = React.forwardRef<HTMLSpanElement, BadgeProps>(
  ({
    color = 'primary',
    variant = 'solid',
    dot = false,
    icon,
    className,
    children,
    ...props
  }, ref) => {
    const colorClasses = {
      primary: 'badge-primary',
      secondary: 'badge-secondary',
      success: 'badge-success',
      warning: 'badge-warning',
      error: 'badge-error',
    }

    const variantClasses = {
      solid: '',
      outline: 'bg-transparent border-2 border-current',
      dot: 'gap-1.5',
    }

    const dotColorClasses = {
      primary: 'bg-primary-500',
      secondary: 'bg-gray-500',
      success: 'bg-success-500',
      warning: 'bg-warning-500',
      error: 'bg-error-500',
    }

    return (
      <span
        ref={ref}
        className={cn(
          'badge',
          colorClasses[color],
          variantClasses[variant],
          className
        )}
        {...props}
      >
        {dot && (
          <span
            className={cn(
              'h-1.5 w-1.5 rounded-full',
              dotColorClasses[color]
            )}
            aria-hidden="true"
          />
        )}
        {icon && (
          <span className="mr-1" aria-hidden="true">
            {icon}
          </span>
        )}
        <span>{children}</span>
      </span>
    )
  }
)

Badge.displayName = 'Badge'

export { Badge }
