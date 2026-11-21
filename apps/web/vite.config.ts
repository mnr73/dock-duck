import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { URL, fileURLToPath } from "node:url";
import Icons from "unplugin-icons/vite";
import { type Plugin, defineConfig } from "vite";

/** ~icons/* modules have no side effects; lets rollup drop unused re-exports from src/icons.ts. */
function iconsTreeShaking(): Plugin {
    return {
        name: "icons-tree-shaking",
        enforce: "post",
        transform(code, id) {
            if (id.includes("~icons/")) {
                return { code, map: null, moduleSideEffects: false };
            }
        },
    };
}

/** Backend for the dev proxy; override to run a second dev stack: DOCKDUCK_API=http://127.0.0.1:18100 pnpm dev */
const apiTarget = process.env.DOCKDUCK_API ?? "http://127.0.0.1:8080";

export default defineConfig({
    plugins: [
        vue(),
        tailwindcss(),
        Icons({ compiler: "vue3" }),
        iconsTreeShaking(),
    ],
    resolve: {
        alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
    },
    server: {
        port: 5173,
        // changeOrigin stays false on purpose: the backend refuses state-changing
        // requests whose Origin does not match the Host it sees (CSRF / DNS rebinding
        // guard), and a rewritten Host would make every dev request look cross-origin.
        proxy: {
            "/api": { target: apiTarget },
            "/ws": { target: apiTarget.replace("http", "ws"), ws: true },
        },
    },
    build: { outDir: "dist", emptyOutDir: true },
});
