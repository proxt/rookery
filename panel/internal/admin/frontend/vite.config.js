import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// base must match the path the Go server mounts the built assets under
// (admin.RegisterRoutes strips "/admin/" before serving from the embedded
// dist/ dir, but the browser's own requests — and therefore every asset
// URL baked into index.html — go out as "/admin/...").
export default defineConfig({
  base: '/admin/',
  plugins: [svelte(), tailwindcss()],
  server: {
    proxy: {
      '/admin/api': 'http://127.0.0.1:8090',
      '/sub': 'http://127.0.0.1:8090',
    },
  },
})
