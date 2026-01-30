import React, { useMemo } from 'react'
import { cn } from '@/lib/cn'

/**
 * Password strength levels
 */
export type PasswordStrength = 'weak' | 'medium' | 'strong'

/**
 * PasswordStrength component props
 */
export interface PasswordStrengthProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Current password
   */
  password?: string
  
  /**
   * Minimum length for strong password
   * @default 8
   */
  minLength?: number
  
  /**
   * Show requirements list
   * @default true
   */
  showRequirements?: boolean
}

/**
 * PasswordStrength component for visual password strength indicator
 */
const PasswordStrength = React.forwardRef<HTMLDivElement, PasswordStrengthProps>(
  ({
    password = '',
    minLength = 8,
    showRequirements = true,
    className,
    ...props
  }, ref) => {
    const requirements = useMemo(() => [
      { id: 'length', label: `At least ${minLength} characters`, met: password.length >= minLength },
      { id: 'uppercase', label: 'Uppercase letter', met: /[A-Z]/.test(password) },
      { id: 'lowercase', label: 'Lowercase letter', met: /[a-z]/.test(password) },
      { id: 'number', label: 'Number', met: /[0-9]/.test(password) },
      { id: 'special', label: 'Special character', met: /[^A-Za-z0-9]/.test(password) },
    ], [password, minLength])

    const metCount = requirements.filter(r => r.met).length
    const totalCount = requirements.length

    const strength: PasswordStrength = useMemo(() => {
      if (metCount <= 2) return 'weak'
      if (metCount <= 4) return 'medium'
      return 'strong'
    }, [metCount])

    const strengthColors = {
      weak: 'bg-error-500',
      medium: 'bg-warning-500',
      strong: 'bg-success-500',
    }

    const strengthPercent = (metCount / totalCount) * 100

    return (
      <div ref={ref} className={cn('space-y-2', className)} {...props}>
        <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200">
          <div
            className={cn(
              'h-full transition-all duration-300 ease-out',
              strengthColors[strength]
            )}
            style={{ width: `${strengthPercent}%` }}
            role="progressbar"
            aria-valuenow={strengthPercent}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label={`Password strength: ${strength}`}
          />
        </div>

        {showRequirements && (
          <ul className="space-y-1 text-sm">
            {requirements.map(req => (
              <li
                key={req.id}
                className={cn(
                  'flex items-center gap-2',
                  req.met ? 'text-success-600' : 'text-gray-500'
                )}
              >
                <svg
                  className={`h-4 w-4 ${req.met ? 'text-success-500' : 'text-gray-400'}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  {req.met ? (
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  ) : (
                    <circle cx="12" cy="12" r="9" />
                  )}
                </svg>
                {req.label}
              </li>
            ))}
          </ul>
        )}
      </div>
    )
  }
)

PasswordStrength.displayName = 'PasswordStrength'

export { PasswordStrength }
