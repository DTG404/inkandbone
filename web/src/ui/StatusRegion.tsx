export interface StatusRegionProps {
  message?: string | null
  priority?: 'polite' | 'assertive'
  className?: string
}

export function StatusRegion({ message, priority = 'polite', className = '' }: StatusRegionProps) {
  return (
    <div
      className={`ui-status-region ${className}`.trim()}
      role={priority === 'assertive' ? 'alert' : 'status'}
      aria-live={priority}
      aria-atomic="true"
    >
      {message}
    </div>
  )
}
