import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test-utils/renderWithTheme'
import { Label } from './Label'

describe('Label', () => {
  describe('Rendering', () => {
    it('should render label text', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toBeInTheDocument()
      expect(label.tagName).toBe('LABEL')
    })

    it('should render with htmlFor attribute', () => {
      render(<Label htmlFor="email">Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveAttribute('for', 'email')
    })

    it('should render without htmlFor attribute', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).not.toHaveAttribute('for')
    })

    it('should show required indicator when required', () => {
      render(<Label required>Password</Label>)
      const label = screen.getByText('Password')
      const asterisk = label.querySelector('span')
      expect(asterisk).toBeInTheDocument()
      expect(asterisk).toHaveTextContent('*')
    })

    it('should not show required indicator when not required', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      const asterisk = label.querySelector('span')
      expect(asterisk).not.toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(<Label className="custom-class">Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('custom-class')
    })

    it('should merge custom className with base classes', () => {
      render(
        <Label className="text-lg mt-2" htmlFor="email">
          Email
        </Label>
      )
      const label = screen.getByText('Email')
      expect(label).toHaveClass('block', 'text-sm', 'font-medium', 'text-lg', 'mt-2')
    })
  })

  describe('Styling', () => {
    it('should have block display', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('block')
    })

    it('should have proper text size', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('text-sm')
    })

    it('should have font-medium', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('font-medium')
    })

    it('should have proper color', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('text-gray-700')
    })

    it('should have bottom margin', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('mb-1.5')
    })

    it('should have transition', () => {
      render(<Label>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveClass('transition-colors')
    })
  })

  describe('Required Indicator', () => {
    it('should render asterisk after text', () => {
      render(<Label required>Password</Label>)
      const label = screen.getByText('Password')
      const asterisk = label.querySelector('span')
      expect(label.childNodes[0]).toHaveTextContent('Password')
      expect(label.childNodes[1]).toBe(asterisk)
    })

    it('should color asterisk as error red', () => {
      render(<Label required>Password</Label>)
      const asterisk = screen.getByText('*')
      expect(asterisk).toHaveClass('text-error-500')
    })

    it('should have left margin on asterisk', () => {
      render(<Label required>Password</Label>)
      const asterisk = screen.getByText('*')
      expect(asterisk).toHaveClass('ml-1')
    })

    it('should hide asterisk from screen readers', () => {
      render(<Label required>Password</Label>)
      const asterisk = screen.getByText('*')
      expect(asterisk).toHaveAttribute('aria-hidden', 'true')
    })
  })

  describe('Accessibility', () => {
    it('should have proper for attribute', () => {
      render(<Label htmlFor="email">Email</Label>)
      const label = screen.getByText('Email')
      expect(label).toHaveAttribute('for', 'email')
    })

    it('should be clickable to focus input', () => {
      render(
        <>
          <Label htmlFor="email">Email</Label>
          <input id="email" />
        </>
      )
      const label = screen.getByText('Email')
      const input = screen.getByRole('textbox')
      label.click()
      expect(input).toHaveFocus()
    })

    it('should support ARIA attributes', () => {
      render(
        <Label htmlFor="email" aria-label="Email address">
          Email
        </Label>
      )
      const label = screen.getByText('Email')
      expect(label).toHaveAttribute('aria-label', 'Email address')
    })

    it('should support custom data attributes', () => {
      render(<Label data-testid="test-label">Email</Label>)
      expect(screen.getByTestId('test-label')).toBeInTheDocument()
    })
  })

  describe('States', () => {
    it('should handle disabled state', () => {
      render(<Label disabled>Disabled</Label>)
      const label = screen.getByText('Disabled')
      expect(label).toBeDisabled()
    })

    it('should have opacity when disabled', () => {
      render(<Label disabled>Disabled</Label>)
      const label = screen.getByText('Disabled')
      expect(label).toHaveClass('opacity-50')
    })

    it('should have cursor-not-allowed when disabled', () => {
      render(<Label disabled>Disabled</Label>)
      const label = screen.getByText('Disabled')
      expect(label).toHaveClass('cursor-not-allowed')
    })
  })

  describe('Content', () => {
    it('should render simple text', () => {
      render(<Label>Email</Label>)
      expect(screen.getByText('Email')).toBeInTheDocument()
    })

    it('should render long text', () => {
      const longText = 'This is a very long label text'
      render(<Label>{longText}</Label>)
      expect(screen.getByText(longText)).toBeInTheDocument()
    })

    it('should render with icon', () => {
      render(
        <Label>
          <span>📧</span> Email
        </Label>
      )
      expect(screen.getByText('📧')).toBeInTheDocument()
      expect(screen.getByText('Email')).toBeInTheDocument()
    })

    it('should render with React elements', () => {
      render(
        <Label>
          <strong>Email</strong>
        </Label>
      )
      const strong = screen.getByText('Email')
      expect(strong.tagName).toBe('STRONG')
    })

    it('should handle empty children', () => {
      render(<Label></Label>)
      const label = screen.getByRole('presentation') || document.querySelector('label')
      expect(label).toBeInTheDocument()
    })
  })

  describe('Combinations', () => {
    it('should handle htmlFor + required + className', () => {
      render(
        <Label htmlFor="email" required className="text-lg">
          Email
        </Label>
      )
      const label = screen.getByText('Email')
      expect(label).toHaveAttribute('for', 'email')
      expect(label).toHaveClass('text-lg')
      expect(screen.getByText('*')).toBeInTheDocument()
    })

    it('should handle htmlFor + disabled', () => {
      render(
        <Label htmlFor="disabled" disabled>
          Disabled
        </Label>
      )
      const label = screen.getByText('Disabled')
      expect(label).toHaveAttribute('for', 'disabled')
      expect(label).toBeDisabled()
    })

    it('should handle required + disabled', () => {
      render(<Label required disabled>Password</Label>)
      const label = screen.getByText('Password')
      expect(label).toBeDisabled()
      expect(screen.getByText('*')).toBeInTheDocument()
    })

    it('should handle all props together', () => {
      render(
        <Label
          htmlFor="password"
          required
          disabled
          className="mt-4"
        >
          Password
        </Label>
      )
      const label = screen.getByText('Password')
      expect(label).toHaveAttribute('for', 'password')
      expect(label).toBeDisabled()
      expect(label).toHaveClass('mt-4')
      expect(screen.getByText('*')).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    it('should handle whitespace', () => {
      render(<Label>   Email   </Label>)
      const label = screen.getByText('   Email   ')
      expect(label).toBeInTheDocument()
    })

    it('should handle special characters', () => {
      render(<Label>Email & Password</Label>)
      expect(screen.getByText(/Email & Password/)).toBeInTheDocument()
    })

    it('should handle emoji', () => {
      render(<Label>📧 Email Address</Label>)
      expect(screen.getByText('📧 Email Address')).toBeInTheDocument()
    })

    it('should handle numbers', () => {
      render(<Label>Step 1: Email</Label>)
      expect(screen.getByText('Step 1: Email')).toBeInTheDocument()
    })

    it('should handle null htmlFor', () => {
      render(<Label htmlFor={undefined as any}>Email</Label>)
      const label = screen.getByText('Email')
      expect(label).not.toHaveAttribute('for')
    })
  })

  describe('Integration with Form Elements', () => {
    it('should properly connect to input', () => {
      render(
        <>
          <Label htmlFor="email">Email</Label>
          <input id="email" type="email" />
        </>
      )
      const label = screen.getByText('Email')
      const input = screen.getByRole('textbox')
      label.click()
      expect(input).toHaveFocus()
    })

    it('should properly connect to textarea', () => {
      render(
        <>
          <Label htmlFor="message">Message</Label>
          <textarea id="message" />
        </>
      )
      const label = screen.getByText('Message')
      const textarea = screen.getByRole('textbox')
      label.click()
      expect(textarea).toHaveFocus()
    })

    it('should properly connect to select', () => {
      render(
        <>
          <Label htmlFor="country">Country</Label>
          <select id="country">
            <option>USA</option>
          </select>
        </>
      )
      const label = screen.getByText('Country')
      const select = screen.getByRole('combobox')
      label.click()
      expect(select).toHaveFocus()
    })
  })
})
