import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithTheme'
import userEvent from '@testing-library/user-event'
import { Modal } from './Modal'

describe('Modal', () => {
  it('should not render when closed', () => {
    render(<Modal open={false} onClose={vi.fn()}>Content</Modal>)
    expect(screen.queryByText('Content')).not.toBeInTheDocument()
  })

  it('should render when open', () => {
    render(<Modal open={true} onClose={vi.fn()}>Content</Modal>)
    expect(screen.getByText('Content')).toBeInTheDocument()
  })

  it('should render title', () => {
    render(<Modal open={true} onClose={vi.fn()} title="Test Title">Content</Modal>)
    expect(screen.getByText('Test Title')).toBeInTheDocument()
  })

  it('should render footer', () => {
    render(
      <Modal open={true} onClose={vi.fn()} footer={<button>Footer</button>}>
        Content
      </Modal>
    )
    expect(screen.getByText('Footer')).toBeInTheDocument()
  })

  it('should call onClose on backdrop click', () => {
    const handleClose = vi.fn()
    render(<Modal open={true} onClose={handleClose}>Content</Modal>)
    
    const backdrop = screen.getByText('Content').parentElement?.parentElement?.firstElementChild
    fireEvent.click(backdrop!)
    expect(handleClose).toHaveBeenCalled()
  })

  it('should not call onClose when closeOnBackdropClick is false', () => {
    const handleClose = vi.fn()
    render(
      <Modal open={true} onClose={handleClose} closeOnBackdropClick={false}>
        Content
      </Modal>
    )
    
    const backdrop = screen.getByText('Content').parentElement?.parentElement?.firstElementChild
    fireEvent.click(backdrop!)
    expect(handleClose).not.toHaveBeenCalled()
  })

  it('should prevent body scroll when open', () => {
    render(<Modal open={true} onClose={vi.fn()}>Content</Modal>)
    expect(document.body.style.overflow).toBe('hidden')
  })

  it('should have proper role', () => {
    render(<Modal open={true} onClose={vi.fn()}>Content</Modal>)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
  })
})
