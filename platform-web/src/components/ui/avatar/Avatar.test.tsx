import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithTheme'
import userEvent from '@testing-library/user-event'
import { Avatar } from './Avatar'

describe('Avatar', () => {
  describe('Rendering', () => {
    it('should render with image', () => {
      render(<Avatar src="/test.jpg" alt="Test User" />)
      const img = screen.getByAltText('Test User')
      expect(img).toBeInTheDocument()
      expect(img).toHaveAttribute('src', '/test.jpg')
    })

    it('should render xs size', () => {
      render(<Avatar src="/test.jpg" alt="XS" size="xs" />)
      const container = screen.getByAltText('XS').parentElement
      expect(container).toHaveClass('h-6', 'w-6', 'text-xs')
    })

    it('should render sm size', () => {
      render(<Avatar src="/test.jpg" alt="SM" size="sm" />)
      const container = screen.getByAltText('SM').parentElement
      expect(container).toHaveClass('h-8', 'w-8', 'text-sm')
    })

    it('should render md size by default', () => {
      render(<Avatar src="/test.jpg" alt="MD" />)
      const container = screen.getByAltText('MD').parentElement
      expect(container).toHaveClass('h-10', 'w-10', 'text-base')
    })

    it('should render lg size', () => {
      render(<Avatar src="/test.jpg" alt="LG" size="lg" />)
      const container = screen.getByAltText('LG').parentElement
      expect(container).toHaveClass('h-12', 'w-12', 'text-lg')
    })

    it('should render xl size', () => {
      render(<Avatar src="/test.jpg" alt="XL" size="xl" />)
      const container = screen.getByAltText('XL').parentElement
      expect(container).toHaveClass('h-16', 'w-16', 'text-xl')
    })

    it('should render with border', () => {
      render(<Avatar src="/test.jpg" alt="Bordered" bordered />)
      const container = screen.getByAltText('Bordered').parentElement
      expect(container).toHaveClass('ring-2', 'ring-white')
    })

    it('should render without border by default', () => {
      render(<Avatar src="/test.jpg" alt="No Border" />)
      const container = screen.getByAltText('No Border').parentElement
      expect(container).not.toHaveClass('ring-2')
    })

    it('should apply custom className', () => {
      render(<Avatar src="/test.jpg" alt="Custom" className="mt-4" />)
      const container = screen.getByAltText('Custom').parentElement
      expect(container).toHaveClass('mt-4')
    })

    it('should have rounded-full shape', () => {
      render(<Avatar src="/test.jpg" alt="Rounded" />)
      const container = screen.getByAltText('Rounded').parentElement
      expect(container).toHaveClass('rounded-full')
    })

    it('should have inline-flex display', () => {
      render(<Avatar src="/test.jpg" alt="Inline" />)
      const container = screen.getByAltText('Inline').parentElement
      expect(container).toHaveClass('inline-flex')
    })

    it('should have proper object-fit for image', () => {
      render(<Avatar src="/test.jpg" alt="Fit" />)
      const img = screen.getByAltText('Fit')
      expect(img).toHaveClass('object-cover')
    })
  })

  describe('Fallback Handling', () => {
    it('should render initials when no src provided', () => {
      render(<Avatar fallback="JD" alt="John Doe" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should generate initials from full name', () => {
      render(<Avatar fallback="John Doe" alt="Full Name" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should limit initials to 2 characters', () => {
      render(<Avatar fallback="John Michael Doe" alt="Long Name" />)
      expect(screen.getByText('JM')).toBeInTheDocument()
    })

    it('should convert initials to uppercase', () => {
      render(<Avatar fallback="john doe" alt="Lowercase" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should show fallback when image fails to load', () => {
      render(
        <Avatar src="/invalid.jpg" fallback="JD" alt="Error Image" />
      )
      const img = screen.getByAltText('Error Image')
      fireEvent.error(img)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should show fallback when src is empty string', () => {
      render(<Avatar src="" fallback="JD" alt="Empty Src" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should render custom background color for fallback', () => {
      render(
        <Avatar
          fallback="JD"
          fallbackBgColor="bg-blue-500"
          alt="Custom Bg"
        />
      )
      const text = screen.getByText('JD')
      expect(text).toHaveClass('bg-blue-500')
    })

    it('should render custom text color for fallback', () => {
      render(
        <Avatar
          fallback="JD"
          fallbackTextColor="text-black"
          alt="Custom Text"
        />
      )
      const text = screen.getByText('JD')
      expect(text).toHaveClass('text-black')
    })

    it('should handle single character fallback', () => {
      render(<Avatar fallback="J" alt="Single" />)
      expect(screen.getByText('J')).toBeInTheDocument()
    })

    it('should handle special characters in fallback', () => {
      render(<Avatar fallback="J D" alt="Special" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })
  })

  describe('Image Loading', () => {
    it('should handle successful image load', () => {
      render(<Avatar src="/test.jpg" alt="Load" />)
      const img = screen.getByAltText('Load')
      fireEvent.load(img)
      expect(img).toBeInTheDocument()
    })

    it('should remove fallback when image loads successfully', () => {
      render(<Avatar src="/test.jpg" fallback="JD" alt="With Fallback" />)
      const img = screen.getByAltText('With Fallback')
      expect(screen.queryByText('JD')).not.toBeInTheDocument()
    })

    it('should handle image with loading state', () => {
      render(<Avatar src="/test.jpg" alt="Loading" />)
      const img = screen.getByAltText('Loading')
      expect(img).toHaveAttribute('src', '/test.jpg')
    })
  })

  describe('Interactions', () => {
    it('should handle onClick when clickable', () => {
      const handleClick = vi.fn()
      render(
        <Avatar
          src="/test.jpg"
          alt="Clickable"
          onClick={handleClick}
        />
      )
      const avatar = screen.getByRole('button')
      avatar.click()
      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('should have cursor-pointer when onClick is provided', () => {
      render(<Avatar src="/test.jpg" alt="Pointer" onClick={() => {}} />)
      const avatar = screen.getByRole('button')
      expect(avatar).toHaveClass('cursor-pointer')
    })

    it('should not have cursor-pointer when onClick is not provided', () => {
      render(<Avatar src="/test.jpg" alt="No Pointer" />)
      const container = screen.getByAltText('No Pointer').parentElement
      expect(container).not.toHaveClass('cursor-pointer')
    })

    it('should be keyboard accessible when clickable', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()
      render(
        <Avatar
          src="/test.jpg"
          alt="Keyboard"
          onClick={handleClick}
        />
      )
      const avatar = screen.getByRole('button')
      avatar.focus()
      expect(avatar).toHaveFocus()
      await user.keyboard('{Enter}')
      expect(handleClick).toHaveBeenCalled()
    })

    it('should have tabindex 0 when clickable', () => {
      render(<Avatar src="/test.jpg" alt="Tabindex" onClick={() => {}} />)
      const avatar = screen.getByRole('button')
      expect(avatar).toHaveAttribute('tabIndex', '0')
    })

    it('should not have tabindex when not clickable', () => {
      render(<Avatar src="/test.jpg" alt="No Tabindex" />)
      const container = screen.getByAltText('No Tabindex').parentElement
      expect(container).not.toHaveAttribute('tabIndex')
    })
  })

  describe('Accessibility', () => {
    it('should have proper alt text', () => {
      render(<Avatar src="/test.jpg" alt="User Avatar" />)
      expect(screen.getByAltText('User Avatar')).toBeInTheDocument()
    })

    it('should have aria-label when clickable', () => {
      render(<Avatar src="/test.jpg" alt="Accessible" onClick={() => {}} />)
      const avatar = screen.getByRole('button')
      expect(avatar).toHaveAttribute('aria-label', 'Accessible')
    })

    it('should announce fallback text', () => {
      render(<Avatar fallback="JD" alt="John Doe" />)
      const initials = screen.getByText('JD')
      expect(initials).toHaveAttribute('aria-label', 'JD')
    })

    it('should have role="button" when clickable', () => {
      render(<Avatar src="/test.jpg" alt="Button" onClick={() => {}} />)
      expect(screen.getByRole('button')).toBeInTheDocument()
    })

    it('should not have role when not clickable', () => {
      render(<Avatar src="/test.jpg" alt="No Role" />)
      const container = screen.getByAltText('No Role').parentElement
      expect(container).not.toHaveAttribute('role')
    })
  })

  describe('Color Generation', () => {
    const testColors = [
      'bg-red-500',
      'bg-blue-500',
      'bg-green-500',
      'bg-purple-500',
      'bg-orange-500',
      'bg-pink-500',
    ]

    it('should generate different colors for different initials', () => {
      const avatars = ['JD', 'AB', 'XY']
      avatars.forEach(initials => {
        render(<Avatar fallback={initials} alt={initials} />)
        const text = screen.getByText(initials)
        expect(text).toBeInTheDocument()
      })
    })

    it('should use custom color when provided', () => {
      render(
        <Avatar
          fallback="JD"
          fallbackBgColor="bg-purple-500"
          alt="Custom"
        />
      )
      const text = screen.getByText('JD')
      expect(text).toHaveClass('bg-purple-500')
    })

    it('should use white text by default for fallback', () => {
      render(<Avatar fallback="JD" alt="Default Text" />)
      const text = screen.getByText('JD')
      expect(text).toHaveClass('text-white')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty alt text', () => {
      render(<Avatar src="/test.jpg" alt="" />)
      const img = screen.getByAltText('')
      expect(img).toBeInTheDocument()
    })

    it('should handle undefined src', () => {
      render(<Avatar fallback="JD" alt="Undefined" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should handle null src', () => {
      render(
        <Avatar src={null as any} fallback="JD" alt="Null" />
      )
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should handle very long fallback text', () => {
      const longText = 'A'.repeat(100)
      render(<Avatar fallback={longText} alt="Long" />)
      expect(screen.getByText('A')).toBeInTheDocument()
    })

    it('should handle whitespace in fallback', () => {
      render(<Avatar fallback="   JD   " alt="Whitespace" />)
      expect(screen.getByText('JD')).toBeInTheDocument()
    })

    it('should handle emoji in fallback', () => {
      render(<Avatar fallback="😀" alt="Emoji" />)
      expect(screen.getByText('😀')).toBeInTheDocument()
    })

    it('should handle numbers in fallback', () => {
      render(<Avatar fallback="123" alt="Numbers" />)
      expect(screen.getByText('1')).toBeInTheDocument()
    })
  })

  describe('Combinations', () => {
    it('should handle size + border combination', () => {
      render(
        <Avatar
          src="/test.jpg"
          size="xl"
          bordered
          alt="Combo"
        />
      )
      const container = screen.getByAltText('Combo').parentElement
      expect(container).toHaveClass('h-16', 'w-16', 'ring-2')
    })

    it('should handle size + border + className', () => {
      render(
        <Avatar
          src="/test.jpg"
          size="lg"
          bordered
          className="mt-4"
          alt="Full Combo"
        />
      )
      const container = screen.getByAltText('Full Combo').parentElement
      expect(container).toHaveClass('h-12', 'w-12', 'ring-2', 'mt-4')
    })

    it('should handle fallback + size + border', () => {
      render(
        <Avatar
          fallback="JD"
          size="md"
          bordered
          alt="Fallback Combo"
        />
      )
      const container = screen.getByText('JD').parentElement
      expect(container).toHaveClass('h-10', 'w-10', 'ring-2')
    })

    it('should handle custom colors + size + border', () => {
      render(
        <Avatar
          fallback="JD"
          fallbackBgColor="bg-blue-500"
          fallbackTextColor="text-white"
          size="lg"
          bordered
          alt="Color Combo"
        />
      )
      const text = screen.getByText('JD')
      const container = text.parentElement
      expect(text).toHaveClass('bg-blue-500', 'text-white')
      expect(container).toHaveClass('h-12', 'w-12', 'ring-2')
    })
  })

  describe('DOM Structure', () => {
    it('should render as div wrapper', () => {
      render(<Avatar src="/test.jpg" alt="Wrapper" />)
      const container = screen.getByAltText('Wrapper').parentElement
      expect(container?.tagName).toBe('DIV')
    })

    it('should render img tag when src is provided', () => {
      render(<Avatar src="/test.jpg" alt="Img Tag" />)
      const img = screen.getByAltText('Img Tag')
      expect(img.tagName).toBe('IMG')
    })

    it('should render span tag for fallback', () => {
      render(<Avatar fallback="JD" alt="Span Tag" />)
      const span = screen.getByText('JD')
      expect(span.tagName).toBe('SPAN')
    })
  })
})
