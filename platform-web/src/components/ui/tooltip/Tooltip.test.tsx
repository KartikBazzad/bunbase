import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithTheme'
import userEvent from '@testing-library/user-event'
import { Tooltip } from './Tooltip'

describe('Tooltip', () => {
  it('should render trigger', () => {
    render(
      <Tooltip content="Tooltip content">
        <button>Trigger</button>
      </Tooltip>
    )
    expect(screen.getByText('Trigger')).toBeInTheDocument()
  })

  it('should show tooltip on hover', async () => {
    const user = userEvent.setup()
    render(
      <Tooltip content="Tooltip content">
        <button>Trigger</button>
      </Tooltip>
    )
    
    const trigger = screen.getByText('Trigger')
    await user.hover(trigger)
    
    // Wait for tooltip to appear (delay is 200ms by default)
    await new Promise(resolve => setTimeout(resolve, 250))
    
    expect(screen.queryByText('Tooltip content')).toBeInTheDocument()
  })

  it('should hide tooltip on mouse leave', async () => {
    const user = userEvent.setup()
    render(
      <Tooltip content="Tooltip content">
        <button>Trigger</button>
      </Tooltip>
    )
    
    const trigger = screen.getByText('Trigger')
    await user.hover(trigger)
    await user.unhover(trigger)
    expect(screen.queryByText('Tooltip content')).not.toBeInTheDocument()
  })

  it('should show tooltip immediately when noDelay', async () => {
    const user = userEvent.setup()
    render(
      <Tooltip content="Tooltip content" noDelay>
        <button>Trigger</button>
      </Tooltip>
    )
    
    const trigger = screen.getByText('Trigger')
    await user.hover(trigger)
    expect(screen.getByText('Tooltip content')).toBeInTheDocument()
  })

  it('should have proper role', () => {
    render(
      <Tooltip content="Tooltip content">
        <button>Trigger</button>
      </Tooltip>
    )
    const tooltip = screen.getByRole('tooltip')
    expect(tooltip).toBeInTheDocument()
  })
})
