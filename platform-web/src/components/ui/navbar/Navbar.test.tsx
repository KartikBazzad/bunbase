import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithTheme'
import userEvent from '@testing-library/user-event'
import { Navbar } from './Navbar'

describe('Navbar', () => {
  it('should render navbar', () => {
    render(<Navbar />)
    const navbar = screen.getByRole('navigation')
    expect(navbar).toBeInTheDocument()
  })

  it('should render logo', () => {
    render(<Navbar logo={<span data-testid="logo">Logo</span>} />)
    expect(screen.getByTestId('logo')).toBeInTheDocument()
  })

  it('should render children', () => {
    render(<Navbar><nav>Center</nav></Navbar>)
    expect(screen.getByText('Center')).toBeInTheDocument()
  })

  it('should render actions', () => {
    render(<Navbar actions={<button data-testid="action">Action</button>} />)
    expect(screen.getByTestId('action')).toBeInTheDocument()
  })

  it('should apply fixed positioning', () => {
    render(<Navbar fixed />)
    const navbar = screen.getByRole('navigation')
    expect(navbar).toHaveClass('fixed', 'top-0', 'left-0', 'right-0', 'z-50')
  })

  it('should apply transparent background', () => {
    render(<Navbar transparent />)
    const navbar = screen.getByRole('navigation')
    expect(navbar).toHaveClass('border-transparent', 'bg-transparent')
  })
})
