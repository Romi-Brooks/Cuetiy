import { useEffect } from 'react'

export function Toast({
  message,
  onDone,
}: {
  message: string | null
  onDone: () => void
}) {
  useEffect(() => {
    if (!message) return
    const t = setTimeout(onDone, 2000)
    return () => clearTimeout(t)
  }, [message, onDone])

  if (!message) return null

  return (
    <div
      className="fixed left-1/2 -translate-x-1/2 z-[60] bg-black/80 text-white text-sm px-4 py-2 rounded-full shadow-lg pointer-events-none"
      style={{ top: 'calc(env(safe-area-inset-top, 0px) + 16px)' }}
    >
      {message}
    </div>
  )
}
