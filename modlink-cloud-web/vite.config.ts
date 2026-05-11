import type { ServerResponse } from 'http'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.VITE_PROXY_TARGET || 'http://106.13.108.88:8100'

  return {
    plugins: [vue()],
    server: {
      port: 5173,
      proxy: {
        '/mlk': {
          target,
          changeOrigin: true,
          configure: (proxy) => {
            proxy.on('error', (err, _req, res) => {
              const r = res as ServerResponse | undefined
              if (r && !r.headersSent) {
                r.writeHead(502, { 'Content-Type': 'application/json; charset=utf-8' })
                r.end(
                  JSON.stringify({
                    code: 50201,
                    message: 'PROXY_TARGET_UNREACHABLE',
                    detail: {
                      target,
                      error: err instanceof Error ? err.message : String(err),
                      hint: '默认联调 http://106.13.108.88:8100；本机 Platform：go run ./cmd/platform 且 VITE_PROXY_TARGET=http://127.0.0.1:8081',
                    },
                  }),
                )
              }
            })
          },
        },
      },
    },
  }
})
