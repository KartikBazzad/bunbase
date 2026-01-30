import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Card component props
 * 
 * @description
 * Extends native div attributes with variant option.
 */
export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Card visual variant
   * @default 'elevated'
   */
  variant?: 'elevated' | 'flat' | 'outlined'
}

/**
 * Card component for grouping related content
 * 
 * @description
 * A versatile card component supporting elevated, flat, and outlined variants.
 * Can be used with CardHeader, CardBody, and CardFooter for complete layout.
 * 
 * @example
 * ```tsx
 * // Basic card
 * <Card>
 *   <CardBody>Content here</CardBody>
 * </Card>
 * 
 * // Complete card with header and footer
 * <Card variant="elevated">
 *   <CardHeader>
 *     <h2>Card Title</h2>
 *   </CardHeader>
 *   <CardBody>Card content goes here</CardBody>
 *   <CardFooter>
 *     <Button>Action</Button>
 *   </CardFooter>
 * </Card>
 * 
 * // Interactive card
 * <Card variant="elevated" className="card-hover cursor-pointer">
 *   <CardBody>Click me</CardBody>
 * </Card>
 * ```
 * 
 * @accessibility
 * - Proper semantic structure when using with heading elements
 * - Can be made interactive with proper role and keyboard handling
 * 
 * @since 1.0.0
 */
const Card = React.forwardRef<HTMLDivElement, CardProps>(
  ({ variant = 'elevated', className, children, ...props }, ref) => {
    const variantClasses = {
      elevated: 'shadow-soft',
      flat: 'border-none shadow-none',
      outlined: 'border-2 border-gray-200 shadow-none',
    }

    return (
      <div
        ref={ref}
        className={cn(
          'card',
          variantClasses[variant],
          className
        )}
        {...props}
      >
        {children}
      </div>
    )
  }
)

Card.displayName = 'Card'

/**
 * CardHeader component props
 */
export interface CardHeaderProps extends React.HTMLAttributes<HTMLDivElement> {}

/**
 * Card header component
 * 
 * @description
 * Contains the card's title and optional subtitle or actions.
 * Typically includes heading elements (h1-h6).
 * 
 * @example
 * ```tsx
 * <CardHeader>
 *   <h2>Card Title</h2>
 *   <p className="text-sm text-gray-600">Subtitle</p>
 * </CardHeader>
 * ```
 */
const CardHeader = React.forwardRef<HTMLDivElement, CardHeaderProps>(
  ({ className, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn('card-header', className)}
        {...props}
      />
    )
  }
)

CardHeader.displayName = 'CardHeader'

/**
 * CardBody component props
 */
export interface CardBodyProps extends React.HTMLAttributes<HTMLDivElement> {}

/**
 * Card body component
 * 
 * @description
 * Contains the main content of the card. This is where the primary
 * information or actions are displayed.
 * 
 * @example
 * ```tsx
 * <CardBody>
 *   <p>This is the main content of the card.</p>
 *   <img src="image.jpg" alt="Card image" />
 * </CardBody>
 * ```
 */
const CardBody = React.forwardRef<HTMLDivElement, CardBodyProps>(
  ({ className, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn('card-body', className)}
        {...props}
      />
    )
  }
)

CardBody.displayName = 'CardBody'

/**
 * CardFooter component props
 */
export interface CardFooterProps extends React.HTMLAttributes<HTMLDivElement> {}

/**
 * Card footer component
 * 
 * @description
 * Contains actions, metadata, or supplementary information.
 * Often used for buttons, links, or status indicators.
 * 
 * @example
 * ```tsx
 * <CardFooter>
 *   <Button variant="secondary">Cancel</Button>
 *   <Button>Submit</Button>
 * </CardFooter>
 * ```
 */
const CardFooter = React.forwardRef<HTMLDivElement, CardFooterProps>(
  ({ className, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn('card-footer', className)}
        {...props}
      />
    )
  }
)

CardFooter.displayName = 'CardFooter'

export { Card, CardHeader, CardBody, CardFooter }
