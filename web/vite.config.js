import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { quasar, transformAssetUrls } from '@quasar/vite-plugin'

export default defineConfig({
  plugins: [vue({ template: { transformAssetUrls } }), quasar()],
  // embed.go serves dist/spa from inside the Go binary
  build: { outDir: 'dist/spa', emptyOutDir: true },
  server: {
    proxy: { '/api': 'http://127.0.0.1:8080', '/files': 'http://127.0.0.1:8080' }
  }
})
