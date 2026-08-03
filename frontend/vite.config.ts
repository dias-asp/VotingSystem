import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const apiTarget = process.env.VITE_API_PROXY ?? "http://localhost:8080";
const authTarget = process.env.VITE_AUTH_PROXY ?? "http://localhost:8083";
const auditTarget = process.env.VITE_AUDIT_PROXY ?? "http://localhost:8082";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    host: "0.0.0.0",
    proxy: {
      "/api/auth": {
        target: authTarget,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/auth/, ""),
      },
      "/api/audit": {
        target: auditTarget,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/audit/, ""),
      },
      "/api": {
        target: apiTarget,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ""),
      },
    },
  },
});
