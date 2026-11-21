import { computed, ref } from "vue";

import { defineStore } from "pinia";

import { api, errorMessage } from "@/api/client";
import type { PortEntry } from "@/api/types";

export const usePortsStore = defineStore("ports", () => {
    const list = ref<PortEntry[]>([]);
    const loading = ref(false);
    const loaded = ref(false);
    const error = ref<string | null>(null);

    /** Host ports published more than once on the same protocol (a binding conflict to look at). */
    const duplicates = computed(() => {
        const seen = new Map<string, number>();
        for (const p of list.value) {
            const k = `${p.host_port}/${p.proto}`;
            seen.set(k, (seen.get(k) ?? 0) + 1);
        }
        return new Set([...seen].filter(([, n]) => n > 1).map(([k]) => k));
    });

    async function load(quiet = false) {
        if (!quiet) loading.value = true;
        try {
            list.value = (await api.get<PortEntry[] | null>("/ports")) ?? [];
            error.value = null;
        } catch (e) {
            error.value = errorMessage(e);
        } finally {
            loading.value = false;
            loaded.value = true;
        }
    }

    return { list, loading, loaded, error, duplicates, load };
});
