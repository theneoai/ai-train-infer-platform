import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const spinnerVariants = cva('inline-block animate-spin rounded-full border-2 border-current', {
  variants: {
    size: {
      sm: 'h-4 w-4 border-2',
      default: 'h-6 w-6 border-2',
      lg: 'h-8 w-8 border-3',
      xl: 'h-12 w-12 border-4',
    },
  },
  defaultVariants: {
    size: 'default',
  },
})

export interface SpinnerProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof spinnerVariants> {
  asChild?: boolean
}

const Spinner = React.forwardRef<HTMLDivElement, SpinnerProps>(
  ({ className, size, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(spinnerVariants({ size }), 'border-t-transparent', className)}
        {...props}
      />
    )
  }
)
Spinner.displayName = 'Spinner'

export { Spinner, spinnerVariants }
