import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithTheme'
import userEvent from '@testing-library/user-event'
import { Input } from './Input'

describe('Input', () => {
  describe('Rendering', () => {
    it('should render text input by default', () => {
      render(<Input type="text" />)
      const input = screen.getByRole('textbox')
      expect(input).toBeInTheDocument()
      expect(input).toHaveAttribute('type', 'text')
    })

    it('should render email input', () => {
      render(<Input type="email" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('type', 'email')
    })

    it('should render password input', () => {
      render(<Input type="password" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('type', 'password')
    })

    it('should render search input', () => {
      render(<Input type="search" />)
      const input = screen.getByRole('searchbox')
      expect(input).toBeInTheDocument()
    })

    it('should render number input', () => {
      render(<Input type="number" />)
      const input = screen.getByRole('spinbutton')
      expect(input).toBeInTheDocument()
    })

    it('should render url input', () => {
      render(<Input type="url" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('type', 'url')
    })

    it('should render tel input', () => {
      render(<Input type="tel" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('type', 'tel')
    })

    it('should render sm size', () => {
      render(<Input size="sm" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('px-3', 'py-1.5', 'text-sm')
    })

    it('should render md size by default', () => {
      render(<Input />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('px-4', 'py-2.5', 'text-base')
    })

    it('should render lg size', () => {
      render(<Input size="lg" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('px-5', 'py-3', 'text-lg')
    })

    it('should render with placeholder', () => {
      render(<Input placeholder="Enter text" />)
      const input = screen.getByPlaceholderText('Enter text')
      expect(input).toBeInTheDocument()
    })

    it('should render with value', () => {
      render(<Input defaultValue="test value" />)
      const input = screen.getByDisplayValue('test value')
      expect(input).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(<Input className="custom-class" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('custom-class')
    })
  })

  describe('Error State', () => {
    it('should render error state', () => {
      render(<Input error />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('border-error-500')
    })

    it('should render with error message', () => {
      render(<Input error errorMessage="This field is required" />)
      expect(screen.getByText('This field is required')).toBeInTheDocument()
    })

    it('should have aria-invalid when error', () => {
      render(<Input error />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('aria-invalid', 'true')
    })

    it('should not have aria-invalid when not error', () => {
      render(<Input error={false} />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('aria-invalid', 'false')
    })

    it('should have aria-describedby for error message', () => {
      render(<Input id="test" error errorMessage="Error" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('aria-describedby', 'test-error')
      expect(screen.getByText('Error')).toHaveAttribute('id', 'test-error')
    })

    it('should not show error message when no errorMessage', () => {
      render(<Input error />)
      const errorMessage = screen.queryByRole('alert')
      expect(errorMessage).not.toBeInTheDocument()
    })
  })

  describe('Password Toggle', () => {
    it('should show password toggle for password type', () => {
      render(<Input type="password" />)
      const toggleButton = screen.getByRole('button', { name: /show password/i })
      expect(toggleButton).toBeInTheDocument()
    })

    it('should toggle password visibility', async () => {
      const user = userEvent.setup()
      render(<Input type="password" />)
      
      const input = screen.getByRole('textbox')
      const toggleButton = screen.getByRole('button', { name: /show password/i })
      
      expect(input).toHaveAttribute('type', 'password')
      
      await user.click(toggleButton)
      expect(input).toHaveAttribute('type', 'text')
      
      await user.click(toggleButton)
      expect(input).toHaveAttribute('type', 'password')
    })

    it('should update aria-pressed on toggle', async () => {
      const user = userEvent.setup()
      render(<Input type="password" />)
      
      const toggleButton = screen.getByRole('button')
      expect(toggleButton).toHaveAttribute('aria-pressed', 'false')
      
      await user.click(toggleButton)
      expect(toggleButton).toHaveAttribute('aria-pressed', 'true')
    })

    it('should update aria-label on toggle', async () => {
      const user = userEvent.setup()
      render(<Input type="password" />)
      
      const toggleButton = screen.getByRole('button')
      expect(toggleButton).toHaveAttribute('aria-label', 'Show password')
      
      await user.click(toggleButton)
      expect(toggleButton).toHaveAttribute('aria-label', 'Hide password')
    })

    it('should not show toggle when disabled', () => {
      render(<Input type="password" disabled />)
      const toggleButton = screen.queryByRole('button', { name: /show password/i })
      expect(toggleButton).not.toBeInTheDocument()
    })

    it('should not show toggle when readonly', () => {
      render(<Input type="password" readOnly />)
      const toggleButton = screen.queryByRole('button', { name: /show password/i })
      expect(toggleButton).not.toBeInTheDocument()
    })

    it('should respect showPasswordToggle prop', () => {
      render(<Input type="password" showPasswordToggle={false} />)
      const toggleButton = screen.queryByRole('button', { name: /show password/i })
      expect(toggleButton).not.toBeInTheDocument()
    })
  })

  describe('Icons', () => {
    it('should render start icon', () => {
      const icon = <span data-testid="start-icon">★</span>
      render(<Input startIcon={icon} />)
      expect(screen.getByTestId('start-icon')).toBeInTheDocument()
    })

    it('should render end icon', () => {
      const icon = <span data-testid="end-icon">★</span>
      render(<Input endIcon={icon} />)
      expect(screen.getByTestId('end-icon')).toBeInTheDocument()
    })

    it('should render both icons', () => {
      const startIcon = <span data-testid="start-icon">★</span>
      const endIcon = <span data-testid="end-icon">★</span>
      render(<Input startIcon={startIcon} endIcon={endIcon} />)
      expect(screen.getByTestId('start-icon')).toBeInTheDocument()
      expect(screen.getByTestId('end-icon')).toBeInTheDocument()
    })

    it('should hide icons from screen readers', () => {
      const icon = <span data-testid="icon">★</span>
      render(<Input startIcon={icon} />)
      expect(screen.getByTestId('icon')).toHaveAttribute('aria-hidden', 'true')
    })

    it('should not show password toggle with end icon', () => {
      const icon = <span data-testid="end-icon">★</span      render(<Input type="password" endIcon={icon} />)
      const toggleButton = screen.queryByRole('button', { name: /show password/i })
      expect(toggleButton).not.toBeInTheDocument()
      expect(screen.getByTestId('end-icon')).toBeInTheDocument()
    })
  })

  describe('States', () => {
    it('should handle disabled state', () => {
      render(<Input disabled />)
      const input = screen.getByRole('textbox')
      expect(input).toBeDisabled()
    })

    it('should have cursor-not-allowed when disabled', () => {
      render(<Input disabled />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('cursor-not-allowed')
    })

    it('should handle readonly state', () => {
      render(<Input readOnly value="readonly" />)
      const input = screen.getByDisplayValue('readonly')
      expect(input).toHaveAttribute('readonly')
    })

    it('should have bg-gray-50 when readonly', () => {
      render(<Input readOnly />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('bg-gray-50')
    })

    it('should have cursor-default when readonly', () => {
      render(<Input readOnly />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('cursor-default')
    })
  })

  describe('Focus States', () => {
    it('should show focus ring on focus', () => {
      render(<Input />)
      const input = screen.getByRole('textbox')
      input.focus()
      expect(input).toHaveFocus()
    })

    it('should have proper focus styles', () => {
      render(<Input />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('focus:border-primary-500', 'focus:ring-primary-100')
    })
  })

  describe('Value Handling', () => {
    it('should handle controlled value', () => {
      const handleChange = vi.fn()
      render(<Input value="controlled" onChange={handleChange} />)
      const input = screen.getByDisplayValue('controlled')
      expect(input).toBeInTheDocument()
    })

    it('should handle uncontrolled defaultValue', () => {
      render(<Input defaultValue="uncontrolled" />)
      expect(screen.getByDisplayValue('uncontrolled')).toBeInTheDocument()
    })

    it('should call onChange when value changes', async () => {
      const user = userEvent.setup()
      const handleChange = vi.fn()
      render(<Input onChange={handleChange} />)
      
      const input = screen.getByRole('textbox')
      await user.type(input, 'test')
      
      expect(handleChange).toHaveBeenCalled()
    })

    it('should handle number input', async () => {
      const user = userEvent.setup()
      const handleChange = vi.fn()
      render(<Input type="number" onChange={handleChange} />)
      
      const input = screen.getByRole('spinbutton')
      await user.type(input, '123')
      
      expect(handleChange).toHaveBeenCalled()
    })
  })

  describe('Accessibility', () => {
    it('should have proper id attribute', () => {
      render(<Input id="test-input" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('id', 'test-input')
    })

    it('should have proper name attribute', () => {
      render(<Input name="test-name" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('name', 'test-name')
    })

    it('should be keyboard accessible', async () => {
      const user = userEvent.setup()
      render(<Input />)
      
      const input = screen.getByRole('textbox')
      await user.tab()
      expect(input).toHaveFocus()
    })

    it('should have proper aria attributes for error', () => {
      render(<Input id="test" error errorMessage="Error message" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('aria-invalid', 'true')
      expect(input).toHaveAttribute('aria-describedby', 'test-error')
    })

    it('should have role="alert" on error message', () => {
      render(<Input error errorMessage="Error" />)
      const error = screen.getByText('Error')
      expect(error).toHaveAttribute('role', 'alert')
    })
  })

  describe('Padding with Icons', () => {
    it('should have pl-10 with start icon', () => {
      const icon = <span>★</span>
      render(<Input startIcon={icon} />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('pl-10')
    })

    it('should have pr-10 with end icon', () => {
      const icon = <span>★</span>
      render(<Input endIcon={icon} />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('pr-10')
    })

    it('should have pr-10 with password toggle', () => {
      render(<Input type="password" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('pr-10')
    })
  })

  describe('Combinations', () => {
    it('should handle type + size + error', () => {
      render(
        <Input type="email" size="lg" error errorMessage="Invalid" />
      )
      const input = screen.getByRole('textbox')
      expect(input).toHaveClass('px-5', 'py-3', 'text-lg', 'border-error-500')
      expect(screen.getByText('Invalid')).toBeInTheDocument()
    })

    it('should handle icons + password', () => {
      const icon = <span data-testid="icon">★</span>
      render(<Input type="password" startIcon={icon} />)
      expect(screen.getByTestId('icon')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /show password/i })).toBeInTheDocument()
    })

    it('should handle disabled + error', () => {
      render(<Input disabled error errorMessage="Error" />)
      const input = screen.getByRole('textbox')
      expect(input).toBeDisabled()
      expect(input).toHaveClass('border-error-500')
    })

    it('should handle readonly + size', () => {
      render(<Input readOnly size="sm" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('readonly')
      expect(input).toHaveClass('px-3', 'py-1.5', 'text-sm')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty value', () => {
      render(<Input value="" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveValue('')
    })

    it('should handle whitespace', () => {
      render(<Input value="   " />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveValue('   ')
    })

    it('should handle long value', () => {
      const longValue = 'A'.repeat(1000)
      render(<Input defaultValue={longValue} />)
      expect(screen.getByDisplayValue(longValue)).toBeInTheDocument()
    })

    it('should handle special characters in value', () => {
      render(<Input defaultValue="Special & < > '" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveValue('Special & < > "')
    })

    it('should handle maxLength', () => {
      render(<Input maxLength={10} />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('maxlength', '10')
    })

    it('should handle minLength', () => {
      render(<Input minLength={5} />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('minlength', '5')
    })

    it('should handle pattern', () => {
      render(<Input pattern="[a-z]+" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('pattern', '[a-z]+')
    })
  })

  describe('Integration with Label', () => {
    it('should work with Label component', () => {
      render(
        <>
          <label htmlFor="test">Test</label>
          <Input id="test" />
        </>
      )
      const label = screen.getByText('Test')
      const input = screen.getByRole('textbox')
      label.click()
      expect(input).toHaveFocus()
    })
  })

  describe('Form Attributes', () => {
    it('should support autoComplete', () => {
      render(<Input autoComplete="email" />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('autocomplete', 'email')
    })

    it('should support autoFocus', () => {
      render(<Input autoFocus />)
      const input = screen.getByRole('textbox')
      expect(input).toHaveAttribute('autofocus')
    })

    it('should support step for number input', () => {
      render(<Input type="number" step="0.01" />)
      const input = screen.getByRole('spinbutton')
      expect(input).toHaveAttribute('step', '0.01')
    })

    it('should support min/max for number input', () => {
      render(<Input type="number" min="0" max="100" />)
      const input = screen.getByRole('spinbutton')
      expect(input).toHaveAttribute('min', '0')
      expect(input).toHaveAttribute('max', '100')
    })
  })
})
