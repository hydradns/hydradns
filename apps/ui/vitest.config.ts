import { fileURLToPath } from "node:url"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vitest/config"

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
  },
  resolve: {
    alias: {
      // Mirror the "@/*" -> "./*" path alias from tsconfig.json so tests can
      // import app modules the same way as source files.
      "@": fileURLToPath(new URL("./", import.meta.url)),
    },
  },
})
