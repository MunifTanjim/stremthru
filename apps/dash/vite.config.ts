import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import tsConfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  base: "/dash",
  plugins: [
    tsConfigPaths(),
    tanstackStart({
      customViteReactPlugin: true,
      spa: {
        enabled: true,
      },
    }),
    viteReact(),
    tailwindcss(),
  ],
  server: {
    hmr: {
      port: 3001,
    },
    port: 3000,
  },
});
