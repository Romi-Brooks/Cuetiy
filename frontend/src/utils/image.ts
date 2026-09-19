/** 客户端压缩头像，避免超过后端 1MB 限制 */
export async function compressImageToBlob(
  file: File,
  opts: {
    maxSizeBytes?: number
    maxEdge?: number
    quality?: number
    mimeType?: string
  } = {},
): Promise<File> {
  const maxSizeBytes = opts.maxSizeBytes ?? 1024 * 1024
  const maxEdge = opts.maxEdge ?? 1024
  let quality = opts.quality ?? 0.85
  const mimeType = opts.mimeType ?? 'image/jpeg'

  if (!file.type.startsWith('image/')) {
    return file
  }
  // 已经够小且不是超大图，可直接传
  if (file.size <= maxSizeBytes && file.type !== 'image/png') {
    // PNG 头像仍可能很大，统一转 JPEG 更稳
    if (file.size <= maxSizeBytes) return file
  }

  const bitmap = await loadImageBitmap(file)
  try {
    const { width, height } = bitmap
    const scale = Math.min(1, maxEdge / Math.max(width, height))
    const w = Math.max(1, Math.round(width * scale))
    const h = Math.max(1, Math.round(height * scale))

    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d')
    if (!ctx) return file
    ctx.drawImage(bitmap, 0, 0, w, h)

    let blob = await canvasToBlob(canvas, mimeType, quality)
    // 逐步降质
    while (blob && blob.size > maxSizeBytes && quality > 0.4) {
      quality -= 0.12
      blob = await canvasToBlob(canvas, mimeType, quality)
    }
    if (!blob) return file

    // 还是太大再缩边
    let edge = maxEdge
    while (blob && blob.size > maxSizeBytes && edge > 256) {
      edge = Math.round(edge * 0.75)
      const s = Math.min(1, edge / Math.max(width, height))
      canvas.width = Math.max(1, Math.round(width * s))
      canvas.height = Math.max(1, Math.round(height * s))
      ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
      const next = await canvasToBlob(canvas, mimeType, Math.max(quality, 0.55))
      if (next) blob = next
      else break
    }

    if (!blob) return file
    if (blob.size >= file.size && file.size <= maxSizeBytes) {
      return file
    }

    const ext = mimeType === 'image/png' ? 'png' : 'jpg'
    const base = file.name.replace(/\.[^.]+$/, '') || 'avatar'
    return new File([blob], `${base}.${ext}`, { type: mimeType, lastModified: Date.now() })
  } finally {
    if ('close' in bitmap && typeof bitmap.close === 'function') {
      bitmap.close()
    }
  }
}

async function loadImageBitmap(file: File): Promise<ImageBitmap> {
  if (typeof createImageBitmap === 'function') {
    return createImageBitmap(file)
  }
  // 兜底：部分 WebView
  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image()
      el.onload = () => resolve(el)
      el.onerror = () => reject(new Error('image load failed'))
      el.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = img.naturalWidth
    canvas.height = img.naturalHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('no 2d context')
    ctx.drawImage(img, 0, 0)
    return await createImageBitmap(canvas)
  } finally {
    URL.revokeObjectURL(url)
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => {
    canvas.toBlob(resolve, type, quality)
  })
}
