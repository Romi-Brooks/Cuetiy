export function EmojiPicker({
  visible,
  onClose,
}: {
  visible: boolean
  onClose: () => void
}) {
  if (!visible) return null
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-80 p-6">
        <h3 className="text-lg font-semibold text-center text-gray-800 dark:text-gray-200 mb-4">
          选择表情
        </h3>
        <p className="text-sm text-gray-500 dark:text-gray-400 text-center mb-6">
          表情功能正在开发中，后续版本将支持
        </p>
        <div className="flex justify-center">
          <button className="btn-primary px-8" onClick={onClose}>
            关闭
          </button>
        </div>
      </div>
    </div>
  )
}
