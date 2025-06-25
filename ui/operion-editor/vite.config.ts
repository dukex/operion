import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  return {
    resolve: {
      alias: {
        "@": "/src",
      },
    },
    plugins: [react(), tailwindcss()],
    define: {
      "process.env.API_BASE_URL": JSON.stringify(env.API_BASE_URL),
    },
  };
});
