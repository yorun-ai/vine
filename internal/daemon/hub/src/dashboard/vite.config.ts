import { defineConfig } from 'vite'
import { devtools } from '@tanstack/devtools-vite'
import viteReact from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const dashboardDevPort = 7098

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  server: {
    port: dashboardDevPort,
    hmr: {
      protocol: 'ws',
      host: 'localhost',
      port: dashboardDevPort,
      clientPort: dashboardDevPort,
    },
  },
  build: {
    chunkSizeWarningLimit: 1000,
    license: {
      fileName: 'THIRD_PARTY_LICENSES.md',
    },
    rolldownOptions: {
      output: {
        postBanner: '/* Third-party licenses: /THIRD_PARTY_LICENSES.md */',
      },
    },
  },
  plugins: [
    devtools(),
    tailwindcss(),
    viteReact(),
  ],
})

export default config
