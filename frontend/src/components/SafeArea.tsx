import type { ReactNode } from 'react'

/** 统一处理 Android WebView 对 env(safe-area-inset-*) 支持不全的问题 */
export function SafeArea({
  children,
  edges = 'top bottom',
  className = '',
}: {
  children: ReactNode
  edges?: 'top' | 'bottom' | 'top bottom' | 'all'
  className?: string
}) {
  const parts = edges.split(' ')
  const style: React.CSSProperties = {}
  if (parts.includes('top')) style.paddingTop = 'var(--safe-top)'
  if (parts.includes('bottom')) style.paddingBottom = 'var(--safe-bottom)'
  if (edges === 'all') {
    style.paddingLeft = 'var(--safe-left)'
    style.paddingRight = 'var(--safe-right)'
  }
  return (
    <div className={className} style={style}>
      {children}
    </div>
  )
}

export function useSafeBottomStyle(): React.CSSProperties {
  return { paddingBottom: 'var(--safe-bottom)' }
}
