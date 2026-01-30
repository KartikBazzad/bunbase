import React, { useState } from 'react'
import { cn } from '@/lib/cn'

/**
 * Input type options
 */
export type InputType = 'text' | 'email' | 'password' | 'search' | 'number' | 'url' | 'tel'

/**
 * Input size options
 */
export type InputSize = 'sm' | 'md' | 'lg'

/**
 * Input component props
 * 
 * @description
 * Extends native input attributes with additional UI options.
 */
export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  /**
   * Input type
   * @default 'text'
   */
  type?: InputType
  
  /**
   * Input size
   * @default 'md'
   */
  size?: InputSize
  
  /**
   * Error state
   * @default false
   */
  error?: boolean
  
  /**
   * Error message to display
   */
  errorMessage?: string
  
  /**
   * Icon to display on the left
   */
  startIcon?: React.ReactNode
  
  /**
   * Icon to display on the right
   */
  endIcon?: React.ReactNode
  
  /**
   * Show/hide password toggle for password type
   * @default true
   */
  showPasswordToggle?: boolean
}

/**
 * Input component for form fields
 * 
 * @description
 * A versatile input component supporting multiple input types, sizes,
 * error states, icons, and password toggle. Fully accessible with
 * proper ARIA attributes and keyboard support.
 * 
 * @example
 * ```tsx
 * // Basic text input
 * <Input type="text" placeholder="Enter text" />
 * 
 * // With label
 * <Label htmlFor="email">Email</Label>
 * <Input id="email" type="email" placeholder="you@example.com" />
 * 
 * // With error state
 * <Input error errorMessage="Invalid email" />
 * 
 * // Password with toggle
 * <Input type="password" placeholder="Password" />
 * 
 * // With icons
 * <Input
 *   startIcon={<SearchIcon />}
 *   endIcon={<XIcon />}
 *   placeholder="Search..."
 * />
 * 
 * // Different sizes
 * <Input size="sm" />
 * <Input size="md" />
 * <Input size="lg" />
 * 
 * // Disabled state
 * <Input disabled placeholder="Disabled field" />
 * 
 * // Readonly state
 * <Input readOnly value="Cannot edit" />
 * ```
 * 
 * @accessibility
 * - Proper ARIA attributes for error states
 * - Visible focus indicators
 * - Password toggle with proper labeling
 * - Keyboard accessible
 * - Screen reader support for icons (aria-hidden)
 * 
 * @since 1.0.0
 */
const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({
    type = 'text',
    size = 'md',
    error = false,
    errorMessage,
    startIcon,
    endIcon,
    showPasswordToggle = true,
    className,
    disabled,
    readOnly,
    ...props
  }, ref) => {
    const [showPassword, setShowPassword] = useState(false)
    const isPasswordType = type === 'password'
    const effectiveType = isPasswordType && showPassword ? 'text' : type

    const sizeClasses = {
      sm: 'px-3 py-1.5 text-sm',
      md: 'px-4 py-2.5 text-base',
      lg: 'px-5 py-3 text-lg',
    }

    const errorClasses = error
      ? 'border-error-500 focus:border-error-500 focus:ring-error-100 input-error'
      : 'border-gray-300 focus:border-primary-500 focus:ring-primary-100'

    const togglePassword = () => {
      setShowPassword(!showPassword)
    }

    return (
      <div className="relative w-full">
        <div className="relative flex items-center">
          {startIcon && (
            <div
              className="absolute left-3 text-gray-400 pointer-events-none"
              aria-hidden="true"
            >
              {startIcon}
            </div>
          )}

          <input
            ref={ref}
            type={effectiveType}
            className={cn(
              'input',
              sizeClasses[size],
              errorClasses,
              startIcon && 'pl-10',
              (endIcon || (isPasswordType && showPasswordToggle)) && 'pr-10',
              disabled && 'cursor-not-allowed',
              readOnly && 'bg-gray-50 cursor-default',
              className
            )}
            disabled={disabled}
            readOnly={readOnly}
            aria-invalid={error}
            aria-describedby={errorMessage ? `${props.id || ''}-error` : undefined}
            {...props}
          />

          {isPasswordType && showPasswordToggle && !disabled && !readOnly && (
            <button
              type="button"
              onClick={togglePassword}
              className="absolute right-3 text-gray-400 hover:text-gray-600 focus:outline-none transition-colors"
              aria-label={showPassword ? 'Hide password' : 'Show password'}
              aria-pressed={showPassword}
            >
              {showPassword ? (
                <svg
                  className="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
                  />
                </svg>
              ) : (
                <svg
                  className="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                  />
                </svg>
              )}
            </button>
          )}

          {endIcon && !isPasswordType && (
            <div
              className="absolute right-3 text-gray-400 pointer-events-none"
              aria-hidden="true"
            >
              {endIcon}
            </div>
          )}
        </div>

        {error && errorMessage && (
          <p
            id={`${props.id || ''}-error`}
            className="mt-1 text-sm text-error-600"
            role="alert"
          >
            {errorMessage}
          </p>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'

export { Input }
