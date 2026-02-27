import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'

const toastVariants = cva(
  'pointer-events-auto relative flex w-full items-center justify-between space-x-4 overflow-hidden rounded-md border p-4 pr-8 shadow-lg transition-all',
  {
    variants: {
      variant: {
        default: 'border bg-background text-foreground',
        success: 'border-green-200 bg-green-50 text-green-900 dark:border-green-900 dark:bg-green-950 dark:text-green-100',
        error: 'border-red-200 bg-red-50 text-red-900 dark:border-red-900 dark:bg-red-950 dark:text-red-100',
        warning: 'border-yellow-200 bg-yellow-50 text-yellow-900 dark:border-yellow-900 dark:bg-yellow-950 dark:text-yellow-100',
        info: 'border-blue-200 bg-blue-50 text-blue-900 dark:border-blue-900 dark:bg-blue-950 dark:text-blue-100',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

export interface ToastProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof toastVariants> {
  onClose?: () => void
  title?: string
  description?: string
}

const Toast = React.forwardRef<HTMLDivElement, ToastProps>(
  ({ className, variant, onClose, title, description, children, ...props }, ref) => {
    const icons = {
      default: Info,
      success: CheckCircle,
      error: AlertCircle,
      warning: AlertTriangle,
      info: Info,
    }
    const Icon = icons[variant || 'default']

    return (
      <div
        ref={ref}
        className={cn(toastVariants({ variant }), className)}
        {...props}
      >
        <div className="flex items-start gap-3">
          <Icon className="h-5 w-5 shrink-0 mt-0.5" />
          <div className="flex-1">
            {title && <div className="font-semibold">{title}</div>}
            {description && <div className="text-sm opacity-90">{description}</div>}
            {children}
          </div>
        </div>
        {onClose && (
          <button
            onClick={onClose}
            className="absolute right-2 top-2 rounded-md p-1 opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    )
  }
)
Toast.displayName = 'Toast'

// Toast Container
interface ToastContainerProps {
  children: React.ReactNode
  className?: string
}

const ToastContainer = React.forwardRef<HTMLDivElement, ToastContainerProps>(
  ({ className, children }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          'fixed top-4 right-4 z-[100] flex flex-col gap-2 w-full max-w-sm',
          className
        )}
      >
        {children}
      </div>
    )
  }
)
ToastContainer.displayName = 'ToastContainer'

// Toast Hook
interface ToastItem {
  id: string
  variant: 'default' | 'success' | 'error' | 'warning' | 'info'
  title?: string
  description?: string
  duration?: number
}

export function useToast() {
  const [toasts, setToasts] = React.useState<ToastItem[]>([])

  const addToast = React.useCallback((toast: Omit<ToastItem, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9)
    const newToast = { ...toast, id }
    setToasts((prev) => [...prev, newToast])

    if (toast.duration !== 0) {
      setTimeout(() => {
        removeToast(id)
      }, toast.duration || 5000)
    }

    return id
  }, [])

  const removeToast = React.useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const toast = React.useMemo(
    () => ({
      success: (title: string, description?: string, duration?: number) =>
        addToast({ variant: 'success', title, description, duration }),
      error: (title: string, description?: string, duration?: number) =>
        addToast({ variant: 'error', title, description, duration }),
      warning: (title: string, description?: string, duration?: number) =>
        addToast({ variant: 'warning', title, description, duration }),
      info: (title: string, description?: string, duration?: number) =>
        addToast({ variant: 'info', title, description, duration }),
    }),
    [addToast]
  )

  const ToastProvider = React.useCallback(
    () => (
      <ToastContainer>
        {toasts.map((t) => (
          <Toast
            key={t.id}
            variant={t.variant}
            title={t.title}
            description={t.description}
            onClose={() => removeToast(t.id)}
          />
        ))}
      </ToastContainer>
    ),
    [toasts, removeToast]
  )

  return { toast, ToastProvider, toasts, removeToast }
}

export { Toast, ToastContainer }
