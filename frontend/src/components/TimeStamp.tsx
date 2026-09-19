import { formatTime } from '../utils'

export function TimeStamp({ time }: { time: string }) {
  if (!time) return null
  return (
    <div className="flex justify-center mb-4">
      <span className="text-xs text-gray-400 bg-gray-100 dark:bg-gray-800 px-3 py-1 rounded-full">
        {formatTime(time)}
      </span>
    </div>
  )
}
