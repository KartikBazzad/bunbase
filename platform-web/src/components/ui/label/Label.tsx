import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Label component props
 * 
 * @description
 * Extends native label attributes with required indicator options.
 */
export interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
  /**
   * Mark field as required (shows asterisk)
   * @default false
   */
  required?: boolean
  
  /**
   * ID of the form element this label is for
   */
  htmlFor?: string
}

/**
 * Label component for form inputs
 * 
 * @description
 * A label component that properly connects to form inputs with the
 * 'for' attribute. Supports required indicator with asterisk.
 * 
 * @example
 * ```tsx
 * // Basic label
 * <Label htmlFor="email">Email</Label>
 * <Input id="email" />
 * 
 * // Required label
 * <Label htmlFor="password" required>
 *   Password
 * </Label>
 * <Input id="password" type="password" />
 * 
 * // Disabled label
 * <Label htmlFor="disabled" disabled>
 *   Disabled Field
 * </Label>
 * <Input id="disabled" disabled />
 * ```
 * 
 * @accessibility
 * - Proper 'for' attribute to connect with form inputs
 * - Visual required indicator with asterisk
 * - Screen reader support for required fields
 * - Proper focus handling when clicked
 * 
 * @since 1.0.0
 */
const Label = React.forwardRef<HTMLLabelElement, LabelProps>(
  ({ required = false, className, children, htmlFor, ...props }, ref) => {
    return (
      <label
        ref={ref}
        htmlFor={htmlFor}
        className={cn(
          'block text-sm font-medium text-gray-700 dark:text-gray-300',
          'mb-1.5 transition-colors',
          disabled: 'opacity-50 cursor-not-allowed',
          className
        )}
        {...props}
      >
        {children}
        {required && (
          <span className="ml-1 text-error-500" aria-hidden="true">
            *
          </span>
        )}
      </label>
    )
  }
)

Label.displayName = 'Label'

export { Label }
