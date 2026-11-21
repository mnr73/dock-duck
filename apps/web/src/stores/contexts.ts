import { computed, ref } from "vue";

import { defineStore } from "pinia";

import { api, errorMessage } from "@/api/client";
import type { ContextInfo } from "@/api/types";

export const useContextsStore = defineStore("contexts", () => {
    const list = ref<ContextInfo[]>([]);
    const loading = ref(false);
    const loaded = ref(false);
    /** The list call itself failed (docker CLI missing / server error). */
    const error = ref<string | null>(null);

    const selected = computed(() => list.value.find((c) => c.selected) ?? null);
    /** Selected context is known and its engine does not answer. */
    const disconnected = computed(
        () => !!error.value || (!!selected.value && !selected.value.available),
    );

    async function load() {
        loading.value = true;
        try {
            list.value = (await api.get<ContextInfo[]>("/contexts")) ?? [];
            error.value = null;
        } catch (e) {
            error.value = errorMessage(e);
        } finally {
            loading.value = false;
            loaded.value = true;
        }
    }

    /** Switch the app-level context; returns once the server has persisted it. */
    async function select(name: string) {
        await api.put("/contexts/selected", { name });
        await load();
    }

    return {
        list,
        loading,
        loaded,
        error,
        selected,
        disconnected,
        load,
        select,
    };
});
