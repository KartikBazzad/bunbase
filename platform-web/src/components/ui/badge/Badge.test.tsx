import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test-utils/renderWithTheme'
import { Badge } from './Badge'

describe('Badge', () => {
  describe('Rendering', () => {
    it('should render primary color by default', () => {
      render(<Badge>Primary</Badge>)
      const badge = screen.getByText('Primary')
      expect(badge).toBeInTheDocument()
      expect(badge).toHaveClass('badge-primary')
    })

    it('should render primary color explicitly', () => {
      render(<Badge color="primary">Primary</Badge>)
      const badge = screen.getByText('Primary')
      expect(badge).toHaveClass('badge-primary')
    })

    it('should render secondary color', () => {
      render(<Badge color="secondary">Secondary</Badge>)
      const badge = screen.getByText('Secondary')
      expect(badge).toHaveClass('badge-secondary')
    })

    it('should render success color', () => {
      render(<Badge color="success">Success</Badge>)
      const badge = screen.getByText('Success')
      expect(badge).toHaveClass('badge-success')
    })

    it('should render warning color', () => {
      render(<Badge color="warning">Warning</Badge>)
      const badge = screen.getByText('Warning')
      expect(badge).toHaveClass('badge-warning')
    })

    it('should render error color', () => {
      render(<Badge color="error">Error</Badge>)
      const badge = screen.getByText('Error')
      expect(badge).toHaveClass('badge-error')
    })

    it('should render solid variant by default', () => {
      render(<Badge>Solid</Badge>)
      const badge = screen.getByText('Solid')
      expect(badge.parentElement).not.toHaveClass('border-2')
    })

    it('should render solid variant explicitly', () => {
      render(<Badge variant="solid">Solid</Badge>)
      const badge = screen.getByText('Solid')
      expect(badge.parentElement).not.toHaveClass('border-2')
    })

    it('should render outline variant', () => {
      render(<Badge variant="outline">Outlined</Badge>)
      const badge = screen.getByText('Outlined')
      expect(badge.parentElement).toHaveClass('border-2', 'border-current')
    })

    it('should render outline variant with proper background', () => {
      render(<Badge variant="outline">Outlined</Badge>)
      const badge = screen.getByText('Outlined')
      expect(badge.parentElement).toHaveClass('bg-transparent')
    })

    it('should render dot variant', () => {
      render(<Badge variant="dot">Dot</Badge>)
      const badge = screen.getByText('Dot').parentElement
      expect(badge).toHaveClass('gap-1.5')
    })

    it('should render dot indicator', () => {
      render(<Badge variant="dot">With Dot</Badge>)
      const dot = screen.getByText('With Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toBeInTheDocument()
    })

    it('should render dot with primary color', () => {
      render(<Badge variant="dot" color="primary">Dot</Badge>)
      const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toHaveClass('bg-primary-500')
    })

    it('should render dot with secondary color', () => {
      render(<Badge variant="dot" color="secondary">Dot</Badge>)
      const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toHaveClass('bg-gray-500')
    })

    it('should render dot with success color', () => {
      render(<Badge variant="dot" color="success">Dot</Badge>)
      const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toHaveClass('bg-success-500')
    })

    it('should render dot with warning color', () => {
      render(<Badge variant="dot" color="warning">Dot</Badge>)
      const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toHaveClass('bg-warning-500')
    })

    it('should render dot with error color', () => {
      render(<Badge variant="dot" color="error">Dot</Badge>)
      const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toHaveClass('bg-error-500')
    })

    it('should render with icon', () => {
      const icon = <span data-testid="badge-icon">★</span>
      render(<Badge icon={icon}>With Icon</Badge>)
      expect(screen.getByTestId('badge-icon')).toBeInTheDocument()
    })

    it('should render dot and icon together', () => {
      const icon = <span data-testid="badge-icon">★</span>
      render(<Badge variant="dot" icon={icon}>Both</Badge>)
      expect(screen.getByTestId('badge-icon')).toBeInTheDocument()
      expect(screen.getByText('Both').parentElement?.querySelector('.rounded-full')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(<Badge className="custom-class">Custom</Badge>)
      const badge = screen.getByText('Custom').parentElement
      expect(badge).toHaveClass('custom-class')
    })

    it('should merge custom className with base classes', () => {
      render(
        <Badge color="success" variant="outline" className="mt-2">
          Combined
        </Badge>
      )
      const badge = screen.getByText('Combined').parentElement
      expect(badge).toHaveClass('badge', 'badge-success', 'border-2', 'mt-2')
    })
  })

  describe('Content', () => {
    it('should render short text', () => {
      render(<Badge>OK</Badge>)
      expect(screen.getByText('OK')).toBeInTheDocument()
    })

    it('should render long text', () => {
      const longText = 'This is a very long badge text'
      render(<Badge>{longText}</Badge>)
      expect(screen.getByText(longText)).toBeInTheDocument()
    })

    it('should handle empty text', () => {
      render(<Badge></Badge>)
      const badge = screen.getByText('').parentElement
      expect(badge).toBeInTheDocument()
    })

    it('should handle whitespace', () => {
      render(<Badge>   </Badge>)
      const badge = screen.getByText('   ').parentElement
      expect(badge).toBeInTheDocument()
    })

    it('should handle special characters', () => {
      render(<Badge>Special & < > " '</Badge>)
      expect(screen.getByText(/Special &/)).toBeInTheDocument()
    })

    it('should handle emoji', () => {
      render(<Badge>🎉 Emoji 🚀</Badge>)
      expect(screen.getByText('🎉 Emoji 🚀')).toBeInTheDocument()
    })

    it('should handle numbers', () => {
      render(<Badge>123</Badge>)
      expect(screen.getByText('123')).toBeInTheDocument()
    })

    it('should handle mixed content', () => {
      render(
        <Badge>
          <span>Bold</span> text
        </Badge>
      )
      const badge = screen.getByText('Bold text')
      expect(badge).toBeInTheDocument()
      expect(badge.querySelector('span')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('should hide icon from screen readers', () => {
      const icon = <span data-testid="badge-icon">★</span>
      render(<Badge icon={icon}>With Icon</Badge>)
      expect(screen.getByTestId('badge-icon')).toHaveAttribute('aria-hidden', 'true')
    })

    it('should hide dot from screen readers', () => {
      render(<Badge variant="dot">Dot</Badge>)
      const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
      expect(dot).toHaveAttribute('aria-hidden', 'true')
    })

    it('should support ARIA attributes', () => {
      render(
        <Badge role="status" aria-label="Active status">
          Active
        </Badge>
      )
      const badge = screen.getByText('Active')
      expect(badge.parentElement).toHaveAttribute('role', 'status')
      expect(badge.parentElement).toHaveAttribute('aria-label', 'Active status')
    })

    it('should support custom data attributes', () => {
      render(<Badge data-testid="test-badge">Test</Badge>)
      expect(screen.getByTestId('test-badge')).toBeInTheDocument()
    })
  })

  describe('Visual Styles', () => {
    it('should have proper rounded corners', () => {
      render(<Badge>Rounded</Badge>)
      const badge = screen.getByText('Rounded').parentElement
      expect(badge).toHaveClass('rounded-full')
    })

    it('should have inline display', () => {
      render(<Badge>Inline</Badge>)
      const badge = screen.getByText('Inline').parentElement
      expect(badge).toHaveClass('inline-flex')
    })

    it('should center items', () => {
      render(<Badge>Centered</Badge>)
      const badge = screen.getByText('Centered').parentElement
      expect(badge).toHaveClass('items-center')
    })
  })

  describe('Color Combinations', () => {
    const colors: BadgeColor[] = ['primary', 'secondary', 'success', 'warning', 'error']
    const variants: BadgeVariant[] = ['solid', 'outline', 'dot']

    colors.forEach((color) => {
      variants.forEach((variant) => {
        it(`should render ${color} color with ${variant} variant`, () => {
          render(
            <Badge color={color} variant={variant}>
              Badge
            </Badge>
          )
          expect(screen.getByText('Badge')).toBeInTheDocument()
        })
      })
    })
  })

  describe('Combinations', () => {
    it('should handle color + variant + className', () => {
      render(
        <Badge color="success" variant="outline" className="px-4">
          Combined
        </Badge>
      )
      const badge = screen.getByText('Combined').parentElement
      expect(badge).toHaveClass('badge-success', 'border-2', 'px-4')
    })

    it('should handle color + variant + icon + className', () => {
      const icon = <span data-testid="icon">★</span>
      render(
        <Badge color="error" variant="dot" icon={icon} className="py-2">
          Combo
        </Badge>
      )
      const badge = screen.getByText('Combo').parentElement
      expect(badge).toHaveClass('badge-error', 'gap-1.5', 'py-2')
      expect(screen.getByTestId('icon')).toBeInTheDocument()
      expect(badge.querySelector('.rounded-full')).toBeInTheDocument()
    })

    it('should handle outline with color', () => {
      render(<Badge color="primary" variant="outline">Outlined Primary</Badge>)
      const badge = screen.getByText('Outlined Primary').parentElement
      expect(badge).toHaveClass('badge-primary', 'border-2', 'border-current')
    })

    it('should handle dot with all colors', () => {
      const colors = ['primary', 'secondary', 'success', 'warning', 'error'] as const
      colors.forEach((color) => {
        render(<Badge color={color} variant="dot">Dot</Badge>)
        const dot = screen.getByText('Dot').parentElement?.querySelector('.rounded-full')
        expect(dot).toHaveClass(`bg-${color}-500`)
      })
    })
  })

  describe('Edge Cases', () => {
    it('should handle very long text', () => {
      const longText = 'A'.repeat(500)
      render(<Badge>{longText}</Badge>)
      expect(screen.getByText(longText)).toBeInTheDocument()
    })

    it('should handle icon as only child', () => {
      const icon = <span data-testid="only-icon">★</span>
      render(<Badge icon={icon}></Badge>)
      expect(screen.getByTestId('only-icon')).toBeInTheDocument()
    })

    it('should handle dot without text', () => {
      render(<Badge variant="dot"></Badge>)
      const badge = screen.getByText('').parentElement
      expect(badge?.querySelector('.rounded-full')).toBeInTheDocument()
    })

    it('should handle null/undefined children', () => {
      render(<Badge>{null}</Badge>)
      const badge = screen.getByText('').parentElement
      expect(badge).toBeInTheDocument()
    })
  })

  describe('Icon Position', () => {
    it('should position icon before text', () => {
      const icon = <span data-testid="icon">★</span>
      render(<Badge icon={icon}>Text</Badge>)
      const badge = screen.getByText('Text').parentElement
      const iconElement = screen.getByTestId('icon')
      expect(badge?.contains(iconElement)).toBe(true)
    })

    it('should position dot before text', () => {
      render(<Badge variant="dot">Text</Badge>)
      const badge = screen.getByText('Text').parentElement
      const dot = badge?.querySelector('.rounded-full')
      expect(badge?.firstChild).toBe(dot)
    })

    it('should order dot, icon, text correctly', () => {
      const icon = <span data-testid="icon">★</span>
      render(<Badge variant="dot" icon={icon}>Text</Badge>)
      const badge = screen.getByText('Text').parentElement
      const children = badge?.children
      expect(children?.[0]).toHaveClass('rounded-full')
      expect(children?.[1]).toHaveAttribute('data-testid', 'icon')
      expect(children?.[2]).toHaveTextContent('Text')
    })
  })
})
