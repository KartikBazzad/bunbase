import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithTheme'
import userEvent from '@testing-library/user-event'
import { Button } from './Button'

describe('Button', () => {
  describe('Rendering', () => {
    it('should render primary variant', () => {
      render(<Button variant="primary">Primary</Button>)
      const button = screen.getByRole('button', { name: 'Primary' })
      expect(button).toBeInTheDocument()
      expect(button).toHaveClass('btn-primary')
    })

    it('should render secondary variant', () => {
      render(<Button variant="secondary">Secondary</Button>)
      const button = screen.getByRole('button', { name: 'Secondary' })
      expect(button).toHaveClass('btn-secondary')
    })

    it('should render ghost variant', () => {
      render(<Button variant="ghost">Ghost</Button>)
      const button = screen.getByRole('button', { name: 'Ghost' })
      expect(button).toHaveClass('btn-ghost')
    })

    it('should render danger variant', () => {
      render(<Button variant="danger">Danger</Button>)
      const button = screen.getByRole('button', { name: 'Danger' })
      expect(button).toHaveClass('btn-danger')
    })

    it('should render outline variant', () => {
      render(<Button variant="outline">Outline</Button>)
      const button = screen.getByRole('button', { name: 'Outline' })
      expect(button).toHaveClass('btn-outline')
    })

    it('should render xs size', () => {
      render(<Button size="xs">Extra Small</Button>)
      const button = screen.getByRole('button', { name: 'Extra Small' })
      expect(button).toHaveClass('btn-xs')
    })

    it('should render sm size', () => {
      render(<Button size="sm">Small</Button>)
      const button = screen.getByRole('button', { name: 'Small' })
      expect(button).toHaveClass('btn-sm')
    })

    it('should render md size by default', () => {
      render(<Button>Medium</Button>)
      const button = screen.getByRole('button', { name: 'Medium' })
      expect(button).toHaveClass('btn-md')
    })

    it('should render lg size', () => {
      render(<Button size="lg">Large</Button>)
      const button = screen.getByRole('button', { name: 'Large' })
      expect(button).toHaveClass('btn-lg')
    })

    it('should render xl size', () => {
      render(<Button size="xl">Extra Large</Button>)
      const button = screen.getByRole('button', { name: 'Extra Large' })
      expect(button).toHaveClass('btn-xl')
    })

    it('should render with loading state', () => {
      render(<Button loading>Loading</Button>)
      const button = screen.getByRole('button', { name: 'Loading' })
      expect(button).toBeDisabled()
      expect(button).toHaveAttribute('aria-busy', 'true')
      expect(screen.getByText('Loading')).toBeInTheDocument()
    })

    it('should show spinner when loading', () => {
      render(<Button loading>Loading</Button>)
      const spinner = screen.getByRole('button').querySelector('.spinner')
      expect(spinner).toBeInTheDocument()
    })

    it('should render disabled state', () => {
      render(<Button disabled>Disabled</Button>)
      const button = screen.getByRole('button', { name: 'Disabled' })
      expect(button).toBeDisabled()
      expect(button).toHaveAttribute('disabled')
    })

    it('should render with start icon', () => {
      const icon = <span data-testid="start-icon">→</span>
      render(<Button startIcon={icon}>With Icon</Button>)
      expect(screen.getByTestId('start-icon')).toBeInTheDocument()
    })

    it('should render with end icon', () => {
      const icon = <span data-testid="end-icon">→</span>
      render(<Button endIcon={icon}>With Icon</Button>)
      expect(screen.getByTestId('end-icon')).toBeInTheDocument()
    })

    it('should render with both icons', () => {
      const startIcon = <span data-testid="start-icon">←</span>
      const endIcon = <span data-testid="end-icon">→</span>
      render(
        <Button startIcon={startIcon} endIcon={endIcon}>
          With Icons
        </Button>
      )
      expect(screen.getByTestId('start-icon')).toBeInTheDocument()
      expect(screen.getByTestId('end-icon')).toBeInTheDocument()
    })

    it('should not show icons when loading', () => {
      const startIcon = <span data-testid="start-icon">←</span>
      const endIcon = <span data-testid="end-icon">→</span>
      render(
        <Button loading startIcon={startIcon} endIcon={endIcon}>
          Loading
        </Button>
      )
      expect(screen.queryByTestId('start-icon')).not.toBeInTheDocument()
      expect(screen.queryByTestId('end-icon')).not.toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(<Button className="custom-class">Custom</Button>)
      const button = screen.getByRole('button', { name: 'Custom' })
      expect(button).toHaveClass('custom-class')
    })

    it('should merge custom className with base classes', () => {
      render(
        <Button variant="primary" size="lg" className="mt-4">
          Combined
        </Button>
      )
      const button = screen.getByRole('button', { name: 'Combined' })
      expect(button).toHaveClass('btn', 'btn-primary', 'btn-lg', 'mt-4')
    })
  })

  describe('Interactions', () => {
    it('should call onClick handler when clicked', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(<Button onClick={handleClick}>Click me</Button>)

      const button = screen.getByRole('button')
      await user.click(button)

      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('should not call onClick when disabled', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(
        <Button onClick={handleClick} disabled>
          Disabled
        </Button>
      )

      const button = screen.getByRole('button')
      await user.click(button)

      expect(handleClick).not.toHaveBeenCalled()
    })

    it('should not call onClick when loading', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(
        <Button onClick={handleClick} loading>
          Loading
        </Button>
      )

      const button = screen.getByRole('button')
      await user.click(button)

      expect(handleClick).not.toHaveBeenCalled()
    })

    it('should handle keyboard Enter key', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(<Button onClick={handleClick}>Submit</Button>)

      const button = screen.getByRole('button')
      button.focus()
      await user.keyboard('{Enter}')

      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('should handle keyboard Space key', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(<Button onClick={handleClick}>Submit</Button>)

      const button = screen.getByRole('button')
      button.focus()
      await user.keyboard(' ')

      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('should receive focus when tabbed to', () => {
      render(<Button>Focusable</Button>)
      const button = screen.getByRole('button')
      button.focus()
      expect(button).toHaveFocus()
    })

    it('should handle long text gracefully', () => {
      const longText = 'This is a very long button text that should wrap or truncate properly without breaking the layout'
      render(<Button>{longText}</Button>)
      expect(screen.getByText(longText)).toBeInTheDocument()
    })

    it('should render nested elements', () => {
      render(
        <Button>
          <span className="font-bold">Bold Text</span>
          <span> and Regular Text</span>
        </Button>
      )
      const button = screen.getByRole('button')
      expect(button.querySelector('.font-bold')).toBeInTheDocument()
    })

    it('should render with React elements as children', () => {
      render(
        <Button>
          <div>Custom Content</div>
          <small>Subtext</small>
        </Button>
      )
      expect(screen.getByText('Custom Content')).toBeInTheDocument()
      expect(screen.getByText('Subtext')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('should have proper role="button"', () => {
      render(<Button>Button</Button>)
      expect(screen.getByRole('button')).toBeInTheDocument()
    })

    it('should set aria-busy when loading', () => {
      render(<Button loading>Loading</Button>)
      expect(screen.getByRole('button')).toHaveAttribute('aria-busy', 'true')
    })

    it('should not set aria-busy when not loading', () => {
      render(<Button>Not Loading</Button>)
      expect(screen.getByRole('button')).not.toHaveAttribute('aria-busy')
    })

    it('should be keyboard accessible', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(<Button onClick={handleClick}>Accessible</Button>)

      const button = screen.getByRole('button')
      await user.tab()
      expect(button).toHaveFocus()
      await user.keyboard('{Enter}')
      expect(handleClick).toHaveBeenCalled()
    })

    it('should hide icons from screen readers', () => {
      const startIcon = <span data-testid="start-icon">←</span>
      const endIcon = <span data-testid="end-icon">→</span>
      render(
        <Button startIcon={startIcon} endIcon={endIcon}>
          Button Text
        </Button>
      )

      expect(screen.getByTestId('start-icon')).toHaveAttribute('aria-hidden', 'true')
      expect(screen.getByTestId('end-icon')).toHaveAttribute('aria-hidden', 'true')
    })

    it('should have proper type attribute', () => {
      const { rerender } = render(<Button type="submit">Submit</Button>)
      expect(screen.getByRole('button')).toHaveAttribute('type', 'submit')

      rerender(<Button type="reset">Reset</Button>)
      expect(screen.getByRole('button')).toHaveAttribute('type', 'reset')

      rerender(<Button type="button">Button</Button>)
      expect(screen.getByRole('button')).toHaveAttribute('type', 'button')
    })

    it('should default to type="button"', () => {
      render(<Button>Default</Button>)
      expect(screen.getByRole('button')).toHaveAttribute('type', 'button')
    })
  })

  describe('Visual States', () => {
    it('should have focus ring on focus', () => {
      render(<Button>Focus Test</Button>)
      const button = screen.getByRole('button')
      button.focus()
      
      expect(button).toHaveClass('focus-visible:ring-2', 'focus-visible:ring-primary-500')
    })

    it('should not have focus ring when not focused', () => {
      render(<Button>Focus Test</Button>)
      const button = screen.getByRole('button')
      
      expect(button).not.toHaveClass('ring-2')
    })

    it('should apply opacity when disabled', () => {
      render(<Button disabled>Disabled</Button>)
      const button = screen.getByRole('button')
      
      expect(button).toHaveClass('opacity-50', 'cursor-not-allowed')
    })
  })

  describe('Combined Props', () => {
    it('should handle variant + size + disabled combination', () => {
      render(
        <Button variant="danger" size="xl" disabled>
          Combo
        </Button>
      )
      const button = screen.getByRole('button')
      expect(button).toHaveClass('btn-danger', 'btn-xl')
      expect(button).toBeDisabled()
    })

    it('should handle loading + variant + icons', () => {
      const icon = <span data-testid="icon">Icon</span>
      render(
        <Button loading variant="primary" startIcon={icon}>
          Loading
        </Button>
      )
      const button = screen.getByRole('button')
      expect(button).toHaveClass('btn-primary')
      expect(button).toBeDisabled()
      expect(screen.queryByTestId('icon')).not.toBeInTheDocument()
      expect(screen.getByRole('button').querySelector('.spinner')).toBeInTheDocument()
    })

    it('should handle custom className + variant + size', () => {
      render(
        <Button variant="outline" size="sm" className="w-full mt-4">
          Full Width
        </Button>
      )
      const button = screen.getByRole('button')
      expect(button).toHaveClass('btn-outline', 'btn-sm', 'w-full', 'mt-4')
    })

    it('should pass through additional HTML attributes', () => {
      render(
        <Button
          data-testid="test-button"
          id="my-button"
          title="Button title"
          aria-label="Aria label"
        >
          Attributes
        </Button>
      )
      const button = screen.getByRole('button')
      expect(button).toHaveAttribute('data-testid', 'test-button')
      expect(button).toHaveAttribute('id', 'my-button')
      expect(button).toHaveAttribute('title', 'Button title')
      expect(button).toHaveAttribute('aria-label', 'Aria label')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty text', () => {
      render(<Button></Button>)
      const button = screen.getByRole('button')
      expect(button).toBeInTheDocument()
      expect(button).toBeEmptyDOMElement()
    })

    it('should handle whitespace only', () => {
      render(<Button>   </Button>)
      const button = screen.getByRole('button')
      expect(button).toHaveTextContent('   ')
    })

    it('should handle special characters', () => {
      render(<Button>Special & < > " \'</Button>)
      const button = screen.getByRole('button')
      expect(button).toBeInTheDocument()
    })

    it('should handle emoji', () => {
      render(<Button>🎉 Celebrate! 🚀</Button>)
      const button = screen.getByRole('button')
      expect(button).toHaveTextContent('🎉 Celebrate! 🚀')
    })

    it('should handle very long text', () => {
      const longText = 'A'.repeat(1000)
      render(<Button>{longText}</Button>)
      const button = screen.getByRole('button')
      expect(button).toHaveTextContent(longText)
    })

    it('should handle null/undefined children gracefully', () => {
      render(<Button>{null}</Button>)
      const button = screen.getByRole('button')
      expect(button).toBeInTheDocument()
    })
  })

  describe('Form Integration', () => {
    it('should work as form submit button', () => {
      const handleSubmit = vi.fn((e) => e.preventDefault())
      render(
        <form onSubmit={handleSubmit}>
          <Button type="submit">Submit Form</Button>
        </form>
      )
      
      const button = screen.getByRole('button', { name: 'Submit Form' })
      expect(button).toHaveAttribute('type', 'submit')
    })

    it('should work as form reset button', () => {
      render(
        <form>
          <Button type="reset">Reset Form</Button>
        </form>
      )
      
      const button = screen.getByRole('button', { name: 'Reset Form' })
      expect(button).toHaveAttribute('type', 'reset')
    })
  })
})
