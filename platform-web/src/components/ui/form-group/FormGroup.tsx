import React from 'react'
import { cn } from '@/lib/cn'
import { Label } from '../label'
import { Input } from '../input'

/**
 * FormGroup component props
 */
export interface FormGroupProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Label text
   */
  label?: string
  
  /**
   * HTML id for the form input
   */
  htmlFor?: string
  
  /**
   * Error message to display
   */
  error?: string
  
  /**
   * Hint text to display
   */
  hint?: string
  
  /**
   * Mark field as required
   * @default false
   */
  required?: boolean
  
  /**
   * Form input component
   */
  children: React.ReactNode
}

/**
 * FormGroup component for organizing form inputs with labels and messages
 */
const FormGroup = React.forwardRef<HTMLDivElement, FormGroupProps>(
  ({
    label,
    htmlFor,
    error,
    hint,
    required = false,
    className,
    children,
    ...props
  }, ref) => {
    const inputWithError = React.cloneElement(children as React.ReactElement, {
      error: !!error,
      errorMessage: error,
      id: htmlFor,
    })

    return (
      <div ref={ref} className={cn('space-y-1.5', className)} {...props}>
        {label && (
          <Label htmlFor={htmlFor} required={required}>
            {label}
          </Label>
        )}
        {inputWithError}
        {hint && !error && (
          <p className="text-sm text-gray-500">{hint}</p>
        )}
      </div>
    )
  }
)

FormGroup.displayName = 'FormGroup'

export { FormGroup }
