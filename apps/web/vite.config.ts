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
        proxy: {
            "/api": "http://127.0.0.1:8080",
            "/ws": { target: "ws://127.0.0.1:8080", ws: true },
        },
    },
    build: { outDir: "dist", emptyOutDir: true },
});
