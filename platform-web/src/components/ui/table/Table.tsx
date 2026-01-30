import React, { useState } from 'react'
import { cn } from '@/lib/cn'
import { Checkbox } from '../checkbox'
import { Button } from '../button'

/**
 * Table column definition
 */
export interface TableColumn<T> {
  key: string
  header: string
  width?: string
  sortable?: boolean
  render?: (value: any, row: T, index: number) => React.ReactNode
}

/**
 * Table component props
 */
export interface TableProps<T> extends Omit<React.HTMLAttributes<HTMLTableElement>, 'children'> {
  /**
   * Table data
   */
  data: T[]
  
  /**
   * Column definitions
   */
  columns: TableColumn<T>[]
  
  /**
   * Row key extractor
   */
  rowKey: keyof T | ((row: T) => string)
  
  /**
   * Enable selection
   * @default false
   */
  selectable?: boolean
  
  /**
   * Selected row keys
   */
  selectedKeys?: string[]
  
  /**
   * On selection change
   */
  onSelectionChange?: (keys: string[]) => void
  
  /**
   * Enable sorting
   * @default false
   */
  sortable?: boolean
  
  /**
   * On sort change
   */
  onSortChange?: (column: string, direction: 'asc' | 'desc' | null) => void
  
  /**
   * Loading state
   * @default false
   */
  loading?: boolean
  
  /**
   * Empty state message
   */
  emptyMessage?: string
  
  /**
   * Row click handler
   */
  onRowClick?: (row: T, index: number) => void
}

/**
 * Table component for displaying data
 */
function Table<T extends Record<string, any>>({
  data,
  columns,
  rowKey,
  selectable = false,
  selectedKeys = [],
  onSelectionChange,
  sortable = false,
  onSortChange,
  loading = false,
  emptyMessage = 'No data available',
  onRowClick,
  className,
  ...props
}: TableProps<T>) {
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc' | null>(null)

  const getRowKey = (row: T, index: number): string => {
    if (typeof rowKey === 'function') {
      return rowKey(row)
    }
    return String(row[rowKey])
  }

  const isRowSelected = (row: T, index: number) => {
    return selectedKeys.includes(getRowKey(row, index))
  }

  const toggleRowSelection = (row: T, index: number) => {
    const key = getRowKey(row, index)
    const newSelection = isRowSelected(row, index)
      ? selectedKeys.filter(k => k !== key)
      : [...selectedKeys, key]
    onSelectionChange?.(newSelection)
  }

  const handleSort = (column: TableColumn<T>) => {
    if (!column.sortable) return
    
    const newDirection = sortColumn === column.key
      ? (sortDirection === 'asc' ? 'desc' : null)
      : 'asc'
    
    setSortColumn(newDirection ? column.key : null)
    setSortDirection(newDirection)
    onSortChange?.(column.key, newDirection)
  }

  const sortedData = React.useMemo(() => {
    if (!sortColumn || !sortDirection) return data
    
    return [...data].sort((a, b) => {
      const aVal = a[sortColumn]
      const bVal = b[sortColumn]
      
      if (aVal === bVal) return 0
      
      const comparison = aVal > bVal ? 1 : -1
      return sortDirection === 'asc' ? comparison : -comparison
    })
  }, [data, sortColumn, sortDirection])

  return (
    <div className="w-full overflow-x-auto">
      <table
        className={cn(
          'w-full border-collapse border-separate border-spacing-0',
          className
        )}
        {...props}
      >
        <thead>
          <tr className="border-b border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-800">
            {selectable && (
              <th className="px-4 py-3 text-left">
                <Checkbox
                  checked={selectedKeys.length === data.length && data.length > 0}
                  onChange={(checked) => {
                    onSelectionChange?.(checked ? data.map((_, i) => getRowKey(_, i)) : [])
                  }}
                />
              </th>
            )}
            {columns.map((column) => (
              <th
                key={column.key}
                className={cn(
                  'px-4 py-3 text-left text-sm font-semibold text-gray-900 dark:text-gray-100',
                  column.sortable && 'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700',
                  column.width
                )}
                onClick={() => handleSort(column)}
              >
                <div className="flex items-center gap-2">
                  {column.header}
                  {column.sortable && sortColumn === column.key && (
                    <span className="text-gray-500">
                      {sortDirection === 'asc' ? '↑' : '↓'}
                    </span>
                  )}
                </div>
              </th>
            ))}
          </tr>
        </thead>

        <tbody>
          {loading ? (
            <tr>
              <td colSpan={columns.length + (selectable ? 1 : 0)} className="px-4 py-8 text-center">
                <div className="inline-block animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 h-6 w-6" />
              </td>
            </tr>
          ) : sortedData.length === 0 ? (
            <tr>
              <td colSpan={columns.length + (selectable ? 1 : 0)} className="px-4 py-8 text-center text-gray-500">
                {emptyMessage}
              </td>
            </tr>
          ) : (
            sortedData.map((row, index) => (
              <tr
                key={getRowKey(row, index)}
                className={cn(
                  'border-b border-gray-200 hover:bg-gray-50 dark:border-gray-800 dark:hover:bg-gray-800/50',
                  'transition-colors',
                  onRowClick && 'cursor-pointer'
                )}
                onClick={() => onRowClick?.(row, index)}
              >
                {selectable && (
                  <td className="px-4 py-3">
                    <Checkbox
                      checked={isRowSelected(row, index)}
                      onChange={() => toggleRowSelection(row, index)}
                    />
                  </td>
                )}
                {columns.map((column) => (
                  <td key={column.key} className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                    {column.render
                      ? column.render(row[column.key], row, index)
                      : String(row[column.key] ?? '-')}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

Table.displayName = 'Table'

export { Table }
