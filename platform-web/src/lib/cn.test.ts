import { describe, it, expect } from 'vitest'
import { cn } from './cn'

describe('cn utility', () => {
  describe('Basic merging', () => {
    it('should merge simple class names', () => {
      expect(cn('px-4', 'py-2')).toBe('px-4 py-2')
    })

    it('should handle single class name', () => {
      expect(cn('px-4')).toBe('px-4')
    })

    it('should handle empty inputs', () => {
      expect(cn()).toBe('')
    })

    it('should handle null/undefined', () => {
      expect(cn(null, undefined, 'px-4')).toBe('px-4')
    })
  })

  describe('Conditional classes', () => {
    it('should include class when condition is true', () => {
      expect(cn('px-4', true && 'py-2')).toBe('px-4 py-2')
    })

    it('should exclude class when condition is false', () => {
      expect(cn('px-4', false && 'py-2')).toBe('px-4')
    })

    it('should handle object with boolean values', () => {
      expect(cn({ 'px-4': true, 'py-2': false })).toBe('px-4')
    })

    it('should handle array of conditional classes', () => {
      const isActive = true
      expect(cn('base', isActive && 'active', !isActive && 'inactive')).toBe('base active')
    })
  })

  describe('Tailwind conflict resolution', () => {
    it('should resolve padding conflicts', () => {
      expect(cn('px-4', 'px-6')).toBe('px-6')
    })

    it('should resolve color conflicts', () => {
      expect(cn('text-red-500', 'text-blue-500')).toBe('text-blue-500')
    })

    it('should resolve multiple conflicts', () => {
      expect(cn('px-4 py-2 bg-white', 'px-6 py-4 bg-gray-100')).toBe('px-6 py-4 bg-gray-100')
    })

    it('should keep non-conflicting classes', () => {
      expect(cn('px-4', 'py-2', 'text-red-500')).toBe('px-4 py-2 text-red-500')
    })

    it('should handle responsive class conflicts', () => {
      expect(cn('px-4', 'md:px-6', 'lg:px-8')).toBe('px-4 md:px-6 lg:px-8')
    })

    it('should resolve arbitrary value conflicts', () => {
      expect(cn('p-[10px]', 'p-[20px]')).toBe('p-[20px]')
    })
  })

  describe('Complex scenarios', () => {
    it('should handle complex mixed inputs', () => {
      const isActive = true
      const size = 'lg'
      expect(cn(
        'base-class',
        isActive && 'active-class',
        size === 'lg' && 'lg:text-xl',
        { 'text-center': true },
        ['flex', 'items-center']
      )).toBe('base-class active-class lg:text-xl text-center flex items-center')
    })

    it('should handle template literals', () => {
      const prefix = 'btn'
      const variant = 'primary'
      expect(cn(`${prefix}-${variant}`, 'mt-4')).toBe('btn-primary mt-4')
    })

    it('should handle deeply nested conditions', () => {
      const a = true
      const b = false
      const c = true
      expect(cn(
        a && 'a',
        b && 'b',
        c && 'c',
        a && b && 'ab',
        b && c && 'bc',
        a && c && 'ac'
      )).toBe('a c ac')
    })
  })

  describe('Edge cases', () => {
    it('should handle duplicate classes', () => {
      expect(cn('px-4', 'px-4', 'py-2')).toBe('px-4 py-2')
    })

    it('should handle whitespace', () => {
      expect(cn('  px-4  ', '  py-2  ')).toBe('px-4 py-2')
    })

    it('should handle numbers in array', () => {
      expect(cn('px-4', 0, 'py-2', 1)).toBe('px-4 py-2 1')
    })

    it('should handle zero as class name', () => {
      expect(cn('px-4', 0)).toBe('px-4 0')
    })

    it('should handle falsy values except 0', () => {
      expect(cn('px-4', false, null, undefined, '')).toBe('px-4')
    })
  })

  describe('Real-world scenarios', () => {
    it('should handle button variants', () => {
      const variant = 'primary'
      const size = 'lg'
      const disabled = false

      expect(
        cn(
          'rounded font-medium transition-colors',
          'px-4 py-2',
          variant === 'primary' && 'bg-blue-500 text-white hover:bg-blue-600',
          variant === 'secondary' && 'bg-gray-200 text-gray-900 hover:bg-gray-300',
          size === 'sm' && 'text-sm px-3 py-1.5',
          size === 'lg' && 'text-lg px-6 py-3',
          disabled && 'opacity-50 cursor-not-allowed'
        )
      ).toBe('rounded font-medium transition-colors px-4 py-2 bg-blue-500 text-white hover:bg-blue-600 text-lg px-6 py-3')
    })

    it('should handle card variants', () => {
      const elevated = true
      const bordered = false

      expect(
        cn(
          'rounded-lg',
          'bg-white',
          elevated && 'shadow-lg',
          bordered && 'border border-gray-200',
          'p-6'
        )
      ).toBe('rounded-lg bg-white shadow-lg p-6')
    })

    it('should handle input states', () => {
      const error = true
      const focused = false

      expect(
        cn(
          'w-full px-3 py-2 border rounded-md',
          error && 'border-red-500 text-red-900 focus:ring-red-500',
          !error && 'border-gray-300 focus:ring-blue-500',
          focused && 'ring-2'
        )
      ).toBe('w-full px-3 py-2 border rounded-md border-red-500 text-red-900 focus:ring-red-500')
    })
  })
})
