import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test-utils/renderWithTheme'
import { Skeleton } from './Skeleton'

describe('Skeleton', () => {
  describe('Rendering', () => {
    it('should render rectangular variant by default', () => {
      render(<Skeleton />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toBeInTheDocument()
      expect(skeleton).toHaveClass('rounded-md')
    })

    it('should render rectangular variant explicitly', () => {
      render(<Skeleton variant="rectangular" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-md')
    })

    it('should render circular variant', () => {
      render(<Skeleton variant="circular" width={40} height={40} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-full')
    })

    it('should render text variant', () => {
      render(<Skeleton variant="text" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('h-4', 'w-full')
    })

    it('should render with custom width', () => {
      render(<Skeleton width={200} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '200px' })
    })

    it('should render with custom height', () => {
      render(<Skeleton height={20} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ height: '20px' })
    })

    it('should render with string width', () => {
      render(<Skeleton width="100%" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '100%' })
    })

    it('should render with string height', () => {
      render(<Skeleton height="100px" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ height: '100px' })
    })

    it('should apply custom className', () => {
      render(<Skeleton className="custom-class" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('custom-class')
    })

    it('should merge custom className with base classes', () => {
      render(
        <Skeleton
          variant="circular"
          width={40}
          height={40}
          className="mt-4"
        />
      )
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-full', 'mt-4')
    })
  })

  describe('Animation', () => {
    it('should have animation by default', () => {
      render(<Skeleton />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('animate-pulse')
    })

    it('should have animated enabled explicitly', () => {
      render(<Skeleton animated={true} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('animate-pulse')
    })

    it('should have no animation when disabled', () => {
      render(<Skeleton animated={false} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).not.toHaveClass('animate-pulse')
    })
  })

  describe('Text Variant', () => {
    it('should render single line of text by default', () => {
      render(<Skeleton variant="text" />)
      const skeletons = screen.getAllByRole('status')
      expect(skeletons).toHaveLength(1)
    })

    it('should render multiple lines of text', () => {
      render(<Skeleton variant="text" lines={3} />)
      const skeletons = screen.getAllByRole('status')
      expect(skeletons).toHaveLength(3)
    })

    it('should render last line shorter for text', () => {
      render(<Skeleton variant="text" lines={3} />)
      const skeletons = screen.getAllByRole('status')
      const lastSkeleton = skeletons[skeletons.length - 1]
      expect(lastSkeleton).toHaveStyle({ width: '60%' })
    })

    it('should have proper spacing for multiple lines', () => {
      render(<Skeleton variant="text" lines={3} />)
      const container = screen.getAllByRole('status')[0].parentElement
      expect(container).toHaveClass('space-y-2')
    })

    it('should apply width to text variant', () => {
      render(<Skeleton variant="text" width={300} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '300px' })
    })
  })

  describe('Circular Variant', () => {
    it('should have rounded-full shape', () => {
      render(<Skeleton variant="circular" width={40} height={40} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-full')
    })

    it('should render with equal dimensions for circle', () => {
      render(<Skeleton variant="circular" width={50} height={50} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '50px', height: '50px' })
    })

    it('should render as avatar size', () => {
      render(<Skeleton variant="circular" width={40} height={40} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '40px', height: '40px' })
    })
  })

  describe('Rectangular Variant', () => {
    it('should have rounded-md corners', () => {
      render(<Skeleton variant="rectangular" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-md')
    })

    it('should render card-like dimensions', () => {
      render(<Skeleton width={300} height={200} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '300px', height: '200px' })
    })

    it('should render image-like dimensions', () => {
      render(<Skeleton width={100} height={100} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '100px', height: '100px' })
    })
  })

  describe('Accessibility', () => {
    it('should have role="status"', () => {
      render(<Skeleton />)
      expect(screen.getByRole('status')).toBeInTheDocument()
    })

    it('should have aria-busy="true"', () => {
      render(<Skeleton />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveAttribute('aria-busy', 'true')
    })

    it('should have multiple aria-busy for text lines', () => {
      render(<Skeleton variant="text" lines={3} />)
      const skeletons = screen.getAllByRole('status')
      skeletons.forEach((skeleton) => {
        expect(skeleton).toHaveAttribute('aria-busy', 'true')
      })
    })
  })

  describe('Visual Styles', () => {
    it('should have proper background color', () => {
      render(<Skeleton />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('bg-gray-200')
    })

    it('should have proper border radius for rectangular', () => {
      render(<Skeleton variant="rectangular" />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-md')
    })

    it('should have proper border radius for circular', () => {
      render(<Skeleton variant="circular" width={40} height={40} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-full')
    })

    it('should have inline display by default', () => {
      render(<Skeleton />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).not.toHaveClass('block')
    })
  })

  describe('Edge Cases', () => {
    it('should handle zero width', () => {
      render(<Skeleton width={0} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '0px' })
    })

    it('should handle zero height', () => {
      render(<Skeleton height={0} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ height: '0px' })
    })

    it('should handle large width', () => {
      render(<Skeleton width={10000} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '10000px' })
    })

    it('should handle large height', () => {
      render(<Skeleton height={10000} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ height: '10000px' })
    })

    it('should handle zero lines', () => {
      render(<Skeleton variant="text" lines={0} />)
      const container = screen.getByRole('status')
      expect(container).toBeEmptyDOMElement()
    })

    it('should handle single line', () => {
      render(<Skeleton variant="text" lines={1} />)
      const skeletons = screen.getAllByRole('status')
      expect(skeletons).toHaveLength(1)
    })

    it('should handle many lines', () => {
      render(<Skeleton variant="text" lines={20} />)
      const skeletons = screen.getAllByRole('status')
      expect(skeletons).toHaveLength(20)
    })
  })

  describe('Combinations', () => {
    it('should handle variant + width + height', () => {
      render(
        <Skeleton variant="rectangular" width={200} height={100} />
      )
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-md')
      expect(skeleton).toHaveStyle({ width: '200px', height: '100px' })
    })

    it('should handle variant + animation', () => {
      render(<Skeleton variant="circular" width={40} height={40} animated={false} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-full')
      expect(skeleton).not.toHaveClass('animate-pulse')
    })

    it('should handle variant + className + animation', () => {
      render(
        <Skeleton
          variant="text"
          className="mt-4"
          animated={true}
        />
      )
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('h-4', 'w-full', 'mt-4', 'animate-pulse')
    })

    it('should handle text variant + lines + animation', () => {
      render(<Skeleton variant="text" lines={3} animated={true} />)
      const skeletons = screen.getAllByRole('status')
      expect(skeletons).toHaveLength(3)
      skeletons.forEach((skeleton) => {
        expect(skeleton).toHaveClass('animate-pulse')
      })
    })

    it('should handle custom style with width/height', () => {
      render(
        <Skeleton
          width={200}
          height={100}
          style={{ margin: '10px' }}
        />
      )
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({
        width: '200px',
        height: '100px',
        margin: '10px',
      })
    })
  })

  describe('Common Use Cases', () => {
    it('should render avatar skeleton', () => {
      render(<Skeleton variant="circular" width={40} height={40} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-full')
      expect(skeleton).toHaveStyle({ width: '40px', height: '40px' })
    })

    it('should render card skeleton', () => {
      render(<Skeleton variant="rectangular" width={300} height={200} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-md')
      expect(skeleton).toHaveStyle({ width: '300px', height: '200px' })
    })

    it('should render paragraph skeleton', () => {
      render(<Skeleton variant="text" lines={4} />)
      const skeletons = screen.getAllByRole('status')
      expect(skeletons).toHaveLength(4)
    })

    it('should render title skeleton', () => {
      render(<Skeleton variant="text" width={200} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveStyle({ width: '200px' })
    })

    it('should render image skeleton', () => {
      render(<Skeleton variant="rectangular" width={400} height={300} />)
      const skeleton = screen.getByRole('status')
      expect(skeleton).toHaveClass('rounded-md')
      expect(skeleton).toHaveStyle({ width: '400px', height: '300px' })
    })
  })
})
