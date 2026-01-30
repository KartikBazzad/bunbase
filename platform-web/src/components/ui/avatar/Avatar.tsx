import React, { useState } from 'react'
import { cn } from '@/lib/cn'

/**
 * Avatar size options
 */
export type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

/**
 * Avatar component props
 * 
 * @description
 * Extends native img attributes with size, alt, and fallback options.
 */
export interface AvatarProps extends Omit<React.ImgHTMLAttributes<HTMLImageElement>, 'size'> {
  /**
   * Avatar size
   * @default 'md'
   */
  size?: AvatarSize
  
  /**
   * Fallback text to show when image fails or no src
   */
  fallback?: string
  
  /**
   * Show border around avatar
   * @default false
   */
  bordered?: boolean
  
  /**
   * Custom background color for fallback
   */
  fallbackBgColor?: string
  
  /**
   * Custom text color for fallback
   */
  fallbackTextColor?: string
}

/**
 * Avatar component for user profile images
 * 
 * @description
 * A versatile avatar component supporting multiple sizes, bordered variant,
 * and automatic fallback to initials when image is not provided or fails to load.
 * 
 * @example
 * ```tsx
 * // Basic avatar with image
 * <Avatar
 *   src="/user.jpg"
 *   alt="John Doe"
 * />
 * 
 * // With size
 * <Avatar
 *   src="/user.jpg"
 *   size="lg"
 *   alt="Large Avatar"
 * />
 * 
 * // With fallback initials
 * <Avatar
 *   fallback="JD"
 *   alt="John Doe"
 * />
 * 
 * // Bordered avatar
 * <Avatar
 *   src="/user.jpg"
 *   bordered
 *   alt="Bordered"
 * />
 * 
 * // Custom colors for fallback
 * <Avatar
 *   fallback="AB"
 *   fallbackBgColor="bg-blue-500"
 *   fallbackTextColor="text-white"
 *   alt="Custom"
 * />
 * ```
 * 
 * @accessibility
 * - Proper alt text for screen readers
 * - Fallback initials also announced
 * - Keyboard accessible when used as button
 * - Proper focus management
 * 
 * @since 1.0.0
 */
const Avatar = React.forwardRef<HTMLImageElement, AvatarProps>(
  ({
    size = 'md',
    fallback,
    bordered = false,
    fallbackBgColor,
    fallbackTextColor,
    className,
    src,
    alt = '',
    onClick,
    ...props
  }, ref) => {
    const [imageError, setImageError] = useState(!src)
    const [imageLoaded, setImageLoaded] = useState(!!src)

    const sizeClasses = {
      xs: 'h-6 w-6 text-xs',
      sm: 'h-8 w-8 text-sm',
      md: 'h-10 w-10 text-base',
      lg: 'h-12 w-12 text-lg',
      xl: 'h-16 w-16 text-xl',
    }

    const borderedClasses = bordered
      ? 'ring-2 ring-white ring-offset-2 ring-offset-gray-100 dark:ring-offset-gray-900'
      : ''

    const handleImageError = () => {
      setImageError(true)
      setImageLoaded(false)
    }

  const handleImageLoad = () => {
    setImageError(false)
    setImageLoaded(true)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (onClick && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault()
      onClick(e)
    }
  }

  // Generate initials from fallback text
    const trimmedFallback = fallback?.trim() || ''
    const initials = trimmedFallback
      ? trimmedFallback.length <= 2
        ? trimmedFallback.toUpperCase()
        : trimmedFallback
            .split(' ')
            .map((word) => word[0])
            .join('')
            .toUpperCase()
            .slice(0, 2)
      : ''

    // Generate color from initials if no custom color provided
    const stringToColor = (str: string) => {
      let hash = 0
      for (let i = 0; i < str.length; i++) {
        hash = str.charCodeAt(i) + ((hash << 5) - hash)
      }
      const colors = [
        'bg-red-500',
        'bg-orange-500',
        'bg-amber-500',
        'bg-yellow-500',
        'bg-lime-500',
        'bg-green-500',
        'bg-emerald-500',
        'bg-teal-500',
        'bg-cyan-500',
        'bg-sky-500',
        'bg-blue-500',
        'bg-indigo-500',
        'bg-violet-500',
        'bg-purple-500',
        'bg-fuchsia-500',
        'bg-pink-500',
        'bg-rose-500',
      ]
      return colors[Math.abs(hash) % colors.length]
    }

    const autoBgColor = fallbackBgColor || stringToColor(initials)
    const autoTextColor = fallbackTextColor || 'text-white'

    return (
      <div
        className={cn(
          'relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full',
          sizeClasses[size],
          borderedClasses,
          onClick && 'cursor-pointer',
          className
        )}
        onClick={onClick}
        onKeyDown={handleKeyDown}
        role={onClick ? 'button' : undefined}
        tabIndex={onClick ? 0 : undefined}
        aria-label={onClick ? alt : undefined}
      >
        {src && !imageError && imageLoaded ? (
          <img
            ref={ref}
            src={src}
            alt={alt}
            className="h-full w-full object-cover"
            onError={handleImageError}
            onLoad={handleImageLoad}
            {...props}
          />
        ) : (
          <span
            className={cn(
              'font-medium',
              autoTextColor,
              fallbackBgColor || autoBgColor
            )}
            aria-label={fallback || alt}
          >
            {initials}
          </span>
        )}
      </div>
    )
  }
)

Avatar.displayName = 'Avatar'

export { Avatar }
