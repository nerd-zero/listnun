import vue from '@vitejs/plugin-vue';
import { defineConfig, loadEnv } from 'vite';
import { fileURLToPath, URL } from 'node:url';

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  return {
    plugins: [vue()],
    base: '/admin',
    mode,
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    build: {
      assetsDir: 'static',
    },
    server: {
      allowedHosts: true,
      port: parseInt(env.LISTMONK_FRONTEND_PORT, 10) || 8080,
      proxy: {
        '^/$': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
        '^/(api|webhooks|subscription|public|health)': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
        '^/admin/login': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
        '^/(admin\\/custom\\.(css|js))': {
          target: env.LISTMONK_API_URL || 'http://127.0.0.1:9000',
        },
      },
    },
  };
});
