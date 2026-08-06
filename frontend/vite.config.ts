import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    // 允许 cloudflared / 临时隧道域名访问（本地开发）
    allowedHosts: true,
    proxy: {
      '/api': {
        target: 'http://localhost:18080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            const host = req.headers.host;
            if (host) {
              proxyReq.setHeader('X-Forwarded-Host', host);
            }
            proxyReq.setHeader('X-Forwarded-Proto', 'http');
          });
        },
      },
    },
  },
})
