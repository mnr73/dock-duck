import { computed, watch } from "vue";

import { useDebounceFn, useWebSocket } from "@vueuse/core";

import type { ContainerEvent } from "@/api/types";
import { useContextsStore } from "@/stores/contexts";
import { useProjectsStore } from "@/stores/projects";

/**
 * Live container events (WebSocket). On every event the project list and any
 * already-loaded project detail are refreshed (debounced). Reconnects forever
 * and re-opens when the selected Docker context changes.
 */
export function useLiveEvents() {
    const projects = useProjectsStore();
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
