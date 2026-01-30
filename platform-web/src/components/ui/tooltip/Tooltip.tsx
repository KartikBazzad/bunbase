import React, { useState, useRef, useEffect } from 'react'
import { cn } from '@/lib/cn'

/**
 * Tooltip position
 */
export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right'

/**
 * Tooltip component props
 */
export interface TooltipProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Tooltip content
   */
  content: React.ReactNode
  
  /**
   * Tooltip position
   * @default 'top'
   */
  position?: TooltipPosition
  
  /**
   * Show delay (ms)
   * @default 200
   */
  delay?: number
  
  /**
   * Hide delay (ms)
   * @default 0
   */
  hideDelay?: number
  
  /**
   * Disable hover delay
   * @default false
   */
  noDelay?: boolean
}

/**
 * Tooltip component for additional information
 */
const Tooltip = React.forwardRef<HTMLDivElement, TooltipProps>(
  ({
    content,
    position = 'top',
    delay = 200,
    hideDelay = 0,
    noDelay = false,
    className,
    children,
    ...props
  }, ref) => {
    const [visible, setVisible] = useState(false)
    const timeoutRef = useRef<NodeJS.Timeout>()

    useEffect(() => {
      return () => {
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current)
        }
      }
    }, [])

    const handleMouseEnter = () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
      
      if (noDelay) {
        setVisible(true)
      } else {
        timeoutRef.current = setTimeout(() => setVisible(true), delay)
      }
    }

    const handleMouseLeave = () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
      
      timeoutRef.current = setTimeout(() => setVisible(false), hideDelay)
    }

    const positionClasses = {
      top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
      bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
      left: 'right-full top-1/2 -translate-y-1/2 mr-2',
      right: 'left-full top-1/2 -translate-y-1/2 ml-2',
    }

    const arrowClasses = {
      top: 'bottom-0 left-1/2 -translate-x-1/2 translate-y-1/2 rotate-45',
      bottom: 'top-0 left-1/2 -translate-x-1/2 -translate-y-1/2 rotate-45',
      left: 'right-0 top-1/2 -translate-y-1/2 translate-x-1/2 rotate-45',
      right: 'left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 rotate-45',
    }

    return (
      <div
        ref={ref}
        className={cn('relative inline-block', className)}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onFocus={() => setVisible(true)}
        onBlur={() => setVisible(false)}
        {...props}
      >
        {children}
        
        {visible && (
          <div
            className={cn(
              'absolute z-50 w-max rounded-lg bg-gray-900 px-3 py-2 text-sm text-white',
              'shadow-lg',
              'animate-in fade-in zoom-in-95 duration-150',
              positionClasses[position]
            )}
            role="tooltip"
          >
            {content}
            <div
              className={cn(
                'absolute h-2 w-2 bg-gray-900',
                arrowClasses[position]
              )}
            />
          </div>
        )}
      </div>
    )
  }
)

Tooltip.displayName = 'Tooltip'

export { Tooltip }
