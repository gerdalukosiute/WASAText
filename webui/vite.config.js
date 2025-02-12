import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig(({command, mode, ssrBuild}) => {
  const ret = {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    server: {
      host: true,
      port: 8080, // Changed to 8080 for frontend
    },
    define: {
      __VUE_OPTIONS_API__: false,
      __VUE_PROD_DEVTOOLS__: false,
      __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false,
      'process.env': {}
    }
  };
  ret.define = {
    // Do not modify this constant, it is used in the evaluation.
    "__API_URL__": JSON.stringify("http://localhost:3000"),
  };
  return ret;
})
