import type { ReactNode } from 'react'
import { Dialog } from './Dialog'

export interface DrawerProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  side?: 'left' | 'right'
  modal?: boolean
}

export function Drawer({ open, title, onClose, children, side = 'right', modal = true }: DrawerProps) {
  return (
    <Dialog open={open} title={title} onClose={onClose} modal={modal} className={`ui-drawer ui-drawer-${side}`}>
      {children}
    </Dialog>
  )
}
