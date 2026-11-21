import { type Ref, ref } from "vue";

import { useRouter } from "vue-router";

import { useEventListener } from "@vueuse/core";

/** True when the key press is meant for a text field, the code editor or the terminal. */
export function isTyping(e: KeyboardEvent): boolean {
    const el = e.target;
    if (!(el instanceof HTMLElement)) return false;
    return (
        el.isContentEditable ||
        el.matches("input, textarea, select") ||
        !!el.closest(".cm-editor, .xterm")
    );
}

export interface Shortcuts {
    palette: Ref<boolean>;
    help: Ref<boolean>;
}

/**
 * App-wide shortcuts. Ctrl/Cmd+K works everywhere (it is not a printable key a
 * terminal needs), but single keys never fire while typing and never with a
 * modifier, so they cannot swallow input.
 *
 *   Ctrl/Cmd+K   command palette        ?        this list
 *   g then p/o/r/s   Projects / pOrts / Registries / Settings  (like GitHub)
 *   [            collapse or expand the sidebar
 */
export function useShortcuts(toggleSidebar: () => void): Shortcuts {
    const router = useRouter();
    const palette = ref(false);
    const help = ref(false);
    let pendingG = 0;

    useEventListener(window, "keydown", (e: KeyboardEvent) => {
        if (
            (e.ctrlKey || e.metaKey) &&
            !e.altKey &&
            e.key.toLowerCase() === "k"
        ) {
            e.preventDefault();
            help.value = false;
            palette.value = !palette.value;
            return;
        }
        if (e.ctrlKey || e.metaKey || e.altKey || isTyping(e)) return;
        if (palette.value || help.value) return;

        const now = Date.now();
        if (pendingG && now - pendingG < 1200) {
            pendingG = 0;
            const to = {
                p: "/",
                o: "/ports",
                r: "/registries",
                s: "/settings",
            }[e.key.toLowerCase() as "p" | "o" | "r" | "s"];
            if (to) {
                e.preventDefault();
                void router.push(to);
            }
            return;
        }
        if (e.key === "g") pendingG = now;
        else if (e.key === "?") {
            e.preventDefault();
            help.value = true;
        } else if (e.key === "[") {
            e.preventDefault();
            toggleSidebar();
        }
    });

    return { palette, help };
}
