import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 本地联调: VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080
// 默认同源/空，由登录页运行时配置服务器
const baseUrl = process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: baseUrl,
        changeOrigin: true,
        ws: true,
      },
      '/static': {
        target: baseUrl,
        changeOrigin: true,
      },
      '/storage': {
        target: baseUrl,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        entryFileNames: 'js/[name]-[hash].js',
        chunkFileNames: 'js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          const name = assetInfo.name || ''
          if (name.endsWith('.css')) return 'css/[name]-[hash][extname]'
          if (/\.(png|jpe?g|gif|svg|webp|avif|ico)$/i.test(name)) return 'img/[name]-[hash][extname]'
          if (/\.(woff2?|eot|ttf|otf)$/i.test(name)) return 'fonts/[name]-[hash][extname]'
          if (/\.(mp3|wav|ogg|mp4|webm)$/i.test(name)) return 'media/[name]-[hash][extname]'
          return 'assets/[name]-[hash][extname]'
        },
      },
    },
  },
})
