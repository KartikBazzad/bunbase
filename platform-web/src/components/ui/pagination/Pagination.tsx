import React from 'react'
import { cn } from '@/lib/cn'
import { Button } from '../button'

/**
 * Pagination component props
 */
export interface PaginationProps extends React.HTMLAttributes<HTMLDivElement> {
  /**
   * Current page (1-indexed)
   */
  page: number
  
  /**
   * Total number of pages
   */
  totalPages: number
  
  /**
   * On page change handler
   */
  onPageChange: (page: number) => void
  
  /**
   * Show page numbers
   * @default true
   */
  showPageNumbers?: boolean
  
  /**
   * Maximum page numbers to show
   * @default 5
   */
  maxVisiblePages?: number
}

/**
 * Pagination component for navigating through pages
 */
const Pagination = React.forwardRef<HTMLDivElement, PaginationProps>(
  ({
    page,
    totalPages,
    onPageChange,
    showPageNumbers = true,
    maxVisiblePages = 5,
    className,
  }, ref) => {
    const getPageNumbers = () => {
      if (!showPageNumbers) return []
      
      const pages = []
      const half = Math.floor(maxVisiblePages / 2)
      
      let start = Math.max(1, page - half)
      let end = Math.min(totalPages, page + half)
      
      if (end - start < maxVisiblePages - 1) {
        start = Math.max(1, end - maxVisiblePages + 1)
      }
      
      for (let i = start; i <= end; i++) {
        pages.push(i)
      }
      
      return pages
    }

    const pageNumbers = getPageNumbers()

    return (
      <div
        ref={ref}
        className={cn('flex items-center gap-2', className)}
        role="navigation"
        aria-label="Pagination"
      >
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onPageChange(page - 1)}
          disabled={page === 1}
          aria-label="Previous page"
        >
          ←
        </Button>

        {pageNumbers.map((pageNum) => (
          <Button
            key={pageNum}
            variant={pageNum === page ? 'primary' : 'ghost'}
            size="sm"
            onClick={() => onPageChange(pageNum)}
            aria-label={`Page ${pageNum}`}
            aria-current={pageNum === page ? 'page' : undefined}
          >
            {pageNum}
          </Button>
        ))}

        <Button
          variant="ghost"
          size="sm"
          onClick={() => onPageChange(page + 1)}
          disabled={page === totalPages}
          aria-label="Next page"
        >
          →
        </Button>
      </div>
    )
  }
)

Pagination.displayName = 'Pagination'

export { Pagination }
