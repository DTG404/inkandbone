import type { ButtonHTMLAttributes, ReactNode } from 'react'

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'aria-label' | 'children'> {
  label: string
  icon: ReactNode
}

export function IconButton({ label, icon, className = '', type = 'button', ...props }: IconButtonProps) {
  return (
    <button
      {...props}
      type={type}
      aria-label={label}
      className={`ui-icon-button ${className}`.trim()}
    >
      <span aria-hidden="true">{icon}</span>
    </button>
  )
}
