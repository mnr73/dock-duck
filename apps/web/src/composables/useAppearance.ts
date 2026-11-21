import { watchEffect } from "vue";

import { usePrefsStore } from "@/stores/prefs";
import { brandDataUrl } from "@/utils/brand";

import { useTheme } from "./useTheme";

/**
 * Applies the accent colour and row density to <html>. The accent overrides the
 * theme's four accent tokens; the readable text variant is darker on a light
 * background so it keeps its contrast.
 */
export function useAppearance() {
    const prefs = usePrefsStore();
    const theme = useTheme();

    watchEffect(() => {
        const root = document.documentElement;
        const c = prefs.accent;
        if (c) {
            const light = theme.state.value === "light";
            root.style.setProperty("--dd-accent", c);
            root.style.setProperty(
                "--dd-accent-soft",
                `color-mix(in srgb, ${c} ${light ? 16 : 12}%, transparent)`,
            );
            root.style.setProperty(
                "--dd-accent-text",
                light ? `color-mix(in srgb, ${c} 55%, black)` : c,
            );
        } else {
            for (const k of [
                "--dd-accent",
                "--dd-accent-soft",
                "--dd-accent-text",
            ])
                root.style.removeProperty(k);
        }
        root.dataset.density = prefs.density;

        // the tab icon follows the accent too
        const icon = document.querySelector<HTMLLinkElement>("link[rel=icon]");
        if (icon)
            icon.href = prefs.accent
                ? brandDataUrl(prefs.accent)
                : "/favicon.svg";
    });
}
