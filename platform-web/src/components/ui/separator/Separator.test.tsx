import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test-utils/renderWithTheme'
import { Separator } from './Separator'

describe('Separator', () => {
  describe('Rendering', () => {
    it('should render horizontal separator by default', () => {
      render(<Separator />)
      const separator = screen.getByRole('separator')
      expect(separator).toBeInTheDocument()
      expect(separator).toHaveAttribute('aria-orientation', 'horizontal')
      expect(separator).toHaveClass('border-t')
    })

    it('should render vertical separator', () => {
      render(<Separator orientation="vertical" className="h-8" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveAttribute('aria-orientation', 'vertical')
      expect(separator).toHaveClass('border-l', 'h-8')
    })

    it('should render with label', () => {
      render(<Separator label="OR" />)
      expect(screen.getByText('OR')).toBeInTheDocument()
    })

    it('should render without label', () => {
      render(<Separator />)
      const textElements = screen.queryAllByRole('text')
      expect(textElements.length).toBe(0)
    })

    it('should apply custom className', () => {
      render(<Separator className="my-4" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('my-4')
    })

    it('should merge custom className with base classes', () => {
      render(<Separator className="my-4 border-red-500" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('my-4', 'border-t', 'border-red-500')
    })
  })

  describe('Label Display', () => {
    it('should display label text', () => {
      render(<Separator label="Section Title" />)
      expect(screen.getByText('Section Title')).toBeInTheDocument()
    })

    it('should render label with proper styling', () => {
      render(<Separator label="Styled Label" />)
      const label = screen.getByText('Styled Label')
      expect(label).toHaveClass('text-sm', 'font-medium', 'text-gray-500')
    })

    it('should render border lines on both sides of label for horizontal', () => {
      render(<Separator label="Label" />)
      const label = screen.getByText('Label')
      const separator = label.parentElement
      expect(separator).toHaveClass('flex', 'items-center', 'gap-4')
    })

    it('should handle long label text', () => {
      const longLabel = 'This is a very long separator label text'
      render(<Separator label={longLabel} />)
      expect(screen.getByText(longLabel)).toBeInTheDocument()
    })

    it('should handle special characters in label', () => {
      render(<Separator label="Label & < > " '" />)
      expect(screen.getByText(/Label &/)).toBeInTheDocument()
    })

    it('should handle emoji in label', () => {
      render(<Separator label="🎉 Celebration" />)
      expect(screen.getByText('🎉 Celebration')).toBeInTheDocument()
    })
  })

  describe('Orientation', () => {
    it('should apply horizontal orientation classes', () => {
      render(<Separator orientation="horizontal" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('w-full', 'border-t')
    })

    it('should apply vertical orientation classes', () => {
      render(<Separator orientation="vertical" className="h-8" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('h-full', 'border-l')
    })

    it('should render horizontal with label', () => {
      render(<Separator orientation="horizontal" label="Label" />)
      const label = screen.getByText('Label')
      expect(label.parentElement).toHaveClass('flex', 'items-center')
    })

    it('should render vertical with label', () => {
      render(<Separator orientation="vertical" label="Label" className="h-8" />)
      const label = screen.getByText('Label')
      expect(label.parentElement).toHaveClass('flex', 'flex-col', 'items-center')
    })
  })

  describe('Accessibility', () => {
    it('should have role="separator" by default', () => {
      render(<Separator />)
      expect(screen.getByRole('separator')).toBeInTheDocument()
    })

    it('should have aria-orientation="horizontal" by default', () => {
      render(<Separator />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveAttribute('aria-orientation', 'horizontal')
    })

    it('should have aria-orientation="vertical" for vertical', () => {
      render(<Separator orientation="vertical" className="h-8" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveAttribute('aria-orientation', 'vertical')
    })

    it('should be marked as decorative by default', () => {
      render(<Separator />)
      expect(screen.getByRole('separator')).toBeInTheDocument()
    })
  })

  describe('Visual Styles', () => {
    it('should have proper border color', () => {
      render(<Separator />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('border-gray-200')
    })

    it('should be full width for horizontal', () => {
      render(<Separator />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('w-full')
    })

    it('should be full height for vertical', () => {
      render(<Separator orientation="vertical" className="h-8" />)
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('h-full')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty label', () => {
      render(<Separator label="" />)
      const separator = screen.getByRole('separator')
      expect(separator).toBeInTheDocument()
    })

    it('should handle whitespace label', () => {
      render(<Separator label="   " />)
      const separator = screen.getByRole('separator')
      expect(separator).toBeInTheDocument()
    })

    it('should handle label with only spaces and content', () => {
      render(<Separator label="  Label  " />)
      expect(screen.getByText('  Label  ')).toBeInTheDocument()
    })

    it('should handle numbers in label', () => {
      render(<Separator label="123" />)
      expect(screen.getByText('123')).toBeInTheDocument()
    })

    it('should handle null label', () => {
      render(<Separator label={null as any} />)
      const separator = screen.getByRole('separator')
      expect(separator).toBeInTheDocument()
    })
  })

  describe('Combinations', () => {
    it('should handle orientation + className', () => {
      render(
        <Separator
          orientation="vertical"
          className="h-8 my-4 border-red-500"
        />
      )
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('h-full', 'border-l', 'my-4', 'border-red-500')
    })

    it('should handle label + orientation + className', () => {
      render(
        <Separator
          label="Test"
          orientation="horizontal"
          className="my-6"
        />
      )
      expect(screen.getByText('Test')).toBeInTheDocument()
      const separator = screen.getByRole('separator')
      expect(separator).toHaveClass('my-6')
    })
  })
})
