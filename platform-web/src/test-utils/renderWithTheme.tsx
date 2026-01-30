import { ReactElement } from 'react'
import { render, RenderOptions } from '@testing-library/react'

/**
 * Custom render function with theme provider
 * 
 * @description
 * Wraps components in any necessary providers (theme, etc.)
 * for consistent testing across all components.
 * 
 * @param ui - React element to render
 * @param options - Additional render options
 * @returns Render result with additional queries
 * 
 * @since 1.0.0
 */
export function renderWithTheme(ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>) {
  // In the future, wrap with theme provider here
  // For now, just render as-is since we're using CSS variables
  return render(ui, options)
}

// Re-export everything from testing-library
export * from '@testing-library/react'
export { default as userEvent } from '@testing-library/user-event'
