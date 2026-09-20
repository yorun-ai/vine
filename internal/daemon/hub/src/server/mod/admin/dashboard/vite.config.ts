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
    rolldownOptions: {
      output: {
        assetFileNames: (asset) => {
          const isFont = asset.names.some((name) => /\.(woff2?|ttf|otf|eot)$/i.test(name))
          return isFont ? 'fonts/[name]-[hash][extname]' : 'assets/[name]-[hash][extname]'
        },
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
