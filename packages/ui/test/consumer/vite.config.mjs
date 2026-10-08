import tailwind from '@tailwindcss/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [tailwind()],
  build: { license: { fileName: 'THIRD_PARTY_LICENSES.json' } },
})
