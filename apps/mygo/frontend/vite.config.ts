import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: 9245,
    strictPort: true,
  },
  resolve: {
    alias: { $lib: path.resolve(__dirname, "./src/lib") },
  },
  plugins: [tailwindcss(), svelte()],
});
