import { computed, watch } from "vue";

import { useDebounceFn, useWebSocket } from "@vueuse/core";

import type { ContainerEvent } from "@/api/types";
import { useAuthStore } from "@/stores/auth";
import { useContextsStore } from "@/stores/contexts";
import { useProjectsStore } from "@/stores/projects";

/**
 * Live container events (WebSocket). On every event the project list and any
 * already-loaded project detail are refreshed (debounced). Reconnects forever
 * and re-opens when the selected Docker context changes.
 */
export function useLiveEvents() {
    const projects = useProjectsStore();
    const auth = useAuthStore();
    const contexts = useContextsStore();

    const touched = new Set<string>();
    const refresh = useDebounceFn(async () => {
        const names = new Set(touched);
        touched.clear();
        await projects.load(true);
        for (const p of projects.projects) {
            if (names.has(p.name) && projects.details[p.id]) {
                projects.loadDetail(p.id, true).catch(() => {});
            }
        }
    }, 400);

    const proto = location.protocol === "https:" ? "wss" : "ws";
    const url = `${proto}://${location.host}/ws/v1/events`;

    const { status, open, close } = useWebSocket(url, {
        autoReconnect: { retries: Number.POSITIVE_INFINITY, delay: 2000 },
        // the server closes this socket when the login session ends (logout elsewhere,
        // password change); asking /auth/status shows the login screen right away
        // instead of waiting for the next click to hit a 401
        onDisconnected: () => void auth.load(),
        onMessage(_ws, e) {
            try {
                const ev = JSON.parse(e.data) as ContainerEvent;
                touched.add(ev.project);
                refresh();
            } catch {
                /* ignore malformed frame */
            }
        },
    });

    watch(
        () => contexts.selected?.name,
        (now, before) => {
            if (before && now !== before) {
                close();
                open();
            }
        },
    );

    return { status, live: computed(() => status.value === "OPEN") };
}
