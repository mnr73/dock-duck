import { type Ref, computed, ref, watch } from "vue";

import { useIntervalFn } from "@vueuse/core";

import { api } from "@/api/client";
import type { ServiceInfo, Usage } from "@/api/types";

/**
 * Polls /projects/{id}/stats while the page is mounted and at least one
 * container runs, then folds container usage into per-service totals.
 * (The server caches `docker stats` for 3 s, so this is cheap.)
 */
export function useProjectUsage(
    projectId: Ref<string>,
    services: Ref<ServiceInfo[] | undefined>,
    intervalMs = 5000,
) {
    const byContainer = ref<Record<string, Usage>>({});
    const anyRunning = computed(() =>
        (services.value ?? []).some((s) =>
            s.containers.some((c) => c.state === "running"),
        ),
    );

    async function refresh() {
        if (!anyRunning.value) {
            byContainer.value = {};
            return;
        }
        try {
            byContainer.value =
                (await api.get<Record<string, Usage>>(
                    `/projects/${projectId.value}/stats`,
                )) ?? {};
        } catch {
            /* usage is optional: keep showing the previous values */
        }
    }

    const { pause, resume } = useIntervalFn(refresh, intervalMs, {
        immediate: false,
        immediateCallback: true,
    });
    watch(
        [projectId, anyRunning],
        () => {
            byContainer.value = {};
            if (anyRunning.value) resume();
            else pause();
        },
        { immediate: true },
    );

    /** service name → summed usage of its running containers */
    const byService = computed(() => {
        const out: Record<string, { cpu: number; mem: number } | undefined> =
            {};
        for (const s of services.value ?? []) {
            const us = s.containers
                .map((c) => byContainer.value[c.id])
                .filter((u): u is Usage => !!u);
            if (us.length)
                out[s.name] = {
                    cpu: us.reduce((n, u) => n + u.cpu_percent, 0),
                    mem: us.reduce((n, u) => n + u.mem_bytes, 0),
                };
        }
        return out;
    });

    return { byService, refresh };
}
