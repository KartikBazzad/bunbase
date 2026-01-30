import React, { useState, useEffect, useRef } from 'react'
import { cn } from '@/lib/cn'

/**
 * Textarea component props
 */
export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  /**
   * Enable auto-resize
   * @default false
   */
  autoResize?: boolean
  
  /**
   * Show character count
   * @default false
   */
  showCount?: boolean
  
  /**
   * Maximum character count
   */
  maxLength?: number
  
  /**
   * Minimum number of rows
   * @default 3
   */
  minRows?: number
  
  /**
   * Maximum number of rows
   */
  maxRows?: number
  
  /**
   * Error state
   * @default false
   */
  error?: boolean
  
  /**
   * Error message to display
   */
  errorMessage?: string
}

/**
 * Textarea component for multi-line text input
 */
const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({
    autoResize = false,
    showCount = false,
    maxLength,
    minRows = 3,
    maxRows,
    error = false,
    errorMessage,
    className,
    disabled,
    readOnly,
    value,
    defaultValue,
    ...props
  }, ref) => {
    const internalRef = useRef<HTMLTextAreaElement>(null)
    const textareaRef = (ref || internalRef) as React.RefObject<HTMLTextAreaElement>
    const [charCount, setCharCount] = useState(0)

    useEffect(() => {
      const current = textareaRef.current
      if (!current) return

      const updateHeight = () => {
        current.style.height = 'auto'
        const scrollHeight = current.scrollHeight
        
        if (maxRows) {
          const lineHeight = parseInt(getComputedStyle(current).lineHeight)
          const maxHeight = lineHeight * maxRows
          current.style.height = `${Math.min(scrollHeight, maxHeight)}px`
        } else {
          current.style.height = `${scrollHeight}px`
        }
      }

      const updateCount = () => {
        setCharCount(current.value.length)
      }

      if (autoResize) {
        current.style.resize = 'none'
        updateHeight()
        current.addEventListener('input', updateHeight)
      }

      if (showCount) {
        updateCount()
        current.addEventListener('input', updateCount)
      }

      return () => {
        current.removeEventListener('input', updateHeight)
        current.removeEventListener('input', updateCount)
      }
    }, [autoResize, showCount, maxRows, textareaRef])

    const errorClasses = error
      ? 'border-error-500 focus:border-error-500 focus:ring-error-100'
      : 'border-gray-300 focus:border-primary-500 focus:ring-primary-100'

    return (
      <div className="relative w-full">
        <textarea
          ref={textareaRef}
          rows={minRows}
          className={cn(
            'input w-full',
            errorClasses,
            disabled && 'cursor-not-allowed',
            readOnly && 'bg-gray-50 cursor-default',
            autoResize && '!resize-none',
            className
          )}
          disabled={disabled}
          readOnly={readOnly}
          maxLength={maxLength}
          value={value}
          defaultValue={defaultValue}
          aria-invalid={error}
          aria-describedby={errorMessage ? `${props.id || ''}-error` : undefined}
          {...props}
        />

        {(showCount || errorMessage) && (
          <div className="mt-1 flex justify-between items-center">
            {error && errorMessage && (
              <p
                id={`${props.id || ''}-error`}
                className="text-sm text-error-600"
                role="alert"
              >
                {errorMessage}
              </p>
            )}
            {showCount && (
              <span className="text-sm text-gray-500">
                {charCount}
                {maxLength && ` / ${maxLength}`}
              </span>
            )}
          </div>
        )}
      </div>
    )
  }
)

Textarea.displayName = 'Textarea'

export { Textarea }
