import React from 'react'
import { cn } from '@/lib/cn'

/**
 * Progress size
 */
export type ProgressSize = 'sm' | 'md' | 'lg'

/**
 * Progress component props
 */
export interface ProgressProps extends Omit<React.HTMLAttributes<HTMLDivElement>, 'value'> {
  /**
   * Progress value (0-100)
   * @default 0
   */
  value?: number
  
  /**
   * Progress size
   * @default 'md'
   */
  size?: ProgressSize
  
  /**
   * Indeterminate state
   * @default false
   */
  indeterminate?: boolean
  
  /**
   * Striped animation
   * @default false
   */
  striped?: boolean
  
  /**
   * Show percentage label
   * @default false
   */
  showLabel?: boolean
  
  /**
   * Label formatter
   */
  labelFormatter?: (value: number) => string
}

/**
 * Progress component for showing completion status
 */
const Progress = React.forwardRef<HTMLDivElement, ProgressProps>(
  ({
    value = 0,
    size = 'md',
    indeterminate = false,
    striped = false,
    showLabel = false,
    labelFormatter = (v) => `${Math.round(v)}%`,
    className,
    ...props
  }, ref) => {
    const sizeClasses = {
      sm: 'h-2',
      md: 'h-4',
      lg: 'h-6',
    }

    const progressValue = indeterminate ? undefined : Math.min(100, Math.max(0, value))

    return (
      <div ref={ref} className={cn('space-y-2', className)} {...props}>
        <div
          className="w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-800"
          role="progressbar"
          aria-valuenow={progressValue}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className={cn(
              'h-full rounded-full bg-primary-600 transition-all duration-300',
              striped && 'animate-pulse',
              indeterminate && 'animate-[progress_1s_linear_infinite]'
            )}
            style={{
              width: indeterminate ? '100%' : `${progressValue}%`,
              transformOrigin: indeterminate ? 'left' : undefined,
              animation: indeterminate ? 'progress 1s linear infinite' : undefined,
            }}
          />
        </div>

        {showLabel && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600 dark:text-gray-400">
              Progress
            </span>
            <span className="font-medium text-gray-900 dark:text-gray-100">
              {indeterminate ? 'Loading...' : labelFormatter(value)}
            </span>
          </div>
        )}
      </div>
    )
  }
)

Progress.displayName = 'Progress'

export { Progress }
