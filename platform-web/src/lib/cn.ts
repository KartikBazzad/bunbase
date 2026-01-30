import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * Utility function to merge Tailwind CSS classes
 * 
 * @description
 * Combines clsx for conditional class names and tailwind-merge
 * to resolve Tailwind class conflicts intelligently.
 * 
 * @example
 * ```tsx
 * cn('px-4', 'py-2', isActive && 'bg-blue-500', className)
 * ```
 * 
 * @param inputs - Class values to merge
 * @returns Merged class string
 * 
 * @since 1.0.0
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
