import { useStorage } from "@vueuse/core";
import { defineStore } from "pinia";

export type Density = "comfortable" | "compact";

/** Accent swatches from the design board; `null` keeps the theme's own amber. */
export const ACCENTS: { name: string; value: string | null }[] = [
    { name: "Amber", value: null },
    { name: "Periwinkle", value: "#8b91ff" },
    { name: "Sky", value: "#5ec4f0" },
    { name: "Mint", value: "#45c46b" },
    { name: "Coral", value: "#ff8a6b" },
];

/**
 * Per-browser preferences (localStorage). The server only keeps what the server
 * itself uses (scan depth, default context); how this browser looks and asks is
 * not shared between machines.
 */
export const usePrefsStore = defineStore("prefs", () => {
    const accent = useStorage<string | null>("dockduck.accent", null);
    const density = useStorage<Density>("dockduck.density", "comfortable");
    /** ask before `compose down` (volume removal always asks, in its own dialog) */
    const confirmDown = useStorage("dockduck.confirmDown", true);
    /** toasts for finished operations; failures are always shown */
    const doneToasts = useStorage("dockduck.doneToasts", true);

    return { accent, density, confirmDown, doneToasts };
});
