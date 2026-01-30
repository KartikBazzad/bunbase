import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test-utils/renderWithTheme'
import { Card, CardHeader, CardBody, CardFooter } from './Card'

describe('Card', () => {
  describe('Rendering', () => {
    it('should render elevated variant by default', () => {
      render(<Card>Card Content</Card>)
      const card = screen.getByText('Card Content').parentElement
      expect(card).toBeInTheDocument()
      expect(card).toHaveClass('shadow-soft')
    })

    it('should render elevated variant explicitly', () => {
      render(<Card variant="elevated">Elevated Card</Card>)
      const card = screen.getByText('Elevated Card').parentElement
      expect(card).toHaveClass('shadow-soft')
    })

    it('should render flat variant', () => {
      render(<Card variant="flat">Flat Card</Card>)
      const card = screen.getByText('Flat Card').parentElement
      expect(card).toHaveClass('border-none', 'shadow-none')
    })

    it('should render outlined variant', () => {
      render(<Card variant="outlined">Outlined Card</Card>)
      const card = screen.getByText('Outlined Card').parentElement
      expect(card).toHaveClass('border-2', 'border-gray-200', 'shadow-none')
    })

    it('should render with header only', () => {
      render(
        <Card>
          <CardHeader>Header Content</CardHeader>
        </Card>
      )
      expect(screen.getByText('Header Content')).toBeInTheDocument()
      expect(screen.getByRole('article')).toBeInTheDocument()
    })

    it('should render with body only', () => {
      render(
        <Card>
          <CardBody>Body Content</CardBody>
        </Card>
      )
      expect(screen.getByText('Body Content')).toBeInTheDocument()
    })

    it('should render with footer only', () => {
      render(
        <Card>
          <CardFooter>Footer Content</CardFooter>
        </Card>
      )
      expect(screen.getByText('Footer Content')).toBeInTheDocument()
    })

    it('should render with all sections', () => {
      render(
        <Card>
          <CardHeader>Header</CardHeader>
          <CardBody>Body</CardBody>
          <CardFooter>Footer</CardFooter>
        </Card>
      )
      expect(screen.getByText('Header')).toBeInTheDocument()
      expect(screen.getByText('Body')).toBeInTheDocument()
      expect(screen.getByText('Footer')).toBeInTheDocument()
    })

    it('should render with multiple children in body', () => {
      render(
        <Card>
          <CardBody>
            <p>Paragraph 1</p>
            <p>Paragraph 2</p>
            <button>Action</button>
          </CardBody>
        </Card>
      )
      expect(screen.getByText('Paragraph 1')).toBeInTheDocument()
      expect(screen.getByText('Paragraph 2')).toBeInTheDocument()
      expect(screen.getByRole('button')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(<Card className="custom-class">Custom Card</Card>)
      const card = screen.getByText('Custom Card').parentElement
      expect(card).toHaveClass('custom-class')
    })

    it('should merge custom className with base classes', () => {
      render(
        <Card variant="elevated" className="mt-4 p-6">
          Combined
        </Card>
      )
      const card = screen.getByText('Combined').parentElement
      expect(card).toHaveClass('card', 'shadow-soft', 'mt-4', 'p-6')
    })
  })

  describe('CardHeader', () => {
    it('should render header content', () => {
      render(
        <Card>
          <CardHeader>Header Title</CardHeader>
        </Card>
      )
      const header = screen.getByText('Header Title')
      expect(header).toBeInTheDocument()
      expect(header.parentElement).toHaveClass('card-header')
    })

    it('should render with heading element', () => {
      render(
        <Card>
          <CardHeader>
            <h2>Card Title</h2>
          </CardHeader>
        </Card>
      )
      expect(screen.getByRole('heading', { level: 2 })).toBeInTheDocument()
    })

    it('should render with multiple elements', () => {
      render(
        <Card>
          <CardHeader>
            <h2>Title</h2>
            <p>Subtitle</p>
            <button>Action</button>
          </CardHeader>
        </Card>
      )
      expect(screen.getByText('Title')).toBeInTheDocument()
      expect(screen.getByText('Subtitle')).toBeInTheDocument()
      expect(screen.getByRole('button')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(
        <Card>
          <CardHeader className="custom-header">Header</CardHeader>
        </Card>
      )
      const header = screen.getByText('Header').parentElement
      expect(header).toHaveClass('custom-header')
    })

    it('should pass through HTML attributes', () => {
      render(
        <Card>
          <CardHeader data-testid="card-header">Header</CardHeader>
        </Card>
      )
      expect(screen.getByTestId('card-header')).toBeInTheDocument()
    })
  })

  describe('CardBody', () => {
    it('should render body content', () => {
      render(
        <Card>
          <CardBody>Body Content</CardBody>
        </Card>
      )
      const body = screen.getByText('Body Content')
      expect(body).toBeInTheDocument()
      expect(body.parentElement).toHaveClass('card-body')
    })

    it('should render with text content', () => {
      render(
        <Card>
          <CardBody>
            <p>First paragraph</p>
            <p>Second paragraph</p>
          </CardBody>
        </Card>
      )
      expect(screen.getByText('First paragraph')).toBeInTheDocument()
      expect(screen.getByText('Second paragraph')).toBeInTheDocument()
    })

    it('should render with images', () => {
      render(
        <Card>
          <CardBody>
            <img src="/test.jpg" alt="Test image" />
          </CardBody>
        </Card>
      )
      expect(screen.getByAltText('Test image')).toBeInTheDocument()
    })

    it('should render with nested components', () => {
      render(
        <Card>
          <CardBody>
            <div className="nested">
              <span>Nested content</span>
            </div>
          </CardBody>
        </Card>
      )
      expect(screen.getByText('Nested content')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(
        <Card>
          <CardBody className="custom-body">Body</CardBody>
        </Card>
      )
      const body = screen.getByText('Body').parentElement
      expect(body).toHaveClass('custom-body')
    })

    it('should pass through HTML attributes', () => {
      render(
        <Card>
          <CardBody data-testid="card-body">Body</CardBody>
        </Card>
      )
      expect(screen.getByTestId('card-body')).toBeInTheDocument()
    })
  })

  describe('CardFooter', () => {
    it('should render footer content', () => {
      render(
        <Card>
          <CardFooter>Footer Content</CardFooter>
        </Card>
      )
      const footer = screen.getByText('Footer Content')
      expect(footer).toBeInTheDocument()
      expect(footer.parentElement).toHaveClass('card-footer')
    })

    it('should render with buttons', () => {
      render(
        <Card>
          <CardFooter>
            <button>Cancel</button>
            <button>Submit</button>
          </CardFooter>
        </Card>
      )
      const buttons = screen.getAllByRole('button')
      expect(buttons).toHaveLength(2)
      expect(screen.getByText('Cancel')).toBeInTheDocument()
      expect(screen.getByText('Submit')).toBeInTheDocument()
    })

    it('should render with links', () => {
      render(
        <Card>
          <CardFooter>
            <a href="/link">Learn more</a>
          </CardFooter>
        </Card>
      )
      expect(screen.getByText('Learn more')).toBeInTheDocument()
    })

    it('should render with metadata', () => {
      render(
        <Card>
          <CardFooter>
            <time dateTime="2024-01-01">Jan 1, 2024</time>
            <span>Author: John</span>
          </CardFooter>
        </Card>
      )
      expect(screen.getByText('Jan 1, 2024')).toBeInTheDocument()
      expect(screen.getByText('Author: John')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(
        <Card>
          <CardFooter className="custom-footer">Footer</CardFooter>
        </Card>
      )
      const footer = screen.getByText('Footer').parentElement
      expect(footer.parentElement).toHaveClass('custom-footer')
    })

    it('should pass through HTML attributes', () => {
      render(
        <Card>
          <CardFooter data-testid="card-footer">Footer</CardFooter>
        </Card>
      )
      expect(screen.getByTestId('card-footer')).toBeInTheDocument()
    })
  })

  describe('Complete Card Structure', () => {
    it('should render complete card with all sections', () => {
      render(
        <Card>
          <CardHeader>
            <h2>Card Title</h2>
            <p>Card subtitle</p>
          </CardHeader>
          <CardBody>
            <p>Main content goes here</p>
          </CardBody>
          <CardFooter>
            <button>Cancel</button>
            <button>Submit</button>
          </CardFooter>
        </Card>
      )

      expect(screen.getByRole('heading', { level: 2 })).toBeInTheDocument()
      expect(screen.getByText('Card subtitle')).toBeInTheDocument()
      expect(screen.getByText('Main content goes here')).toBeInTheDocument()
      expect(screen.getByText('Cancel')).toBeInTheDocument()
      expect(screen.getByText('Submit')).toBeInTheDocument()
    })

    it('should render with multiple body sections', () => {
      render(
        <Card>
          <CardHeader>Title</CardHeader>
          <CardBody>
            <p>First section</p>
          </CardBody>
          <CardBody>
            <p>Second section</p>
          </CardBody>
          <CardFooter>Footer</CardFooter>
        </Card>
      )
      expect(screen.getByText('First section')).toBeInTheDocument()
      expect(screen.getByText('Second section')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('should be accessible with proper heading structure', () => {
      render(
        <Card>
          <CardHeader>
            <h2>Accessible Title</h2>
          </CardHeader>
          <CardBody>Content</CardBody>
        </Card>
      )
      expect(screen.getByRole('heading', { level: 2 })).toBeInTheDocument()
    })

    it('should support ARIA attributes', () => {
      render(
        <Card role="article" aria-labelledby="card-title">
          <CardHeader>
            <h2 id="card-title">Title</h2>
          </CardHeader>
          <CardBody>Content</CardBody>
        </Card>
      )
      const card = screen.getByRole('article')
      expect(card).toHaveAttribute('aria-labelledby', 'card-title')
    })

    it('should support keyboard focus on interactive elements', () => {
      render(
        <Card>
          <CardFooter>
            <button>Focusable Button</button>
          </CardFooter>
        </Card>
      )
      const button = screen.getByRole('button')
      button.focus()
      expect(button).toHaveFocus()
    })
  })

  describe('Visual Styles', () => {
    it('should apply hover effect with card-hover class', () => {
      render(
        <Card className="card-hover cursor-pointer">
          Hoverable Card
        </Card>
      )
      const card = screen.getByText('Hoverable Card').parentElement
      expect(card).toHaveClass('card-hover')
    })

    it('should maintain border on flat variant', () => {
      render(<Card variant="flat">Flat Card</Card>)
      const card = screen.getByText('Flat Card').parentElement
      expect(card).toHaveClass('border-none', 'shadow-none')
    })

    it('should maintain border on outlined variant', () => {
      render(<Card variant="outlined">Outlined Card</Card>)
      const card = screen.getByText('Outlined Card').parentElement
      expect(card).toHaveClass('border-2', 'border-gray-200')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty content', () => {
      render(<Card></Card>)
      const card = screen.getByRole('article')
      expect(card).toBeInTheDocument()
      expect(card).toBeEmptyDOMElement()
    })

    it('should handle whitespace', () => {
      render(<Card>   </Card>)
      const card = screen.getByRole('article')
      expect(card).toHaveTextContent('   ')
    })

    it('should handle long content', () => {
      const longText = 'A'.repeat(1000)
      render(
        <Card>
          <CardBody>{longText}</CardBody>
        </Card>
      )
      expect(screen.getByText(longText)).toBeInTheDocument()
    })

    it('should handle special characters', () => {
      render(
        <Card>
          <CardBody>Special & < > " '</CardBody>
        </Card>
      )
      expect(screen.getByText(/Special &/)).toBeInTheDocument()
    })

    it('should handle emoji', () => {
      render(
        <Card>
          <CardBody>🎉 Emoji content 🚀</CardBody>
        </Card>
      )
      expect(screen.getByText('🎉 Emoji content 🚀')).toBeInTheDocument()
    })
  })

  describe('Combinations', () => {
    it('should handle variant + className combination', () => {
      render(
        <Card variant="outlined" className="w-full max-w-md">
          Combined Card
        </Card>
      )
      const card = screen.getByText('Combined Card').parentElement
      expect(card).toHaveClass('border-2', 'w-full', 'max-w-md')
    })

    it('should handle all sections with custom classes', () => {
      render(
        <Card className="shadow-lg">
          <CardHeader className="pb-2">
            <h2>Header</h2>
          </CardHeader>
          <CardBody className="pt-4">Body</CardBody>
          <CardFooter className="justify-end">Footer</CardFooter>
        </Card>
      )
      expect(screen.getByText('Header').parentElement).toHaveClass('pb-2')
      expect(screen.getByText('Body').parentElement).toHaveClass('pt-4')
      expect(screen.getByText('Footer').parentElement).toHaveClass('justify-end')
    })

    it('should pass through event handlers', () => {
      const handleClick = vi.fn()
      render(
        <Card onClick={handleClick}>
          <CardBody>Clickable</CardBody>
        </Card>
      )
      const card = screen.getByRole('article')
      card.click()
      expect(handleClick).toHaveBeenCalled()
    })
  })
})
